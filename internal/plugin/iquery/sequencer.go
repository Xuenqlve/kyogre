package iquery

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/xuenqlve/common/schema_store"
	"github.com/xuenqlve/kyogre/pkg/tool/range_pool"
)

func MakeSequenceSpec(schema schema_store.SchemaKey, columns []ColumnParam) SequenceSpec {
	spec := SequenceSpec{
		Schema: schema,
		Fields: columns,
	}
	if len(columns) == 1 {
		column := columns[0]
		if isNumericType(strings.ToLower(strings.TrimSpace(column.Type))) {
			spec.Field = column.Column
		}
	}
	return spec
}

// SequenceSpec defines a unique constraint dimension to manage.
// For the "90% case", it is a single int primary key column.
type SequenceSpec struct {
	Schema schema_store.SchemaKey
	// Field is the legacy single-column unique field.
	// Prefer Fields for multi-column unique constraints.
	Field string
	// Fields are the columns participating in the unique constraint (ordered).
	Fields []ColumnParam
}

func (spec SequenceSpec) Key() string {
	if spec.Schema == nil {
		return ""
	}
	cols := make([]string, 0, len(spec.Fields)+1)
	if spec.Field != "" {
		cols = append(cols, spec.Field)
	}
	for _, f := range spec.Fields {
		if f.Column != "" {
			cols = append(cols, f.Column)
		}
	}
	return fmt.Sprintf("%s:%v", spec.Schema.UniqueID(), cols)
}

// Sequencer allocates id ranges for unique constraints.
// It guarantees per-key serialized allocation inside a single process.
type Sequencer struct {
	pipeline   string
	freeLookup Lookup
	liveLookup Lookup
	cfg        *SequencerConfig

	stateMu sync.RWMutex
	states  map[string]sequenceState

	keyMuMu sync.Mutex
	keyMu   map[string]*sync.Mutex
}

func NewSequencer() (*Sequencer, error) {
	return &Sequencer{
		cfg:    &SequencerConfig{},
		states: make(map[string]sequenceState),
		keyMu:  make(map[string]*sync.Mutex),
	}, nil
}

func (s *Sequencer) Configure(pipeline string, opts ...SequencerOptions) error {
	s.pipeline = pipeline
	for _, opt := range opts {
		opt(s.cfg)
	}
	if err := s.cfg.ValidateAndSetDefault(); err != nil {
		return err
	}
	if s.freeLookup == nil {
		lookup, err := GetIQueryModule(Memory)
		if err != nil {
			return err
		}
		s.freeLookup = lookup
	}
	if err := s.freeLookup.Configure(pipeline, s.cfg.FreeLookupConfig); err != nil {
		return err
	}
	if s.liveLookup == nil {
		s.liveLookup = s.freeLookup
	}
	return nil
}

func (s *Sequencer) BindLookup(lookup Lookup) { s.liveLookup = lookup }

// ReserveInsert reserves a continuous range for INSERT operations from the free partition.
func (s *Sequencer) ReserveInsert(ctx context.Context, spec SequenceSpec, need int64) (Provider, error) {
	key := spec.Key()
	if key == "" {
		return nil, fmt.Errorf("invalid spec")
	}
	mu := s.keyLock(key)
	mu.Lock()
	defer mu.Unlock()

	st, err := s.initState(ctx, spec)
	if err != nil {
		return nil, err
	}
	return st.reserveInsertProvider(ctx, need)
}

// ReserveUpdate picks an existing live range for UPDATE.
func (s *Sequencer) ReserveUpdate(ctx context.Context, spec SequenceSpec, need int64) (Provider, error) {
	key := spec.Key()
	if key == "" {
		return nil, fmt.Errorf("invalid spec")
	}
	mu := s.keyLock(key)
	mu.Lock()
	defer mu.Unlock()

	st, err := s.initState(ctx, spec)
	if err != nil {
		return nil, err
	}
	return st.reserveUpdateProvider(ctx, need)
}

// ReserveDelete reserves an existing live range for DELETE by consuming from the live partition.
// It does not return the range to the free partition; reuse depends on the lookup refill strategy.
func (s *Sequencer) ReserveDelete(ctx context.Context, spec SequenceSpec, need int64) (Provider, error) {
	key := spec.Key()
	if key == "" {
		return nil, fmt.Errorf("invalid spec")
	}
	mu := s.keyLock(key)
	mu.Lock()
	defer mu.Unlock()

	st, err := s.initState(ctx, spec)
	if err != nil {
		return nil, err
	}
	return st.reserveDeleteProvider(ctx, need)
}

