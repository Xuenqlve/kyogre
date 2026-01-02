package iquery

import (
	"context"
	"fmt"
	"math"
	"strings"
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

	wrapAt int64
	alive  *aliveSet

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

	// Strategy split:
	// - Single numeric field: use range pool
	// - Otherwise: use AliveSet (ScanValues)
	if len(params) != 1 || !isNumericType(params[0].Type) {
		st := &sequenceState{
			spec:  spec,
			alive: newAliveSet(params, s.cfg.AliveSetCapacity, s.cfg.AliveSetBatchSize),
		}
		s.setState(key, st)
		return st, nil
	}

	req := LookupRequest{
		Schema: spec.Schema,
		Params: params,
	}
	res, err := s.lookup.LookupBounds(ctx, req)
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
	st.wrapAt = minInt64(s.cfg.WrapAt, hardMaxInt64(params[0].Type))
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

	if st.alive != nil {
		return nil, fmt.Errorf("ReserveInsert not supported for AliveSet strategy (use ReserveInsertProvider)")
	}

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
	if st.wrapAt > 0 && st.nextNew > st.wrapAt {
		return nil, fmt.Errorf("insert range exhausted for %s: next=%d wrapAt=%d", key, st.nextNew, st.wrapAt)
	}
	if st.nextNew > st.newEnd {
		// Refresh high-water.
		req := LookupRequest{Schema: spec.Schema, Params: []BoundParam{{Column: st.specField()}}}
		res, err := s.lookup.LookupBounds(ctx, req)
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
	if st.wrapAt > 0 && end > st.wrapAt {
		end = st.wrapAt
	}
	if end > st.newEnd {
		end = st.newEnd
	}
	if end < start {
		return nil, fmt.Errorf("insert range exhausted for %s: next=%d wrapAt=%d", key, st.nextNew, st.wrapAt)
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
		size, err := normalizeSize(need, s.cfg.AllowedSizes, s.cfg.StrictSizes)
		if err != nil {
			return nil, nil, err
		}

		if r, ok := st.pool.reserveFromFreeForInsert(size); ok {
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

		if st.wrapAt > 0 && st.nextNew > st.wrapAt {
			return nil, nil, fmt.Errorf("insert range exhausted for %s: next=%d wrapAt=%d", key, st.nextNew, st.wrapAt)
		}
		if st.nextNew > st.newEnd {
			req := LookupRequest{Schema: spec.Schema, Params: []BoundParam{{Column: st.specField()}}}
			res, err := s.lookup.LookupBounds(ctx, req)
			if err != nil {
				return nil, nil, err
			}
			if len(res.Bounds) == 0 {
				return nil, nil, fmt.Errorf("lookup returned empty bounds for %s", key)
			}
			maxV, err := toInt64(res.Bounds[0].MaxValue)
			if err != nil {
				return nil, nil, err
			}
			if maxV >= st.nextNew {
				st.nextNew = maxV + 1
			}
			st.newEnd = st.nextNew + s.cfg.InsertWindowSize - 1
		}

		start := st.nextNew
		end := start + size - 1
		if st.wrapAt > 0 && end > st.wrapAt {
			end = st.wrapAt
		}
		if end > st.newEnd {
			end = st.newEnd
		}
		if end < start {
			return nil, nil, fmt.Errorf("insert range exhausted for %s: next=%d wrapAt=%d", key, st.nextNew, st.wrapAt)
		}
		st.nextNew = end + 1
		r := IntRange{Start: start, End: end}
		st.pool.markPendingInsert(r)

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

	// AliveSet: tuple provider -> map[column]any for IN batch usage
	rows, err := st.alive.take(ctx, s.lookup, spec.Schema, int(need))
	if err != nil {
		return nil, nil, err
	}
	cols := make([]string, 0, len(st.alive.columns))
	for _, c := range st.alive.columns {
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

	if st.alive != nil {
		return nil, fmt.Errorf("ReserveUpdate not supported for AliveSet strategy (use ReserveUpdateProvider)")
	}

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

	rows, err := st.alive.take(ctx, s.lookup, spec.Schema, int(need))
	if err != nil {
		return nil, nil, err
	}
	cols := make([]string, 0, len(st.alive.columns))
	for _, c := range st.alive.columns {
		cols = append(cols, c.Column)
	}
	tp, err := newTupleProvider(cols, rows)
	if err != nil {
		return nil, nil, err
	}
	return tp, nil, nil
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

	if st.alive != nil {
		return nil, fmt.Errorf("ReserveDelete not supported for AliveSet strategy (use ReserveDeleteProvider)")
	}

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
			return nil, nil, fmt.Errorf("no live range available for delete: %s", key)
		}
		_ = st.pool.reserveDelete(r)
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

	rows, err := st.alive.take(ctx, s.lookup, spec.Schema, int(need))
	if err != nil {
		return nil, nil, err
	}
	cols := make([]string, 0, len(st.alive.columns))
	for _, c := range st.alive.columns {
		cols = append(cols, c.Column)
	}
	tp, err := newTupleProvider(cols, rows)
	if err != nil {
		return nil, nil, err
	}
	return tp, nil, nil
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

func hardMaxInt64(typ string) int64 {
	if typ == "" {
		return math.MaxInt64
	}
	t := strings.ToLower(strings.TrimSpace(typ))
	switch {
	case strings.Contains(t, "bigint unsigned"):
		return math.MaxInt64 // cannot represent full uint64 in int64, cap for safety
	case strings.Contains(t, "bigint"):
		return math.MaxInt64
	case strings.Contains(t, "int unsigned"):
		return math.MaxInt32
	case strings.Contains(t, "int"):
		return math.MaxInt32
	case strings.Contains(t, "smallint unsigned"):
		return math.MaxInt16
	case strings.Contains(t, "smallint"):
		return math.MaxInt16
	case strings.Contains(t, "tinyint unsigned"):
		return math.MaxInt8
	case strings.Contains(t, "tinyint"):
		return math.MaxInt8
	default:
		return math.MaxInt64
	}
}

func isNumericType(typ string) bool {
	if typ == "" {
		return true
	}
	t := strings.ToLower(strings.TrimSpace(typ))
	switch {
	case strings.Contains(t, "int"),
		strings.Contains(t, "decimal"),
		strings.Contains(t, "numeric"),
		strings.Contains(t, "float"),
		strings.Contains(t, "double"):
		return true
	default:
		return false
	}
}

func minInt64(a, b int64) int64 {
	if a <= 0 {
		return b
	}
	if b <= 0 {
		return a
	}
	if a < b {
		return a
	}
	return b
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
