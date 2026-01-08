package range_pool

import (
	"context"
	"fmt"
	"math/rand"
	"sort"
	"sync"
	"time"

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
	rnd := cfg.rand()
	live, err := newTierPartition(RangePoolLiveName, cfg.LiveTiers, cfg.Refill, rnd)
	if err != nil {
		return nil, err
	}
	free, err := newTierPartition(RangePoolFreeName, cfg.FreeTiers, cfg.Refill, rnd)
	if err != nil {
		return nil, err
	}
	pool := &RangePool{
		cfg:  cfg,
		live: live,
		free: free,
	}
	if err = pool.live.bootstrap(); err != nil {
		return nil, err
	}
	if err = pool.free.bootstrap(); err != nil {
		return nil, err
	}

	// Start background refiller goroutines
	ctx := context.Background()
	pool.live.startRefiller(ctx)
	pool.free.startRefiller(ctx)

	return pool, nil
}

func (p *RangePool) DebugLog(t string) {
	ls, lc, le, ll := p.live.windowState()
	fs, fc, fe, fl := p.free.windowState()
	switch t {
	case RangePoolFreeName:
		log.Infof("[RangePool] free window start=%d cursor=%d end=%d loop=%v", fs, fc, fe, fl)
		p.free.mu.RLock()
		for _, v := range p.free.tiers {
			if len(v.segments) == 0 {
				log.Infof("size:%v range <empty> len:%d", v.cfg.Size, 0)
				continue
			}
			log.Infof("size:%v range %d~%d len:%d", v.cfg.Size, v.segments[0].Start, v.segments[len(v.segments)-1].End, len(v.segments))
		}
		p.free.mu.RUnlock()
	case RangePoolLiveName:
		log.Infof("[RangePool] live window start=%d cursor=%d end=%d loop=%v", ls, lc, le, ll)
		p.live.mu.RLock()
		for _, v := range p.live.tiers {
			if len(v.segments) == 0 {
				log.Infof("size:%v range <empty> len:%d", v.cfg.Size, 0)
				continue
			}
			log.Infof("size:%v range %d~%d len:%d", v.cfg.Size, v.segments[0].Start, v.segments[len(v.segments)-1].End, len(v.segments))
		}
		p.live.mu.RUnlock()
	default:
		log.Infof("[RangePool] live window start=%d cursor=%d end=%d loop=%v", ls, lc, le, ll)
		log.Infof("[RangePool] free window start=%d cursor=%d end=%d loop=%v", fs, fc, fe, fl)
		log.Infof("[RangePool] live tiers ...")
		p.live.mu.RLock()
		for _, v := range p.live.tiers {
			if len(v.segments) == 0 {
				log.Infof("size:%v range <empty> len:%d", v.cfg.Size, 0)
				continue
			}
			log.Infof("size:%v range %d~%d len:%d", v.cfg.Size, v.segments[0].Start, v.segments[len(v.segments)-1].End, len(v.segments))
		}
		p.live.mu.RUnlock()
		log.Infof("[RangePool] free tiers ...")
		p.free.mu.RLock()
		for _, v := range p.free.tiers {
			if len(v.segments) == 0 {
				log.Infof("size:%v range <empty> len:%d", v.cfg.Size, 0)
				continue
			}
			log.Infof("size:%v range %d~%d len:%d", v.cfg.Size, v.segments[0].Start, v.segments[len(v.segments)-1].End, len(v.segments))
		}
		p.free.mu.RUnlock()
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
	if r, ok := p.live.randomFromWindow(size); ok {
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
	if r, ok = p.live.randomFromWindow(size); ok {
		return r, true
	}
	return IntRange{}, false
}

// LiveWindow returns (start, cursor, end, enableLoop) for the live allocation window.
func (p *RangePool) LiveWindow() (int64, int64, int64, bool) { return p.live.windowState() }

// FreeWindow returns (start, cursor, end, enableLoop) for the free allocation window.
func (p *RangePool) FreeWindow() (int64, int64, int64, bool) { return p.free.windowState() }

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

// tierPartition represents the tier hierarchy backing either the live or free pool.
type tierPartition struct {
	name   string
	tiers  []*tierState
	idx    map[int64]int
	rand   *rand.Rand
	refill RangePoolRefillFunc

	mu       sync.RWMutex
	refillMu sync.Mutex
	randMu   sync.Mutex     // Protects rand for thread-safe random number generation

	windowInit  bool
	windowStart int64
	windowEnd   int64
	cursor      int64
	enableLoop  bool

	// Background refiller fields
	refillSignal chan int      // Buffered channel for refill signals (tier index)
	stopChan     chan struct{} // Signal to stop the refiller goroutine
	wg           sync.WaitGroup
	ctx          context.Context
	cancel       context.CancelFunc
}

type tierState struct {
	cfg        TierConfig
	segments   []IntRange
	lastRefill time.Time // Last refill time (throttling)
}

// newTierPartition builds an ordered set of tiers and an O(1) lookup from size to tier index.
func newTierPartition(name string, tiers []TierConfig, refill RangePoolRefillFunc, rnd *rand.Rand) (*tierPartition, error) {
	if len(tiers) == 0 {
		return nil, fmt.Errorf("%s partition requires at least one tier", name)
	}
	sort.Slice(tiers, func(i, j int) bool { return tiers[i].Size < tiers[j].Size })
	index := make(map[int64]int, len(tiers))
	tp := &tierPartition{name: name, rand: rnd, refill: refill, idx: index}
	for i := range tiers {
		if tiers[i].Size <= 0 {
			return nil, fmt.Errorf("%s tier %d has invalid size", name, tiers[i].Size)
		}
		if tiers[i].Threshold < 0 {
			return nil, fmt.Errorf("%s tier %d threshold must be >= 0", name, tiers[i].Size)
		}
		copyCfg := tiers[i]
		tp.tiers = append(tp.tiers, &tierState{cfg: copyCfg})
		index[copyCfg.Size] = len(tp.tiers) - 1
	}
	return tp, nil
}

func (p *tierPartition) windowState() (start, cursor, end int64, enableLoop bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.windowStart, p.cursor, p.windowEnd, p.enableLoop
}

func (p *tierPartition) pickTierSize(need int64) (int64, bool) {
	if need <= 0 {
		need = 1
	}
	for _, tier := range p.tiers {
		if tier.cfg.Size >= need {
			return tier.cfg.Size, true
		}
	}
	return 0, false
}

func (p *tierPartition) randomFromWindow(size int64) (IntRange, bool) {
	if size <= 0 {
		return IntRange{}, false
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if !p.windowInit {
		return IntRange{}, false
	}
	window := IntRange{Start: p.windowStart, End: p.windowEnd}
	if !window.Valid() {
		return IntRange{}, false
	}
	if size > window.Len() {
		return IntRange{}, false
	}
	maxOffset := window.Len() - size
	offset := int64(0)
	if maxOffset > 0 {
		p.randMu.Lock()
		offset = p.rand.Int63n(maxOffset + 1)
		p.randMu.Unlock()
	}
	start := window.Start + offset
	end := start + size - 1
	r := IntRange{Start: start, End: end}
	return r, r.Valid()
}

func (p *tierPartition) bootstrap() error {
	needInit := p.initNeedCapacity()
	initNeed := needInit * 2
	if initNeed <= 0 {
		initNeed = 1
	}
	if err := p.ensureWindowCapacity(initNeed, true); err != nil {
		return err
	}

	if needInit <= 0 {
		return nil
	}

	for _, tier := range p.tiers {
		if tier.cfg.MaxCount <= 0 {
			continue
		}
		segs, err := p.allocateSequential(tier.cfg.Size, tier.cfg.MaxCount, true)
		if err != nil {
			return err
		}
		tier.segments = append(tier.segments, segs...)
	}
	return nil
}

func (p *tierPartition) initNeedCapacity() int64 {
	var need int64
	for _, tier := range p.tiers {
		if tier.cfg.MaxCount <= 0 {
			continue
		}
		need += tier.cfg.Size * int64(tier.cfg.MaxCount)
	}
	return need
}

func (p *tierPartition) ensureWindowCapacity(need int64, bootstrap bool) error {
	if p.refill == nil {
		// Window must already be initialized by seed.
		p.mu.RLock()
		inited := p.windowInit
		window := IntRange{Start: p.windowStart, End: p.windowEnd}
		p.mu.RUnlock()
		if !inited || !window.Valid() {
			return fmt.Errorf("%s partition requires either a valid seed window or a refill callback", p.name)
		}
		if window.Len() < need {
			return fmt.Errorf("%s allocation window capacity insufficient: need=%d have=%d", p.name, need, window.Len())
		}
		return nil
	}
	if need <= 0 {
		return nil
	}

	for {
		p.mu.RLock()
		inited := p.windowInit
		window := IntRange{Start: p.windowStart, End: p.windowEnd}
		p.mu.RUnlock()

		if !inited {
			// Two-phase: do IO refill out of lock, then apply under lock.
			enableLoop, win, err := p.refillWindow(need)
			if err != nil {
				return err
			}
			if !win.Valid() {
				return fmt.Errorf("%s refill returned invalid window", p.name)
			}
			p.mu.Lock()
			// Another goroutine may have initialized the window while we were refilling.
			if !p.windowInit {
				p.windowInit = true
				p.windowStart = win.Start
				p.windowEnd = win.End
				p.cursor = win.Start
				p.enableLoop = p.enableLoop || enableLoop
				p.mu.Unlock()
				continue
			}
			// Window already initialized: enforce stable start.
			if win.Start != p.windowStart {
				p.mu.Unlock()
				return fmt.Errorf("%s refill window start changed: %d -> %d", p.name, p.windowStart, win.Start)
			}
			p.enableLoop = p.enableLoop || enableLoop
			if win.End > p.windowEnd {
				p.windowEnd = win.End
			}
			p.mu.Unlock()
			continue
		}

		if window.Len() >= need {
			return nil
		}

		missing := need - window.Len()
		prevEnd := window.End
		enableLoop, win, err := p.refillWindow(missing)
		if err != nil {
			return err
		}
		if !win.Valid() {
			return fmt.Errorf("%s refill returned invalid window", p.name)
		}
		p.mu.Lock()
		if win.Start != p.windowStart {
			p.mu.Unlock()
			return fmt.Errorf("%s refill window start changed: %d -> %d", p.name, p.windowStart, win.Start)
		}
		p.enableLoop = p.enableLoop || enableLoop
		if win.End > p.windowEnd {
			p.windowEnd = win.End
		} else if bootstrap && win.End <= prevEnd {
			p.mu.Unlock()
			return fmt.Errorf("%s allocation window capacity insufficient after refill: need=%d have=%d", p.name, need, (IntRange{Start: p.windowStart, End: p.windowEnd}).Len())
		}
		p.mu.Unlock()
	}
}

func (p *tierPartition) allocateSequential(size int64, count int, bootstrap bool) ([]IntRange, error) {
	if count <= 0 {
		return nil, nil
	}
	if size <= 0 {
		return nil, fmt.Errorf("%s allocate invalid size %d", p.name, size)
	}
	for {
		p.mu.Lock()
		if !p.windowInit {
			p.mu.Unlock()
			return nil, fmt.Errorf("%s allocation window not initialized", p.name)
		}
		window := IntRange{Start: p.windowStart, End: p.windowEnd}
		if !window.Valid() {
			p.mu.Unlock()
			return nil, fmt.Errorf("%s allocation window not initialized", p.name)
		}
		if window.Len() < size {
			p.mu.Unlock()
			return nil, fmt.Errorf("%s allocation window too small for size=%d", p.name, size)
		}

		// If looping is enabled and we are at the tail, wrap before computing need.
		if p.enableLoop && p.cursor+size-1 > p.windowEnd {
			p.cursor = p.windowStart
		}

		startCursor := p.cursor
		desiredEnd := startCursor + size*int64(count) - 1
		prevEnd := p.windowEnd

		if desiredEnd <= p.windowEnd {
			segs := make([]IntRange, 0, count)
			cursor := p.cursor
			for i := 0; i < count; i++ {
				end := cursor + size - 1
				segs = append(segs, IntRange{Start: cursor, End: end})
				cursor = end + 1
			}
			p.cursor = cursor
			p.mu.Unlock()
			return segs, nil
		}
		p.mu.Unlock()

		needExtra := desiredEnd - prevEnd
		if p.refill != nil {
			enableLoop, win, err := p.refillWindow(needExtra)
			if err != nil {
				return nil, err
			}
			if !win.Valid() {
				return nil, fmt.Errorf("%s refill returned invalid window", p.name)
			}
			p.mu.Lock()
			if win.Start != p.windowStart {
				p.mu.Unlock()
				return nil, fmt.Errorf("%s refill window start changed: %d -> %d", p.name, p.windowStart, win.Start)
			}
			p.enableLoop = p.enableLoop || enableLoop
			if win.End > p.windowEnd {
				p.windowEnd = win.End
				p.mu.Unlock()
				continue
			}
			// No growth: if loop is enabled, we may wrap on next iteration; otherwise fail.
			if bootstrap {
				p.mu.Unlock()
				return nil, fmt.Errorf("%s allocation window exhausted during bootstrap: cursor=%d size=%d desiredEnd=%d windowEnd=%d", p.name, p.cursor, size, desiredEnd, p.windowEnd)
			}
			if !p.enableLoop {
				p.mu.Unlock()
				return nil, &partitionError{name: p.name, size: size}
			}
			p.mu.Unlock()
			continue
		}

		// No refill: only possible via loop.
		p.mu.RLock()
		loop := p.enableLoop
		winStart := p.windowStart
		winEnd := p.windowEnd
		p.mu.RUnlock()
		if bootstrap {
			return nil, fmt.Errorf("%s allocation window exhausted during bootstrap: cursor=%d size=%d desiredEnd=%d windowEnd=%d", p.name, startCursor, size, desiredEnd, winEnd)
		}
		if !loop {
			return nil, &partitionError{name: p.name, size: size}
		}
		// Wrap and retry.
		p.mu.Lock()
		p.cursor = winStart
		p.mu.Unlock()
	}
}

// peek returns a random segment from the requested tier without consuming it.
// Uses read lock for concurrent access and sends async refill signal if needed.
func (p *tierPartition) peek(size int64) (IntRange, error) {
	idx, ok := p.idx[size]
	if !ok {
		return IntRange{}, fmt.Errorf("%s tier %d not configured", p.name, size)
	}

	p.mu.RLock()
	tier := p.tiers[idx]
	if len(tier.segments) == 0 {
		p.mu.RUnlock()
		return IntRange{}, fmt.Errorf("%s tier %d exhausted", p.name, size)
	}

	// Use randMu to protect rand access
	p.randMu.Lock()
	sel := p.rand.Intn(len(tier.segments))
	p.randMu.Unlock()

	r := tier.segments[sel]

	// Check if refill is needed
	needRefill := len(tier.segments) <= tier.cfg.Threshold
	p.mu.RUnlock()

	// Post-peek: send non-blocking refill signal
	if needRefill && p.refillSignal != nil {
		select {
		case p.refillSignal <- idx:
			// Signal sent successfully
		default:
			// Channel full, signal already queued
		}
	}

	return r, nil
}

// consume removes and returns a random segment from the requested tier.
// Uses post-consumption async refill signal instead of blocking ensure.
func (p *tierPartition) consume(size int64) (IntRange, error) {
	idx, ok := p.idx[size]
	if !ok {
		return IntRange{}, fmt.Errorf("%s tier %d not configured", p.name, size)
	}

	// 1. Read lock: quick check if tier has segments
	p.mu.RLock()
	tier := p.tiers[idx]
	if len(tier.segments) == 0 {
		p.mu.RUnlock()
		return IntRange{}, fmt.Errorf("%s tier %d exhausted", p.name, size)
	}
	p.mu.RUnlock()

	// 2. Write lock: consume segment
	p.mu.Lock()
	// Double-check after acquiring write lock
	if len(tier.segments) == 0 {
		p.mu.Unlock()
		return IntRange{}, fmt.Errorf("%s tier %d exhausted", p.name, size)
	}

	// Use randMu to protect rand access
	p.randMu.Lock()
	sel := p.rand.Intn(len(tier.segments))
	p.randMu.Unlock()

	r := tier.segments[sel]
	tier.segments[sel] = tier.segments[len(tier.segments)-1]
	tier.segments = tier.segments[:len(tier.segments)-1]

	// Check if refill is needed
	needRefill := len(tier.segments) <= tier.cfg.Threshold
	p.mu.Unlock()

	// 3. Post-consumption: send non-blocking refill signal
	if needRefill && p.refillSignal != nil {
		select {
		case p.refillSignal <- idx:
			// Signal sent successfully
		default:
			// Channel full, signal already queued
		}
	}

	return r, nil
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

// splitFromUpper takes one segment from tier idx+1 and splits it into multiple lower segments.
func (p *tierPartition) splitFromUpper(idx int) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	upperIdx := idx + 1
	if upperIdx >= len(p.tiers) {
		return false
	}
	upper := p.tiers[upperIdx]
	if len(upper.segments) == 0 {
		return false
	}
	lowerSize := p.tiers[idx].cfg.Size
	if lowerSize == 0 || upper.cfg.Size%lowerSize != 0 {
		return false
	}
	ratio := int(upper.cfg.Size / lowerSize)
	if ratio <= 1 {
		return false
	}
	p.randMu.Lock()
	sel := p.rand.Intn(len(upper.segments))
	p.randMu.Unlock()
	parent := upper.segments[sel]
	upper.segments[sel] = upper.segments[len(upper.segments)-1]
	upper.segments = upper.segments[:len(upper.segments)-1]
	childSize := lowerSize
	cursor := parent.Start
	for i := 0; i < ratio; i++ {
		child := IntRange{Start: cursor, End: cursor + childSize - 1}
		p.tiers[idx].segments = append(p.tiers[idx].segments, child)
		cursor += childSize
	}
	return true
}

func (p *tierPartition) refillWindow(need int64) (enableLoop bool, refillWindow IntRange, err error) {
	if p.refill == nil {
		return false, IntRange{}, fmt.Errorf("%s partition missing refill callback", p.name)
	}
	if need <= 0 {
		need = 1
	}
	p.refillMu.Lock()
	defer p.refillMu.Unlock()
	return p.refill(p.name, need)
}

// addRange splits the input range into tier-sized segments (greedy from largest to smallest)
// and stores those segments in their corresponding tiers.
//func (p *tierPartition) addRange(r IntRange) error {
//	if !r.Valid() {
//		return nil
//	}
//	remain := r.Len()
//	cursor := r.Start
//	for i := len(p.tiers) - 1; i >= 0; i-- {
//		size := p.tiers[i].cfg.Size
//		for remain >= size {
//			seg := IntRange{Start: cursor, End: cursor + size - 1}
//			p.tiers[i].segments = append(p.tiers[i].segments, seg)
//			cursor += size
//			remain -= size
//		}
//	}
//	if remain != 0 {
//		return fmt.Errorf("range length %d not aligned with tier sizes", r.Len())
//	}
//	return nil
//}

// addExactRange appends r as-is to the tier that matches r.Len().
//func (p *tierPartition) addExactRange(r IntRange) error {
//	idx, ok := p.idx[r.Len()]
//	if !ok {
//		return fmt.Errorf("%s tier %d not configured", p.name, r.Len())
//	}
//	p.tiers[idx].segments = append(p.tiers[idx].segments, r)
//	return nil
//}

// defaultRangePoolConfig returns the default tier ladder used for both partitions.
func defaultRangePoolConfig() RangePoolConfig {
	tiers := []TierConfig{
		{Size: 1, Threshold: 5, MaxCount: 10},
		{Size: 5, Threshold: 6, MaxCount: 10},
		{Size: 10, Threshold: 6, MaxCount: 10},
		{Size: 20, Threshold: 5, MaxCount: 10},
		{Size: 100, Threshold: 5, MaxCount: 10},
		{Size: 500, Threshold: 1, MaxCount: 2},
	}
	return RangePoolConfig{LiveTiers: tiers, FreeTiers: tiers}
}

// rand returns a *rand.Rand constructed from cfg.RandSource, defaulting to a time-based seed.
func (cfg *RangePoolConfig) rand() *rand.Rand {
	src := cfg.RandSource
	if src == nil {
		src = rand.NewSource(time.Now().UnixNano())
	}
	return rand.New(src)
}

// ValidateAndSetDefault ensures tiers are ready for use.
func (cfg *RangePoolConfig) ValidateAndSetDefault() error {
	if len(cfg.LiveTiers) == 0 {
		cfg.LiveTiers = defaultRangePoolConfig().LiveTiers
	}
	if len(cfg.FreeTiers) == 0 {
		cfg.FreeTiers = defaultRangePoolConfig().FreeTiers
	}
	if cfg.Refill == nil {
		return fmt.Errorf("missing register RangePool refill func")
	}
	_, err := validateTierConfigs(RangePoolLiveName, cfg.LiveTiers)
	if err != nil {
		return err
	}
	_, err = validateTierConfigs(RangePoolFreeName, cfg.FreeTiers)
	if err != nil {
		return err
	}
	if err = validateTierDivisible(RangePoolLiveName, cfg.LiveTiers); err != nil {
		return err
	}
	if err = validateTierDivisible(RangePoolFreeName, cfg.FreeTiers); err != nil {
		return err
	}
	return nil
}

// validateTierConfigs performs lightweight validation that is independent of partition construction.
func validateTierConfigs(name string, tiers []TierConfig) (map[int64]struct{}, error) {
	if len(tiers) == 0 {
		return nil, fmt.Errorf("%s partition requires at least one tier", name)
	}
	sizes := make(map[int64]struct{}, len(tiers))
	for _, tier := range tiers {
		if tier.Size <= 0 {
			return nil, fmt.Errorf("%s tier %d has invalid size", name, tier.Size)
		}
		if tier.Threshold < 0 {
			return nil, fmt.Errorf("%s tier %d threshold must be >= 0", name, tier.Size)
		}
		if tier.MaxCount < 0 {
			return nil, fmt.Errorf("%s tier %d max count must be >= 0", name, tier.Size)
		}
		if tier.MaxCount > 0 && tier.MaxCount <= tier.Threshold {
			return nil, fmt.Errorf("%s tier %d max count must be > threshold", name, tier.Size)
		}
		if _, exists := sizes[tier.Size]; exists {
			return nil, fmt.Errorf("%s tier size %d duplicated", name, tier.Size)
		}
		sizes[tier.Size] = struct{}{}
	}
	return sizes, nil
}

func validateTierDivisible(name string, tiers []TierConfig) error {
	if len(tiers) == 0 {
		return fmt.Errorf("%s partition requires at least one tier", name)
	}
	sorted := append([]TierConfig(nil), tiers...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Size < sorted[j].Size })
	for i := 1; i < len(sorted); i++ {
		lo := sorted[i-1].Size
		hi := sorted[i].Size
		if lo == 0 || hi%lo != 0 {
			return fmt.Errorf("%s tier size %d must be divisible by %d", name, hi, lo)
		}
	}
	return nil
}

// ==================== Background Refiller Implementation ====================

// startRefiller initializes and starts the background refiller goroutine.
func (p *tierPartition) startRefiller(ctx context.Context) {
	p.ctx, p.cancel = context.WithCancel(ctx)
	p.refillSignal = make(chan int, len(p.tiers)) // Buffered channel
	p.stopChan = make(chan struct{})

	p.wg.Add(1)
	go p.refillerLoop()
}

// refillerLoop is the main loop of the background refiller goroutine.
// It listens for refill signals and performs periodic checks.
func (p *tierPartition) refillerLoop() {
	defer p.wg.Done()

	ticker := time.NewTicker(100 * time.Millisecond) // Periodic check interval
	defer ticker.Stop()

	for {
		select {
		case <-p.ctx.Done():
			return
		case <-p.stopChan:
			return

		case <-ticker.C:
			// Periodic check: refill all tiers if needed
			p.checkAndRefillAll()

		case idx := <-p.refillSignal:
			// Triggered refill for specific tier
			p.checkAndRefill(idx)
		}
	}
}

// checkAndRefillAll checks and refills all tiers that need refilling.
func (p *tierPartition) checkAndRefillAll() {
	for i := range p.tiers {
		p.checkAndRefill(i)
	}
}

// checkAndRefill checks if a specific tier needs refilling and performs the refill.
func (p *tierPartition) checkAndRefill(idx int) {
	if idx < 0 || idx >= len(p.tiers) {
		return
	}

	tier := p.tiers[idx]

	// 1. Read lock: quick check if refill is needed
	p.mu.RLock()
	needRefill := len(tier.segments) <= tier.cfg.Threshold
	timeSinceLastRefill := time.Since(tier.lastRefill)
	p.mu.RUnlock()

	if !needRefill {
		return
	}

	// 2. Throttling: avoid refilling too frequently
	if timeSinceLastRefill < 50*time.Millisecond {
		return
	}

	// 3. Perform refill (write lock)
	p.mu.Lock()
	// Double-check after acquiring write lock
	if len(tier.segments) > tier.cfg.Threshold {
		p.mu.Unlock()
		return
	}

	target := tier.cfg.MaxCount
	if target <= 0 {
		target = tier.cfg.Threshold + 1
	}

	if idx == len(p.tiers)-1 {
		// Top tier: refill from allocation window
		p.refillTopTierUnsafe(idx, target)
	} else {
		// Middle tier: refill by splitting from upper tiers
		p.refillMiddleTierUnsafe(idx, target)
	}

	tier.lastRefill = time.Now()
	p.mu.Unlock()

	log.Debugf("[RangePool] %s tier %d refilled to %d segments", p.name, tier.cfg.Size, len(tier.segments))
}

// refillTopTierUnsafe refills the top tier from the allocation window.
// Caller must hold write lock (mu.Lock).
func (p *tierPartition) refillTopTierUnsafe(idx int, target int) {
	tier := p.tiers[idx]
	needCount := target - len(tier.segments)
	if needCount <= 0 {
		return
	}

	// Release lock before IO operation
	p.mu.Unlock()
	segs, err := p.allocateSequential(tier.cfg.Size, needCount, false)
	p.mu.Lock()

	if err != nil {
		log.Warnf("[RangePool] %s tier %d refill failed: %v", p.name, tier.cfg.Size, err)
		return
	}
	p.tiers[idx].segments = append(p.tiers[idx].segments, segs...)
}

// refillMiddleTierUnsafe refills a middle tier by splitting from upper tiers.
// Caller must hold write lock (mu.Lock).
func (p *tierPartition) refillMiddleTierUnsafe(idx int, target int) {
	tier := p.tiers[idx]
	maxAttempts := target * 2 // Avoid infinite loop

	for attempt := 0; attempt < maxAttempts && len(tier.segments) < target; attempt++ {
		if !p.splitFromUpperUnsafe(idx) {
			// Upper tier exhausted, try to refill upper tier
			if idx+1 < len(p.tiers) {
				upperTarget := p.tiers[idx+1].cfg.MaxCount
				if upperTarget <= 0 {
					upperTarget = p.tiers[idx+1].cfg.Threshold + 1
				}
				p.refillMiddleTierUnsafe(idx+1, upperTarget) // Recursive refill

				// Retry splitting after upper refill
				if !p.splitFromUpperUnsafe(idx) {
					break
				}
			} else {
				break
			}
		}
	}
}

// splitFromUpperUnsafe splits one segment from upper tier without acquiring lock.
// Caller must hold write lock (mu.Lock).
func (p *tierPartition) splitFromUpperUnsafe(idx int) bool {
	upperIdx := idx + 1
	if upperIdx >= len(p.tiers) {
		return false
	}
	upper := p.tiers[upperIdx]
	if len(upper.segments) == 0 {
		return false
	}
	lowerSize := p.tiers[idx].cfg.Size
	if lowerSize == 0 || upper.cfg.Size%lowerSize != 0 {
		return false
	}
	ratio := int(upper.cfg.Size / lowerSize)
	if ratio <= 1 {
		return false
	}

	// Use randMu to protect rand access
	p.randMu.Lock()
	sel := p.rand.Intn(len(upper.segments))
	p.randMu.Unlock()

	parent := upper.segments[sel]
	upper.segments[sel] = upper.segments[len(upper.segments)-1]
	upper.segments = upper.segments[:len(upper.segments)-1]

	childSize := lowerSize
	cursor := parent.Start
	for i := 0; i < ratio; i++ {
		child := IntRange{Start: cursor, End: cursor + childSize - 1}
		p.tiers[idx].segments = append(p.tiers[idx].segments, child)
		cursor += childSize
	}
	return true
}

// Close stops the background refiller goroutine and waits for it to finish.
func (p *tierPartition) Close() error {
	if p.cancel != nil {
		p.cancel()
	}
	if p.stopChan != nil {
		close(p.stopChan)
	}
	p.wg.Wait()
	return nil
}
