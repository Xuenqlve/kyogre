package range_pool

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/xuenqlve/common/log"
)

// tierPartition represents the tier hierarchy backing either the live or free pool.
type tierPartition struct {
	name string

	window    *windowManager
	tiers     *tierManager
	scheduler *refillScheduler

	triggerFactor int64
	refillFactor  int64

	mu        sync.RWMutex
	closed    bool
	closedErr error
	closeOnce sync.Once

	refillSignal chan int
	stopChan     chan struct{}
	wg           sync.WaitGroup
	ctx          context.Context
	cancel       context.CancelFunc
}

// newTierPartition 构建一个分区(tierPartition)，并按 Size 升序组织 tiers，同时建立 size->tier 索引表用于快速定位。
func newTierPartition(name string, tiers []TierConfig, refill RangePoolRefillFunc, rnd *rand.Rand) (*tierPartition, error) {
	tm, err := newTierManager(name, tiers, rnd)
	if err != nil {
		return nil, err
	}
	wm := newWindowManager(name)
	scheduler := newRefillScheduler(name, refill, wm)

	return &tierPartition{
		name:      name,
		window:    wm,
		tiers:     tm,
		scheduler: scheduler,
	}, nil
}

// window state is managed by windowManager.

func (p *tierPartition) checkClosed() error {
	p.mu.RLock()
	closed := p.closed
	err := p.closedErr
	p.mu.RUnlock()
	if !closed {
		return nil
	}
	if err == nil {
		return fmt.Errorf("%s partition closed", p.name)
	}
	return err
}

func (p *tierPartition) closeWithError(err error) {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return
	}
	p.closed = true
	p.closedErr = err
	p.mu.Unlock()
	_ = p.Close()
}

// pickTierSize 根据 need 选择一个可用的 tier size（选择满足 Size>=need 的最小 Size）。
func (p *tierPartition) pickTierSize(need int64) (int64, bool) {
	return p.tiers.pickTierSize(need)
}

func (p *tierPartition) setRefillFactors(triggerFactor, refillFactor int64) {
	if triggerFactor <= 0 {
		triggerFactor = 3
	}
	if refillFactor <= 0 {
		refillFactor = 5
	}
	if refillFactor <= triggerFactor {
		refillFactor = triggerFactor + 2
	}
	p.triggerFactor = triggerFactor
	p.refillFactor = refillFactor
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

	for i := 0; i < p.tiers.tiersCount(); i++ {
		cfg, ok := p.tiers.tierConfig(i)
		if !ok || cfg.MaxCount <= 0 {
			continue
		}
		segs, err := p.allocateSequential(cfg.Size, cfg.MaxCount, true)
		if err != nil {
			return err
		}
		p.tiers.appendSegments(i, segs)
	}
	return nil
}

// initNeedCapacity 计算“初始化时填满所有 tier 到 MaxCount”所需的总容量（Σ size_i * maxCount_i）。
func (p *tierPartition) initNeedCapacity() int64 {
	var need int64
	for i := 0; i < p.tiers.tiersCount(); i++ {
		cfg, ok := p.tiers.tierConfig(i)
		if !ok || cfg.MaxCount <= 0 {
			continue
		}
		need += cfg.Size * int64(cfg.MaxCount)
	}
	return need
}

// ensureWindowCapacity 确保当前分配窗口至少具备 need 的容量。
func (p *tierPartition) ensureWindowCapacity(need int64, bootstrap bool) error {
	if err := p.checkClosed(); err != nil {
		return err
	}
	if p.scheduler == nil || p.scheduler.refill == nil {
		state := p.window.stateSnapshot()
		window := IntRange{Start: state.start, End: state.end}
		if !state.init || !window.Valid() {
			return fmt.Errorf("%s partition requires either a valid seed window or a refill callback", p.name)
		}
		if window.Len() < need {
			return fmt.Errorf("%s allocation window capacity insufficient: need=%d have=%d", p.name, need, window.Len())
		}
		return nil
	}
	return p.window.ensureCapacity(need, bootstrap, p.refillWindow)
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
	if err := p.checkClosed(); err != nil {
		return nil, err
	}
	return p.window.allocateSequential(size, count, bootstrap, p.refillWindow)
}

// peek 从指定 size 的 tier 中随机选择一个 segment 返回，但不消费库存（用于 UPDATE 场景）。
// 注意：按约定 peek 不触发补货逻辑（补货由 consume 后置触发或后台定时器兜底）。
func (p *tierPartition) peek(size int64) (IntRange, error) {
	if err := p.checkClosed(); err != nil {
		return IntRange{}, err
	}
	return p.tiers.peek(size)
}

// consume 从指定 size 的 tier 中随机消费一个 segment 并返回（用于 INSERT/DELETE 场景）。
// 消费后若库存触达阈值(<=Threshold)，会尝试发送异步补货信号，由后台 refiller 补到 MaxCount。
func (p *tierPartition) consume(size int64) (IntRange, error) {
	if err := p.checkClosed(); err != nil {
		return IntRange{}, err
	}
	idx, ok := p.tiers.tierIndex(size)
	if !ok {
		return IntRange{}, fmt.Errorf("%s tier %d not configured", p.name, size)
	}

	emergency := func(tierIdx int, target int) int {
		var added int
		if tierIdx == p.tiers.tiersCount()-1 {
			added = p.refillTopTier(tierIdx, target)
		} else {
			added = p.refillMiddleTier(tierIdx, target)
		}
		return added
	}

	r, needRefill, err := p.tiers.consume(size, emergency)
	if err != nil {
		return IntRange{}, err
	}

	// 3. Post-consumption: ensure critical tiers refill synchronously to avoid empty inventory.
	if needRefill {
		if idx == p.tiers.tiersCount()-1 {
			p.checkAndRefill(idx)
		} else if p.refillSignal != nil {
			select {
			case p.refillSignal <- idx:
				// Signal sent successfully
			default:
				// Channel full: fall back to sync refill to avoid empty tiers.
				p.checkAndRefill(idx)
			}
		}
	}

	// 4. Proactive window refill when top tier is close to window end.
	if idx == p.tiers.tiersCount()-1 {
		if cfg, ok := p.tiers.tierConfig(idx); ok {
			p.maybeTriggerWindowRefill(cfg.Size)
		}
	}

	return r, nil
}

