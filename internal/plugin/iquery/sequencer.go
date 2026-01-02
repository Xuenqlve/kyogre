package iquery

import (
	"context"
	"fmt"
	"sync"

	"github.com/xuenqlve/common/schema_store"
)

// SequenceSpec defines a unique constraint dimension to manage.
// For the "90% case", it is a single int primary key column.
type SequenceSpec struct {
	Schema schema_store.SchemaKey
	// Field is the legacy single-column unique field.
	// Prefer Fields for multi-column unique constraints.
	Field string
	// Fields are the columns participating in the unique constraint (ordered).
	Fields []BoundParam
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
	pipeline string
	lookup   Lookup
	cfg      *SequencerConfig

	stateMu sync.RWMutex
	states  map[string]*sequenceState

	keyMuMu sync.Mutex
	keyMu   map[string]*sync.Mutex
}

type sequenceState struct {
	spec SequenceSpec

	mu   sync.Mutex
	pool *rangePool

	// nextNew is the next id in the "new high-water" region to allocate for inserts.
	nextNew int64
	// newEnd is the current end boundary for the in-memory new window.
	newEnd int64
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
	if s.lookup == nil {
		return fmt.Errorf("sequencer lookup is nil")
	}
	return nil
}

func (s *Sequencer) BindLookup(lookup Lookup) { s.lookup = lookup }

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
		params = []BoundParam{{Column: spec.Field}}
	}
	if len(params) == 0 {
		return nil, fmt.Errorf("invalid spec: empty fields")
	}

	req := LookupRequest{
		Schema: spec.Schema,
		Params: params,
	}
	res, err := s.lookup.Lookup(ctx, req)
	if err != nil {
		return nil, err
	}
	// For now, only support "90% case": first field is int and drives allocation.
	if len(res.Bounds) == 0 {
		return nil, fmt.Errorf("lookup returned empty bounds for %s", key)
	}
	b := res.Bounds[0]
	maxV, err := toInt64(b.MaxValue)
	if err != nil {
		return nil, err
	}
	minV, err := toInt64(b.MinValue)
	if err != nil {
		return nil, err
	}
	st := &sequenceState{
		spec:    spec,
		pool:    newRangePool(minV, maxV, maxV),
		nextNew: maxV + 1,
		newEnd:  maxV,
	}
	s.setState(key, st)
	return st, nil
}

