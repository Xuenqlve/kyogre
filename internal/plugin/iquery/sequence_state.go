package iquery

import (
	"context"
	"fmt"
	"sync"

	"github.com/xuenqlve/kyogre/pkg/tool/range_pool"
)

type sequenceState interface {
	reserveInsertProvider(ctx context.Context, need int64) (Provider, error)
	reserveUpdateProvider(ctx context.Context, need int64) (Provider, error)
	reserveDeleteProvider(ctx context.Context, need int64) (Provider, error)
}

type baseSequenceState struct {
	spec SequenceSpec
	mu   sync.Mutex
}

func (st *baseSequenceState) specField() string {
	if st.spec.Field != "" {
		return st.spec.Field
	}
	if len(st.spec.Fields) > 0 {
		return st.spec.Fields[0].Column
	}
	return ""
}

type numericSequenceState struct {
	baseSequenceState
	key string
	cfg *SequencerConfig

	pool *range_pool.RangePool
}

func newNumericSequenceState(spec SequenceSpec, key string, cfg *SequencerConfig, pool *range_pool.RangePool) *numericSequenceState {
	return &numericSequenceState{
		baseSequenceState: baseSequenceState{spec: spec},
		key:               key,
		cfg:               cfg,
		pool:              pool,
	}
}

func (st *numericSequenceState) reserveInsertProvider(ctx context.Context, need int64) (Provider, error) {
	st.mu.Lock()
	defer st.mu.Unlock()
	size, err := normalizeSize(need, st.cfg.AllowedSizes, st.cfg.StrictSizes)
	if err != nil {
		return nil, err
	}
	if r, ok := st.pool.ReserveInsert(size); ok {
		return newRangeProvider(st.specField(), r.Start, r.End, 1), nil
	}
	return nil, fmt.Errorf("no free range available for insert: %s", st.key)
}

func (st *numericSequenceState) reserveUpdateProvider(ctx context.Context, need int64) (Provider, error) {
	st.mu.Lock()
	defer st.mu.Unlock()
	size, err := normalizeSize(need, st.cfg.AllowedSizes, st.cfg.StrictSizes)
	if err != nil {
		return nil, err
	}
	r, ok := st.pool.ReserveUpdate(size)
	if !ok {
		return nil, fmt.Errorf("no live range available for update: %s", st.key)
	}
	return newRangeProvider(st.specField(), r.Start, r.End, 1), nil
}

func (st *numericSequenceState) reserveDeleteProvider(ctx context.Context, need int64) (Provider, error) {
	st.mu.Lock()
	defer st.mu.Unlock()
	size, err := normalizeSize(need, st.cfg.AllowedSizes, st.cfg.StrictSizes)
	if err != nil {
		return nil, err
	}
	r, ok := st.pool.ReserveDelete(size)
	if !ok {
		return nil, fmt.Errorf("no live range available for delete: %s", st.key)
	}
	return newRangeProvider(st.specField(), r.Start, r.End, 1), nil
}

type sequenceLookups struct {
	free Lookup
	live Lookup
}
type tupleSequenceState struct {
	baseSequenceState

	lookups   sequenceLookups
	aliveFree *aliveSet
	aliveLive *aliveSet
}

func newTupleSequenceState(spec SequenceSpec, lookups sequenceLookups, aliveFree *aliveSet, aliveLive *aliveSet) *tupleSequenceState {
	return &tupleSequenceState{
		baseSequenceState: baseSequenceState{spec: spec},
		lookups:           lookups,
		aliveFree:         aliveFree,
		aliveLive:         aliveLive,
	}
}

func (st *tupleSequenceState) reserveInsertProvider(ctx context.Context, need int64) (Provider, error) {
	st.mu.Lock()
	defer st.mu.Unlock()

	rows, err := st.aliveFree.take(ctx, st.lookups.free, st.spec.Schema, int(need))
	if err != nil {
		return nil, err
	}
	cols := make([]string, 0, len(st.aliveFree.columns))
	for _, c := range st.aliveFree.columns {
		cols = append(cols, c.Column)
	}
	return newTupleProvider(cols, rows)
}

func (st *tupleSequenceState) reserveUpdateProvider(ctx context.Context, need int64) (Provider, error) {
	st.mu.Lock()
	defer st.mu.Unlock()

	rows, err := st.aliveLive.take(ctx, st.lookups.live, st.spec.Schema, int(need))
	if err != nil {
		return nil, err
	}
	cols := make([]string, 0, len(st.aliveLive.columns))
	for _, c := range st.aliveLive.columns {
		cols = append(cols, c.Column)
	}
	return newTupleProvider(cols, rows)
}

func (st *tupleSequenceState) reserveDeleteProvider(ctx context.Context, need int64) (Provider, error) {
	st.mu.Lock()
	defer st.mu.Unlock()

	rows, err := st.aliveLive.take(ctx, st.lookups.live, st.spec.Schema, int(need))
	if err != nil {
		return nil, err
	}
	cols := make([]string, 0, len(st.aliveLive.columns))
	for _, c := range st.aliveLive.columns {
		cols = append(cols, c.Column)
	}
	return newTupleProvider(cols, rows)
}
