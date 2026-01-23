package iquery

import (
	"context"
	"fmt"
	"sync"

	"github.com/xuenqlve/common/schema_store"
	"github.com/xuenqlve/kyogre/pkg/tool/range_pool"
)

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
	states  map[string]*sequenceState

	keyMuMu sync.Mutex
	keyMu   map[string]*sync.Mutex
}

type sequenceState struct {
	spec SequenceSpec

	mu   sync.Mutex
	pool *range_pool.RangePool

	aliveFree *aliveSet
	aliveLive *aliveSet
}

func (st *sequenceState) specField() string {
	if st.spec.Field != "" {
		return st.spec.Field
	}
	if len(st.spec.Fields) > 0 {
		return st.spec.Fields[0].Column
	}
	return ""
}

func NewSequencer() (*Sequencer, error) {
	return &Sequencer{
		cfg:    &SequencerConfig{},
		states: make(map[string]*sequenceState),
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

func (s *Sequencer) getState(key string) (*sequenceState, bool) {
	s.stateMu.RLock()
	defer s.stateMu.RUnlock()
	st, ok := s.states[key]
	return st, ok
}

func (s *Sequencer) setState(key string, st *sequenceState) {
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
func (s *Sequencer) initState(ctx context.Context, spec SequenceSpec) (*sequenceState, error) {
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
		st := &sequenceState{
			spec:      spec,
			aliveFree: newAliveSet(params, s.cfg.AliveSetCapacity, s.cfg.AliveSetBatchSize),
			aliveLive: newAliveSet(params, s.cfg.AliveSetCapacity, s.cfg.AliveSetBatchSize),
		}
		s.setState(key, st)
		return st, nil
	}

	st := &sequenceState{
		spec: spec,
	}
	pool, err := range_pool.NewRangePool(range_pool.WithRefill(func(partition string, req range_pool.RefillRequest) (bool, range_pool.IntRange, error) {
		return s.lookupWindow(st, params, partition, req.Need, req.WindowEnd)
	}))
	if err != nil {
		return nil, err
	}
	st.pool = pool
	s.setState(key, st)
	return st, nil
}

func (s *Sequencer) lookupWindow(st *sequenceState, params []ColumnParam, partition string, need int64, windowEnd int64) (bool, range_pool.IntRange, error) {
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
	res, err := lookup.LookupRange(context.Background(), Request{
		Schema:  st.spec.Schema,
		Columns: params,
		Need:    need,
		Cursor:  windowEnd,
	})
	if err != nil {
		return false, range_pool.IntRange{}, err
	}
	if !res.Window.Valid() {
		return false, range_pool.IntRange{}, fmt.Errorf("lookup returned invalid window for %s", st.spec.Key())
	}
	return res.EnableLoop, res.Window, nil
}

// ReserveInsert reserves a continuous range for INSERT operations from the free partition.
func (s *Sequencer) ReserveInsert(ctx context.Context, spec SequenceSpec, need int64) (*SequenceConfig, error) {
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
	st.mu.Lock()
	defer st.mu.Unlock()

	if st.aliveFree != nil {
		return nil, fmt.Errorf("ReserveInsert not supported for AliveSet strategy (use ReserveInsertProvider)")
	}

	size, err := normalizeSize(need, s.cfg.AllowedSizes, s.cfg.StrictSizes)
	if err != nil {
		return nil, err
	}

	if r, ok := st.pool.ReserveInsert(size); ok {
		return &SequenceConfig{
			Key:          key,
			Field:        st.specField(),
			StartValue:   r.Start,
			EndValue:     r.End,
			CurrentValue: r.Start,
			Step:         1,
			Width:        r.Len(),
			Schema:       st.spec.Schema,
		}, nil
	}
	return nil, fmt.Errorf("no free range available for insert: %s", key)
}

func (s *Sequencer) ReserveInsertProvider(ctx context.Context, spec SequenceSpec, need int64) (RowProvider, *SequenceConfig, error) {
	key := spec.Key()
	if key == "" {
		return nil, nil, fmt.Errorf("invalid spec")
	}
	mu := s.keyLock(key)
	mu.Lock()
	defer mu.Unlock()

	st, err := s.initState(ctx, spec)
	if err != nil {
		return nil, nil, err
	}
	st.mu.Lock()
	defer st.mu.Unlock()

	// Numeric: range provider
	if st.pool != nil {
		var size int64
		size, err = normalizeSize(need, s.cfg.AllowedSizes, s.cfg.StrictSizes)
		if err != nil {
			return nil, nil, err
		}

		if r, ok := st.pool.ReserveInsert(size); ok {
			cfg := &SequenceConfig{
				Key:          key,
				Field:        st.specField(),
				StartValue:   r.Start,
				EndValue:     r.End,
				CurrentValue: r.Start,
				Step:         1,
				Width:        r.Len(),
				Schema:       st.spec.Schema,
			}
			return newRangeProvider(cfg.Field, cfg.StartValue, cfg.EndValue, cfg.Step), cfg, nil
		}
		return nil, nil, fmt.Errorf("no free range available for insert: %s", key)
	}

	// AliveSet: tuple provider -> map[column]any for IN batch usage
	rows, err := st.aliveFree.take(ctx, s.freeLookup, spec.Schema, int(need))
	if err != nil {
		return nil, nil, err
	}
	cols := make([]string, 0, len(st.aliveFree.columns))
	for _, c := range st.aliveFree.columns {
		cols = append(cols, c.Column)
	}
	tp, err := newTupleProvider(cols, rows)
	if err != nil {
		return nil, nil, err
	}
	return tp, nil, nil
}

// ReserveUpdate picks an existing live range for UPDATE.
func (s *Sequencer) ReserveUpdate(ctx context.Context, spec SequenceSpec, need int64) (*SequenceConfig, error) {
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
	st.mu.Lock()
	defer st.mu.Unlock()

	if st.aliveLive != nil {
		return nil, fmt.Errorf("ReserveUpdate not supported for AliveSet strategy (use ReserveUpdateProvider)")
	}

	size, err := normalizeSize(need, s.cfg.AllowedSizes, s.cfg.StrictSizes)
	if err != nil {
		return nil, err
	}
	r, ok := st.pool.ReserveUpdate(size)
	if !ok {
		return nil, fmt.Errorf("no live range available for update: %s", key)
	}
	return &SequenceConfig{
		Key:          key,
		Field:        st.specField(),
		StartValue:   r.Start,
		EndValue:     r.End,
		CurrentValue: r.Start,
		Step:         1,
		Width:        r.Len(),
		Schema:       st.spec.Schema,
	}, nil
}

func (s *Sequencer) ReserveUpdateProvider(ctx context.Context, spec SequenceSpec, need int64) (RowProvider, *SequenceConfig, error) {
	key := spec.Key()
	if key == "" {
		return nil, nil, fmt.Errorf("invalid spec")
	}
	mu := s.keyLock(key)
	mu.Lock()
	defer mu.Unlock()

	st, err := s.initState(ctx, spec)
	if err != nil {
		return nil, nil, err
	}
	st.mu.Lock()
	defer st.mu.Unlock()

	if st.pool != nil {
		size, err := normalizeSize(need, s.cfg.AllowedSizes, s.cfg.StrictSizes)
		if err != nil {
			return nil, nil, err
		}
		r, ok := st.pool.ReserveUpdate(size)
		if !ok {
			return nil, nil, fmt.Errorf("no live range available for update: %s", key)
		}
		cfg := &SequenceConfig{
			Key:          key,
			Field:        st.specField(),
			StartValue:   r.Start,
			EndValue:     r.End,
			CurrentValue: r.Start,
			Step:         1,
			Width:        r.Len(),
			Schema:       st.spec.Schema,
		}
		return newRangeProvider(cfg.Field, cfg.StartValue, cfg.EndValue, cfg.Step), cfg, nil
	}

	rows, err := st.aliveLive.take(ctx, s.liveLookup, spec.Schema, int(need))
	if err != nil {
		return nil, nil, err
	}
	cols := make([]string, 0, len(st.aliveLive.columns))
	for _, c := range st.aliveLive.columns {
		cols = append(cols, c.Column)
	}
	tp, err := newTupleProvider(cols, rows)
	if err != nil {
		return nil, nil, err
	}
	return tp, nil, nil
}

// ReserveDelete reserves an existing live range for DELETE by consuming from the live partition.
// It does not return the range to the free partition; reuse depends on the lookup refill strategy.
func (s *Sequencer) ReserveDelete(ctx context.Context, spec SequenceSpec, need int64) (*SequenceConfig, error) {
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
	st.mu.Lock()
	defer st.mu.Unlock()

	if st.aliveLive != nil {
		return nil, fmt.Errorf("ReserveDelete not supported for AliveSet strategy (use ReserveDeleteProvider)")
	}

	size, err := normalizeSize(need, s.cfg.AllowedSizes, s.cfg.StrictSizes)
	if err != nil {
		return nil, err
	}
	r, ok := st.pool.ReserveDelete(size)
	if !ok {
		return nil, fmt.Errorf("no live range available for delete: %s", key)
	}
	return &SequenceConfig{
		Key:          key,
		Field:        st.specField(),
		StartValue:   r.Start,
		EndValue:     r.End,
		CurrentValue: r.Start,
		Step:         1,
		Width:        r.Len(),
		Schema:       st.spec.Schema,
	}, nil
}

func (s *Sequencer) ReserveDeleteProvider(ctx context.Context, spec SequenceSpec, need int64) (RowProvider, *SequenceConfig, error) {
	key := spec.Key()
	if key == "" {
		return nil, nil, fmt.Errorf("invalid spec")
	}
	mu := s.keyLock(key)
	mu.Lock()
	defer mu.Unlock()

	st, err := s.initState(ctx, spec)
	if err != nil {
		return nil, nil, err
	}
	st.mu.Lock()
	defer st.mu.Unlock()

	if st.pool != nil {
		size, err := normalizeSize(need, s.cfg.AllowedSizes, s.cfg.StrictSizes)
		if err != nil {
			return nil, nil, err
		}
		r, ok := st.pool.ReserveDelete(size)
		if !ok {
			return nil, nil, fmt.Errorf("no live range available for delete: %s", key)
		}
		cfg := &SequenceConfig{
			Key:          key,
			Field:        st.specField(),
			StartValue:   r.Start,
			EndValue:     r.End,
			CurrentValue: r.Start,
			Step:         1,
			Width:        r.Len(),
			Schema:       st.spec.Schema,
		}
		return newRangeProvider(cfg.Field, cfg.StartValue, cfg.EndValue, cfg.Step), cfg, nil
	}

	rows, err := st.aliveLive.take(ctx, s.liveLookup, spec.Schema, int(need))
	if err != nil {
		return nil, nil, err
	}
	cols := make([]string, 0, len(st.aliveLive.columns))
	for _, c := range st.aliveLive.columns {
		cols = append(cols, c.Column)
	}
	tp, err := newTupleProvider(cols, rows)
	if err != nil {
		return nil, nil, err
	}
	return tp, nil, nil
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