// ReserveInsert reserves a continuous range for INSERT operations.
// It tries to reuse deleted ranges first; otherwise, it allocates new ids above high-water.
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

	size, err := normalizeSize(need, s.cfg.AllowedSizes, s.cfg.StrictSizes)
	if err != nil {
		return nil, err
	}

	// 1) reuse deleted ids (safe).
	if r, ok := st.pool.reserveFromFreeForInsert(size); ok {
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

	// 2) allocate from new high-water window. Refill window by refreshing max from lookup when needed.
	if st.nextNew > st.newEnd {
		// Refresh high-water.
		req := LookupRequest{Schema: spec.Schema, Params: []BoundParam{{Column: st.specField()}}}
		res, err := s.lookup.Lookup(ctx, req)
		if err != nil {
			return nil, err
		}
		if len(res.Bounds) == 0 {
			return nil, fmt.Errorf("lookup returned empty bounds for %s", key)
		}
		maxV, err := toInt64(res.Bounds[0].MaxValue)
		if err != nil {
			return nil, err
		}
		// Ensure monotonic.
		if maxV >= st.nextNew {
			st.nextNew = maxV + 1
		}
		st.newEnd = st.nextNew + s.cfg.InsertWindowSize - 1
	}

	start := st.nextNew
	end := start + size - 1
	if end > st.newEnd {
		end = st.newEnd
	}
	st.nextNew = end + 1
	r := IntRange{Start: start, End: end}
	st.pool.markPendingInsert(r)

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

	size, err := normalizeSize(need, s.cfg.AllowedSizes, s.cfg.StrictSizes)
	if err != nil {
		return nil, err
	}
	r, ok := st.pool.takeFromLive(size)
	if !ok {
		// best-effort: fall back to exist range based on bounds.
		if st.pool.existMax >= st.pool.existMin && st.pool.existMax > 0 {
			start := st.pool.existMin
			end := start + size - 1
			if end > st.pool.existMax {
				end = st.pool.existMax
			}
			r = IntRange{Start: start, End: end}
			ok = r.Valid()
		}
	}
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

// ReserveDelete reserves an existing live range for DELETE and moves it to free pool on success.
// Since we don't have a success callback here, we optimistically move it to free pool.
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

	size, err := normalizeSize(need, s.cfg.AllowedSizes, s.cfg.StrictSizes)
	if err != nil {
		return nil, err
	}
	r, ok := st.pool.takeFromLive(size)
	if !ok {
		if st.pool.existMax >= st.pool.existMin && st.pool.existMax > 0 {
			start := st.pool.existMin
			end := start + size - 1
			if end > st.pool.existMax {
				end = st.pool.existMax
			}
			r = IntRange{Start: start, End: end}
			ok = r.Valid()
		}
	}
	if !ok {
		return nil, fmt.Errorf("no live range available for delete: %s", key)
	}
	// Reserve the delete range, waiting for commit/rollback.
	_ = st.pool.reserveDelete(r)

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

func (st *sequenceState) specField() string {
	if st.spec.Field != "" {
		return st.spec.Field
	}
	if len(st.spec.Fields) > 0 {
		return st.spec.Fields[0].Column
	}
	return ""
}

// CommitInsert marks a previously reserved insert range as "exists".
func (s *Sequencer) CommitInsert(ctx context.Context, spec SequenceSpec, cfg *SequenceConfig) error {
	if cfg == nil {
		return nil
	}
	key := spec.Key()
	mu := s.keyLock(key)
	mu.Lock()
	defer mu.Unlock()
	st, err := s.initState(ctx, spec)
	if err != nil {
		return err
	}
	st.mu.Lock()
	defer st.mu.Unlock()
	return st.pool.commitInsert(IntRange{Start: cfg.StartValue, End: cfg.EndValue})
}

func (s *Sequencer) RollbackInsert(ctx context.Context, spec SequenceSpec, cfg *SequenceConfig) error {
	if cfg == nil {
		return nil
	}
	key := spec.Key()
	mu := s.keyLock(key)
	mu.Lock()
	defer mu.Unlock()
	st, err := s.initState(ctx, spec)
	if err != nil {
		return err
	}
	st.mu.Lock()
	defer st.mu.Unlock()
	return st.pool.rollbackInsert(IntRange{Start: cfg.StartValue, End: cfg.EndValue})
}

func (s *Sequencer) CommitDelete(ctx context.Context, spec SequenceSpec, cfg *SequenceConfig) error {
	if cfg == nil {
		return nil
	}
	key := spec.Key()
	mu := s.keyLock(key)
	mu.Lock()
	defer mu.Unlock()
	st, err := s.initState(ctx, spec)
	if err != nil {
		return err
	}
	st.mu.Lock()
	defer st.mu.Unlock()
	return st.pool.commitDelete(IntRange{Start: cfg.StartValue, End: cfg.EndValue})
}

func (s *Sequencer) RollbackDelete(ctx context.Context, spec SequenceSpec, cfg *SequenceConfig) error {
	if cfg == nil {
		return nil
	}
	key := spec.Key()
	mu := s.keyLock(key)
	mu.Lock()
	defer mu.Unlock()
	st, err := s.initState(ctx, spec)
	if err != nil {
		return err
	}
	st.mu.Lock()
	defer st.mu.Unlock()
	return st.pool.rollbackDelete(IntRange{Start: cfg.StartValue, End: cfg.EndValue})
}

func (s *Sequencer) Close() error {
	if s.lookup == nil {
		return nil
	}
	return s.lookup.Close()
}