// maybeTriggerWindowRefill 在顶层 tier 剩余空间不足时触发窗口补货（同步）。
func (p *tierPartition) maybeTriggerWindowRefill(topSize int64) {
	state := p.window.stateSnapshot()
	if !state.init {
		return
	}
	remaining := state.end - state.cursor
	if remaining < 0 {
		remaining = 0
	}
	triggerFactor := p.triggerFactor
	if triggerFactor <= 0 {
		triggerFactor = 2
	}
	if remaining > topSize*triggerFactor {
		return
	}
	refillFactor := p.refillFactor
	if refillFactor <= 0 {
		refillFactor = 4
	}
	need := topSize * refillFactor
	if need <= 0 {
		need = topSize
	}
	_ = p.refillWindow(need, "near-window-end", true)
}

func (p *tierPartition) fireAndForgetRefill(need int64, reason string) {
	_ = p.refillWindow(need, reason, false)
}

// refillWindow 发送 refill 请求，wait=true 时同步等待结果；否则仅触发异步补货。
func (p *tierPartition) refillWindow(need int64, reason string, wait bool) error {
	if err := p.checkClosed(); err != nil {
		return err
	}
	if p.scheduler == nil {
		return fmt.Errorf("%s refill worker not started", p.name)
	}
	return p.scheduler.schedule(need, reason, wait)
}

// ==================== Background Refiller Implementation ====================

// startRefillWorker 启动 IO refill 协程：串行执行 refill 回调，避免在业务路径中做 IO。
func (p *tierPartition) startRefillWorker(ctx context.Context) {
	if p.scheduler != nil {
		p.scheduler.start(ctx)
	}
}

// startRefiller 启动后台补货协程：按信号触发补货。
func (p *tierPartition) startRefiller(ctx context.Context) {
	if p.ctx == nil {
		p.ctx, p.cancel = context.WithCancel(ctx)
	}
	p.refillSignal = make(chan int, p.tiers.tiersCount()) // Buffered channel
	p.stopChan = make(chan struct{})
	p.wg.Add(1)
	go p.refillerLoop()
}

// refillerLoop 后台补货主循环：按信号补指定 tier。
func (p *tierPartition) refillerLoop() {
	defer p.wg.Done()

	for {
		select {
		case <-p.ctx.Done():
			return
		case <-p.stopChan:
			return
		case idx := <-p.refillSignal:
			// Triggered refill for specific tier
			p.checkAndRefill(idx)
		}
	}
}

// checkAndRefill 检查指定 tier 是否需要补货（<=Threshold），若需要则补到 MaxCount。
// 中间层通过拆分上层补货；顶层通过分配窗口(window)+refill 扩展补货。
func (p *tierPartition) checkAndRefill(idx int) {
	if idx < 0 || idx >= p.tiers.tiersCount() {
		return
	}

	tierSnap, ok := p.tiers.tierSnapshot(idx)
	if !ok {
		return
	}

	curLen := len(tierSnap.segments)
	needRefill := curLen <= tierSnap.cfg.Threshold
	if !needRefill {
		return
	}

	target := tierSnap.cfg.MaxCount
	if target <= 0 {
		target = tierSnap.cfg.Threshold + 1
	}
	var added int
	if idx == p.tiers.tiersCount()-1 {
		added = p.refillTopTier(idx, target)
	} else {
		added = p.refillMiddleTier(idx, target)
	}

	if added > 0 {
		p.tiers.setLastRefill(idx, time.Now())
	}
}

// refillTopTier 补顶层 tier：从分配窗口顺序切分 segment，直到补到 target。
func (p *tierPartition) refillTopTier(idx int, target int) int {
	if err := p.checkClosed(); err != nil {
		return 0
	}
	cfg, ok := p.tiers.tierConfig(idx)
	if !ok {
		return 0
	}
	alloc := func(size int64, count int) ([]IntRange, error) {
		segs, err := p.window.allocateSequential(size, count, false, p.refillWindow)
		if err != nil {
			log.Warnf("[RangePool] %s tier %d refill failed: %v", p.name, cfg.Size, err)
		}
		return segs, err
	}
	return p.tiers.refillTopTier(idx, target, alloc)
}

// refillMiddleTier 补中间层/底层 tier：通过不断从上层拆分 segment 来补到 target。
func (p *tierPartition) refillMiddleTier(idx int, target int) int {
	return p.tiers.refillMiddleTier(idx, target)
}

// Close 停止后台补货协程并等待退出（需由上层保证只调用一次，避免重复 close channel）。
func (p *tierPartition) Close() error {
	p.closeOnce.Do(func() {
		p.mu.Lock()
		if !p.closed {
			p.closed = true
		}
		p.mu.Unlock()
		if p.cancel != nil {
			p.cancel()
		}
		if p.stopChan != nil {
			close(p.stopChan)
		}
		if p.scheduler != nil {
			p.scheduler.stop()
		}
	})
	p.wg.Wait()
	return nil
}
