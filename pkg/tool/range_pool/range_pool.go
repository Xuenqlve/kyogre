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
	randMu   sync.Mutex // Protects rand for thread-safe random number generation

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

// newTierPartition 构建一个分区(tierPartition)，并按 Size 升序组织 tiers，同时建立 size->tier 索引表用于快速定位。
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

// windowState 返回当前滑动窗口(start/cursor/end)以及是否已进入回绕(enableLoop)。
func (p *tierPartition) windowState() (start, cursor, end int64, enableLoop bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.windowStart, p.cursor, p.windowEnd, p.enableLoop
}

// pickTierSize 根据 need 选择一个可用的 tier size（选择满足 Size>=need 的最小 Size）。
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

// bootstrap 在分区初始化阶段一次性填充各 tier 的初始库存（按 MaxCount）。
// 初始化窗口容量会按“总需求 * 2”预取，减少启动阶段的 refill 次数。
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

// initNeedCapacity 计算“初始化时填满所有 tier 到 MaxCount”所需的总容量（Σ size_i * maxCount_i）。
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

// ensureWindowCapacity 确保当前分配窗口至少具备 need 的容量。
// 该方法采用“两段式”策略：refill(IO) 在锁外执行，回写 window 状态时才加锁。
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

// allocateSequential 从当前 cursor 开始顺序切分出 count 个 size 长度的连续区间，并推进 cursor。
// 当 cursor 越界时会尝试通过 refill 扩展 windowEnd；若 enableLoop=true 则允许回绕到 windowStart。
// 该方法同样遵循“两段式”策略：refill(IO) 在锁外执行。
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

// peek 从指定 size 的 tier 中随机选择一个 segment 返回，但不消费库存（用于 UPDATE 场景）。
// 注意：按约定 peek 不触发补货逻辑（补货由 consume 后置触发或后台定时器兜底）。
func (p *tierPartition) peek(size int64) (IntRange, error) {
	idx, ok := p.idx[size]
	if !ok {
		return IntRange{}, fmt.Errorf("%s tier %d not configured", p.name, size)
	}

	p.mu.RLock()
	defer p.mu.RUnlock()

	tier := p.tiers[idx]
	if len(tier.segments) == 0 {
		return IntRange{}, fmt.Errorf("%s tier %d exhausted", p.name, size)
	}

	sel := p.randInt(len(tier.segments))

	r := tier.segments[sel]
	return r, nil
}

// randInt 线程安全地生成 [0, num) 的随机数（保护 rand.Rand）。
func (p *tierPartition) randInt(num int) int {
	p.randMu.Lock()
	defer p.randMu.Unlock()
	return p.rand.Intn(num)
}

// consume 从指定 size 的 tier 中随机消费一个 segment 并返回（用于 INSERT/DELETE 场景）。
// 消费后若库存触达阈值(<=Threshold)，会尝试发送异步补货信号，由后台 refiller 补到 MaxCount。
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
	sel := p.randInt(len(tier.segments))

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

// splitFromUpper 从上一级 tier(idx+1) 取出一个 segment，并按下一级 size 拆分成多个子 segment 放入当前 tier。
// 该方法会在内部自行加锁（对外暴露的安全版本）。
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

	sel := p.randInt(len(upper.segments))

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

// refillWindow 调用外部 refill 回调获取/扩展窗口（串行化回调，避免并发多次打 DB）。
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

// startRefiller 启动后台补货协程：支持定时扫描补货，以及按信号触发补货。
func (p *tierPartition) startRefiller(ctx context.Context) {
	p.ctx, p.cancel = context.WithCancel(ctx)
	p.refillSignal = make(chan int, len(p.tiers)) // Buffered channel
	p.stopChan = make(chan struct{})

	p.wg.Add(1)
	go p.refillerLoop()
}

// refillerLoop 后台补货主循环：定时检查各 tier 库存是否低于阈值，或按信号补指定 tier。
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

// checkAndRefillAll 遍历所有 tiers，逐个执行检查与补货。
func (p *tierPartition) checkAndRefillAll() {
	for i := range p.tiers {
		p.checkAndRefill(i)
	}
}

// checkAndRefill 检查指定 tier 是否需要补货（<=Threshold），若需要则补到 MaxCount。
// 中间层通过拆分上层补货；顶层通过分配窗口(window)+refill 扩展补货。
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

// refillTopTierUnsafe 补顶层 tier：从分配窗口顺序切分 segment，直到补到 target。
// 调用方必须已持有写锁；该方法会主动释放写锁以执行可能的 IO(refill)，随后再重新加锁写回。
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

// refillMiddleTierUnsafe 补中间层/底层 tier：通过不断从上层拆分 segment 来补到 target。
// 调用方必须已持有写锁；该方法不会做 IO。
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

// splitFromUpperUnsafe 为 splitFromUpper 的无锁版本：调用方必须已持有写锁。
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

// Close 停止后台补货协程并等待退出（需由上层保证只调用一次，避免重复 close channel）。
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