func (s *Sequencer) keyLock(key string) *sync.Mutex {
	s.keyMuMu.Lock()
	defer s.keyMuMu.Unlock()
	if mu, ok := s.keyMu[key]; ok {
		return mu
	}
	mu := &sync.Mutex{}
	s.keyMu[key] = mu
	return mu
}

func (s *Sequencer) getState(key string) (sequenceState, bool) {
	s.stateMu.RLock()
	defer s.stateMu.RUnlock()
	st, ok := s.states[key]
	return st, ok
}

func (s *Sequencer) setState(key string, st sequenceState) {
	s.stateMu.Lock()
	defer s.stateMu.Unlock()
	s.states[key] = st
}

func (s *Sequencer) lookupForPartition(partition string) Lookup {
	if partition == range_pool.RangePoolFreeName {
		return s.freeLookup
	}
	if s.liveLookup != nil {
		return s.liveLookup
	}
	return s.freeLookup
}

// initState loads initial bounds from lookup for this spec.
func (s *Sequencer) initState(ctx context.Context, spec SequenceSpec) (sequenceState, error) {
	key := spec.Key()
	if key == "" {
		return nil, fmt.Errorf("invalid spec key")
	}
	if st, ok := s.getState(key); ok {
		return st, nil
	}

	params := spec.Fields
	if len(params) == 0 && spec.Field != "" {
		params = []ColumnParam{{Column: spec.Field}}
	}
	if len(params) == 0 {
		return nil, fmt.Errorf("invalid spec: empty fields")
	}

	// Strategy split:
	// - Single numeric field: use range pool
	// - Otherwise: use AliveSet (ScanValues)
	if len(params) != 1 || !isNumericType(params[0].Type) {
		st := newTupleSequenceState(spec, sequenceLookups{free: s.freeLookup, live: s.liveLookup},
			newAliveSet(params, s.cfg.AliveSetCapacity, s.cfg.AliveSetBatchSize),
			newAliveSet(params, s.cfg.AliveSetCapacity, s.cfg.AliveSetBatchSize),
		)
		s.setState(key, st)
		return st, nil
	}

	pool, err := range_pool.NewRangePool(
		range_pool.WithRefill(
			func(partition string, req range_pool.RefillRequest) (bool, range_pool.IntRange, error) {
				return s.lookupWindow(spec, params, partition, req.Need, req.WindowEnd)
			},
		),
	)
	if err != nil {
		return nil, err
	}
	st := newNumericSequenceState(spec, key, s.cfg, pool)
	s.setState(key, st)
	return st, nil
}

func (s *Sequencer) lookupWindow(spec SequenceSpec, params []ColumnParam, partition string, need int64, windowEnd int64) (bool, range_pool.IntRange, error) {
	lookup := s.lookupForPartition(partition)
	if lookup == nil {
		return false, range_pool.IntRange{}, fmt.Errorf("sequencer lookup is nil")
	}
	if need <= 0 {
		need = 1
	}
	if partition == range_pool.RangePoolFreeName && s.cfg.InsertWindowSize > 0 && need < s.cfg.InsertWindowSize {
		need = s.cfg.InsertWindowSize
	}
	res, err := lookup.LookupRange(context.Background(), RangeRequest{
		Schema:  spec.Schema,
		Columns: params,
		Need:    need,
		Cursor:  windowEnd,
	})
	if err != nil {
		return false, range_pool.IntRange{}, err
	}
	if !res.Window.Valid() {
		return false, range_pool.IntRange{}, fmt.Errorf("lookup returned invalid window for %s", spec.Key())
	}
	return res.EnableLoop, res.Window, nil
}

func (s *Sequencer) Close() error {
	if s.liveLookup != nil {
		if err := s.liveLookup.Close(); err != nil {
			return err
		}
	}
	if s.freeLookup != nil && s.freeLookup != s.liveLookup {
		if err := s.freeLookup.Close(); err != nil {
			return err
		}
	}
	return nil
}
