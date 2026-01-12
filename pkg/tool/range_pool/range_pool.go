package range_pool

import (
	"context"
	"fmt"
	"math/rand"

	"github.com/xuenqlve/common/log"
)

// IntRange represents a contiguous, inclusive range of int64 identifiers.
type IntRange struct {
	Start int64
	End   int64
}

// Len returns the number of values covered by the range.
func (r IntRange) Len() int64 {
	if r.End < r.Start {
		return 0
	}
	return r.End - r.Start + 1
}

// Valid reports whether the range describes at least one value.
func (r IntRange) Valid() bool { return r.Len() > 0 }

// RangePool distributes ranges between a "live" partition (update/delete consumers)
// and a "free" partition (insert consumers).
//
// live and free are intentionally independent. This package does not implicitly
// transfer segments between partitions; instead, each partition is replenished
// exclusively via its own refill callback (LiveRefill / FreeRefill) or explicit
// AddLiveRange/AddFreeRange calls by the caller.
type RangePool struct {
	cfg RangePoolConfig

	live *tierPartition
	free *tierPartition
}

// RangePoolRefillFunc is invoked when a partition needs additional allocation capacity.
//
// enableLoop indicates the partition's allocation domain has entered looping mode.
// When enableLoop is true, refillWindow.Start must remain stable for the lifetime of the pool.
//
// refillWindow describes the allocation domain window (inclusive) for the partition.
// The window may be expanded over time; holes inside the window are allowed by design.
type RangePoolRefillFunc func(partition string, need int64) (enableLoop bool, refillWindow IntRange, err error)

// TierConfig defines one tier's sizing and threshold behavior.
type TierConfig struct {
	Size      int64
	Threshold int
	MaxCount  int
}

// RangePoolConfig configures live/free tiers and optional refill callbacks.
type RangePoolConfig struct {
	LiveTiers []TierConfig
	FreeTiers []TierConfig

	Refill RangePoolRefillFunc
	//FreeRefill RangePoolRefillFunc

	RandSource rand.Source
}

// RangePoolOption mutates RangePool configuration before construction.
type RangePoolOption func(*RangePoolConfig)

// WithLiveTiers overrides the default live tier configuration.
func WithLiveTiers(tiers []TierConfig) RangePoolOption {
	return func(cfg *RangePoolConfig) {
		cfg.LiveTiers = append([]TierConfig(nil), tiers...)
	}
}

// WithFreeTiers overrides the default free tier configuration.
func WithFreeTiers(tiers []TierConfig) RangePoolOption {
	return func(cfg *RangePoolConfig) {
		cfg.FreeTiers = append([]TierConfig(nil), tiers...)
	}
}

// WithLiveRefill sets the live partition refill callback.
func WithRefill(fn RangePoolRefillFunc) RangePoolOption {
	return func(cfg *RangePoolConfig) { cfg.Refill = fn }
}

//// WithFreeRefill sets the free partition refill callback.
//func WithFreeRefill(fn RangePoolRefillFunc) RangePoolOption {
//	return func(cfg *RangePoolConfig) { cfg.FreeRefill = fn }
//}

// WithRandSource overrides the random source used when selecting segments.
func WithRandSource(src rand.Source) RangePoolOption {
	return func(cfg *RangePoolConfig) { cfg.RandSource = src }
}

const (
	RangePoolLiveName = "live"
	RangePoolFreeName = "free"
)

// NewRangePool builds a RangePool.
//
// liveSeed defines the initial allocation window for the live partition when LiveRefill is nil.
// For free partition, the allocation window is only defined by FreeRefill (or explicitly injected ranges).
func NewRangePool(opts ...RangePoolOption) (*RangePool, error) {
	cfg := defaultRangePoolConfig()
	for _, opt := range opts {
		opt(&cfg)
	}
	if err := cfg.ValidateAndSetDefault(); err != nil {
		return nil, err
	}
	live, err := newTierPartition(RangePoolLiveName, cfg.LiveTiers, cfg.Refill, cfg.rand())
	if err != nil {
		return nil, err
	}
	free, err := newTierPartition(RangePoolFreeName, cfg.FreeTiers, cfg.Refill, cfg.rand())
	if err != nil {
		return nil, err
	}
	pool := &RangePool{
		cfg:  cfg,
		live: live,
		free: free,
	}
	// Start refill IO workers before bootstrap to handle window initialization.
	ctx := context.Background()
	pool.live.startRefillWorker(ctx)
	pool.free.startRefillWorker(ctx)
	if err = pool.live.bootstrap(); err != nil {
		return nil, err
	}
	if err = pool.free.bootstrap(); err != nil {
		return nil, err
	}

	// Start background refiller goroutines
	pool.live.startRefiller(ctx)
	pool.free.startRefiller(ctx)

	return pool, nil
}

func (p *RangePool) DebugLog(t string) {
	ls, lc, le, ll := p.live.window.windowState()
	fs, fc, fe, fl := p.free.window.windowState()
	switch t {
	case RangePoolFreeName:
		log.Infof("[RangePool] free window start=%d cursor=%d end=%d loop=%v", fs, fc, fe, fl)
		p.free.tiers.mu.RLock()
		for _, v := range p.free.tiers.tiers {
			if len(v.segments) == 0 {
				log.Infof("size:%v range <empty> len:%d", v.cfg.Size, 0)
				continue
			}
			log.Infof("size:%v range %d~%d len:%d", v.cfg.Size, v.segments[0].Start, v.segments[len(v.segments)-1].End, len(v.segments))
		}
		p.free.tiers.mu.RUnlock()
	case RangePoolLiveName:
		log.Infof("[RangePool] live window start=%d cursor=%d end=%d loop=%v", ls, lc, le, ll)
		p.live.tiers.mu.RLock()
		for _, v := range p.live.tiers.tiers {
			if len(v.segments) == 0 {
				log.Infof("size:%v range <empty> len:%d", v.cfg.Size, 0)
				continue
			}
			log.Infof("size:%v range %d~%d len:%d", v.cfg.Size, v.segments[0].Start, v.segments[len(v.segments)-1].End, len(v.segments))
		}
		p.live.tiers.mu.RUnlock()
	default:
		log.Infof("[RangePool] live window start=%d cursor=%d end=%d loop=%v", ls, lc, le, ll)
		log.Infof("[RangePool] free window start=%d cursor=%d end=%d loop=%v", fs, fc, fe, fl)
		log.Infof("[RangePool] live tiers ...")
		p.live.tiers.mu.RLock()
		for _, v := range p.live.tiers.tiers {
			if len(v.segments) == 0 {
				log.Infof("size:%v range <empty> len:%d", v.cfg.Size, 0)
				continue
			}
			log.Infof("size:%v range %d~%d len:%d", v.cfg.Size, v.segments[0].Start, v.segments[len(v.segments)-1].End, len(v.segments))
		}
		p.live.tiers.mu.RUnlock()
		log.Infof("[RangePool] free tiers ...")
		p.free.tiers.mu.RLock()
		for _, v := range p.free.tiers.tiers {
			if len(v.segments) == 0 {
				log.Infof("size:%v range <empty> len:%d", v.cfg.Size, 0)
				continue
			}
			log.Infof("size:%v range %d~%d len:%d", v.cfg.Size, v.segments[0].Start, v.segments[len(v.segments)-1].End, len(v.segments))
		}
		p.free.tiers.mu.RUnlock()
	}

}

// ReserveInsert reserves a continuous range for INSERT operations from the free partition.
//
// It consumes the chosen segment from the free tiers. This is typically used to reuse
// previously-deleted id ranges. It does not allocate "new ids" beyond the current pool;
// callers may implement a separate high-water allocator if needed.
func (p *RangePool) ReserveInsert(need int64) (IntRange, bool) {
	size, ok := p.free.pickTierSize(need)
	if !ok {
		return IntRange{}, false
	}
	r, err := p.free.consume(size)
	if err != nil {
		return IntRange{}, false
	}
	return r, true
}

// ReserveUpdate returns a continuous range for UPDATE operations from the live partition.
//
// It does not consume the chosen segment from the live tiers; repeated calls may return
// the same segment. When the live tiers are exhausted, it falls back to best-effort
// sampling within the current live allocation window.
func (p *RangePool) ReserveUpdate(need int64) (IntRange, bool) {
	size, ok := p.live.pickTierSize(need)
	if !ok {
		return IntRange{}, false
	}
	r, err := p.live.peek(size)
	if err == nil && r.Valid() {
		return r, true
	}
	return IntRange{}, false
}

// ReserveDelete reserves a continuous range for DELETE operations from the live partition.
//
// It consumes the chosen segment from the live tiers. When the live tiers are exhausted,
// it falls back to best-effort sampling within the current live allocation window.
func (p *RangePool) ReserveDelete(need int64) (IntRange, bool) {
	size, ok := p.live.pickTierSize(need)
	if !ok {
		return IntRange{}, false
	}
	r, err := p.live.consume(size)
	if err == nil && r.Valid() {
		return r, true
	}
	return IntRange{}, false
}

// LiveWindow returns (start, cursor, end, enableLoop) for the live allocation window.
func (p *RangePool) LiveWindow() (int64, int64, int64, bool) { return p.live.window.windowState() }

// FreeWindow returns (start, cursor, end, enableLoop) for the free allocation window.
func (p *RangePool) FreeWindow() (int64, int64, int64, bool) { return p.free.window.windowState() }

// Close stops all background refiller goroutines and releases resources.
func (p *RangePool) Close() error {
	if err := p.live.Close(); err != nil {
		return err
	}
	if err := p.free.Close(); err != nil {
		return err
	}
	return nil
}

// partitionError indicates a tier is unavailable (used by allocateSequential).
type partitionError struct {
	name string
	size int64
}

// Error implements the error interface.
func (e *partitionError) Error() string {
	return fmt.Sprintf("%s tier %d unavailable", e.name, e.size)
}
