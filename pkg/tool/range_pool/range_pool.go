package range_pool

import (
	"fmt"
	"math/rand"
	"sort"
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
	if err = pool.live.bootstrap(IntRange{}); err != nil {
		return nil, err
	}
	if err = pool.free.bootstrap(IntRange{}); err != nil {
		return nil, err
	}
	return pool, nil
}

func (p *RangePool) DebugLog() {
	ls, lc, le, ll := p.live.windowState()
	fs, fc, fe, fl := p.free.windowState()
	log.Infof("[RangePool] live window start=%d cursor=%d end=%d loop=%v", ls, lc, le, ll)
	log.Infof("[RangePool] free window start=%d cursor=%d end=%d loop=%v", fs, fc, fe, fl)
	log.Infof("[RangePool] live tiers ...")
	for _, v := range p.live.tiers {
		log.Infof("size:%v threshold:%d maxCount:%d", v.cfg.Size, v.cfg.Threshold, v.cfg.MaxCount)
		for index, tmp := range v.segments {
			log.Infof("index: %d %d~%d", index, tmp.Start, tmp.End)
		}
	}
	log.Infof("[RangePool] free tiers ...")
	for _, v := range p.free.tiers {
		log.Infof("size:%v threshold:%d maxCount:%d", v.cfg.Size, v.cfg.Threshold, v.cfg.MaxCount)
		for index, tmp := range v.segments {
			log.Infof("index: %d %d~%d", index, tmp.Start, tmp.End)
		}
	}
}

// ReserveFromFreeForInsert pulls a segment from the free tiers and immediately
// returns it for INSERT usage.
func (p *RangePool) ReserveFromFreeForInsert(size int64) (IntRange, bool) {
	r, err := p.free.consume(size)
	if err != nil {
		return IntRange{}, false
	}
	return r, true
}

// AddLiveRange injects a range into the live partition.
func (p *RangePool) AddLiveRange(r IntRange) error {
	if !r.Valid() {
		return nil
	}
	if err := p.live.addRange(r); err != nil {
		return err
	}
	return nil
}

// AddFreeRange injects a range into the free partition.
func (p *RangePool) AddFreeRange(r IntRange) error {
	if !r.Valid() {
		return nil
	}
	return p.free.addRange(r)
}

// TakeFromLive returns a segment from the live tiers without consuming it.
func (p *RangePool) TakeFromLive(size int64) (IntRange, bool) {
	r, err := p.live.peek(size)
	if err != nil {
		return IntRange{}, false
	}
	return r, true
}

// ReserveDelete removes r from live tiers and recycles it into free tiers.
func (p *RangePool) ReserveDelete(r IntRange) error {
	if !r.Valid() {
		return nil
	}
	if !p.live.removeExact(r) {
		return fmt.Errorf("delete range not found in live pool: [%d,%d]", r.Start, r.End)
	}
	return nil
}

// ExistingRange provides a best-effort range using live partition state.
//
// It prefers the current minimum remaining live segment start, falling back to the live allocation window.
func (p *RangePool) ExistingRange(n int64) (IntRange, bool) {
	if n <= 0 {
		n = 1
	}
	minv, maxv, ok := p.live.remainingBounds()
	if !ok {
		ws, _, we, _ := p.live.windowState()
		window := IntRange{Start: ws, End: we}
		if !window.Valid() {
			return IntRange{}, false
		}
		minv, maxv = window.Start, window.End
	}
	end := minv + n - 1
	if end > maxv {
		end = maxv
	}
	r := IntRange{Start: minv, End: end}
	if !r.Valid() {
		return IntRange{}, false
	}
	return r, true
}

// LiveMin returns the smallest remaining value in the live tiers.
func (p *RangePool) LiveMin() int64 {
	start, _, ok := p.live.remainingBounds()
	if !ok {
		return 0
	}
	return start
}

// FreeMin returns the smallest remaining value in the free tiers.
func (p *RangePool) FreeMin() int64 {
	start, _, ok := p.free.remainingBounds()
	if !ok {
		return 0
	}
	return start
}

// LiveWindow returns (start, cursor, end, enableLoop) for the live allocation window.
func (p *RangePool) LiveWindow() (int64, int64, int64, bool) { return p.live.windowState() }

// FreeWindow returns (start, cursor, end, enableLoop) for the free allocation window.
func (p *RangePool) FreeWindow() (int64, int64, int64, bool) { return p.free.windowState() }

// tierPartition represents the tier hierarchy backing either the live or free pool.
type tierPartition struct {
	name   string
	tiers  []*tierState
	idx    map[int64]int
	rand   *rand.Rand
	refill RangePoolRefillFunc

	windowStart int64
	windowEnd   int64
	cursor      int64
	enableLoop  bool
}

type tierState struct {
	cfg      TierConfig
	segments []IntRange
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
	return p.windowStart, p.cursor, p.windowEnd, p.enableLoop
}

func (p *tierPartition) remainingBounds() (minv, maxv int64, ok bool) {
	for _, tier := range p.tiers {
		for _, seg := range tier.segments {
			if !seg.Valid() {
				continue
			}
			if minv == 0 || seg.Start < minv {
				minv = seg.Start
			}
			if seg.End > maxv {
				maxv = seg.End
			}
		}
	}
	if maxv < minv || maxv == 0 {
		return 0, 0, false
	}
	return minv, maxv, true
}

func (p *tierPartition) bootstrap(seed IntRange) error {
	needInit := p.initNeedCapacity()
	if p.refill == nil && !seed.Valid() {
		return nil
	}

	if seed.Valid() {
		p.windowStart = seed.Start
		p.windowEnd = seed.End
		p.cursor = seed.Start
	}

	if needInit <= 0 {
		return nil
	}

	if err := p.ensureWindowCapacity(needInit, true); err != nil {
		return err
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
		window := IntRange{Start: p.windowStart, End: p.windowEnd}
		if !window.Valid() {
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

	// Initialize window if needed.
	window := IntRange{Start: p.windowStart, End: p.windowEnd}
	if !window.Valid() {
		enableLoop, win, err := p.refill(p.name, need)
		if err != nil {
			return err
		}
		if !win.Valid() {
			return fmt.Errorf("%s refill returned invalid window", p.name)
		}
		p.enableLoop = p.enableLoop || enableLoop
		p.windowStart = win.Start
		p.windowEnd = win.End
		if p.cursor == 0 {
			p.cursor = win.Start
		}
		return nil
	}

	for (IntRange{Start: p.windowStart, End: p.windowEnd}).Len() < need {
		missing := need - (IntRange{Start: p.windowStart, End: p.windowEnd}).Len()
		enableLoop, win, err := p.refill(p.name, missing)
		if err != nil {
			return err
		}
		if !win.Valid() {
			return fmt.Errorf("%s refill returned invalid window", p.name)
		}
		if win.Start != p.windowStart {
			return fmt.Errorf("%s refill window start changed: %d -> %d", p.name, p.windowStart, win.Start)
		}
		p.enableLoop = p.enableLoop || enableLoop
		if win.End > p.windowEnd {
			p.windowEnd = win.End
		} else if bootstrap {
			// In bootstrap, lack of growth means we cannot satisfy the requested capacity.
			break
		}
	}
	return nil
}

func (p *tierPartition) allocateSequential(size int64, count int, bootstrap bool) ([]IntRange, error) {
	if count <= 0 {
		return nil, nil
	}
	if size <= 0 {
		return nil, fmt.Errorf("%s allocate invalid size %d", p.name, size)
	}
	if !(IntRange{Start: p.windowStart, End: p.windowEnd}).Valid() {
		return nil, fmt.Errorf("%s allocation window not initialized", p.name)
	}
	if (IntRange{Start: p.windowStart, End: p.windowEnd}).Len() < size {
		return nil, fmt.Errorf("%s allocation window too small for size=%d", p.name, size)
	}

	segs := make([]IntRange, 0, count)
	for i := 0; i < count; i++ {
		end := p.cursor + size - 1
		if end > p.windowEnd {
			needExtra := end - p.windowEnd
			if p.refill != nil {
				enableLoop, win, err := p.refill(p.name, needExtra)
				if err != nil {
					return nil, err
				}
				if !win.Valid() {
					return nil, fmt.Errorf("%s refill returned invalid window", p.name)
				}
				if win.Start != p.windowStart {
					return nil, fmt.Errorf("%s refill window start changed: %d -> %d", p.name, p.windowStart, win.Start)
				}
				p.enableLoop = p.enableLoop || enableLoop
				if win.End > p.windowEnd {
					p.windowEnd = win.End
				}
			}
		}
		end = p.cursor + size - 1
		if end > p.windowEnd {
			if bootstrap {
				return nil, fmt.Errorf("%s allocation window exhausted during bootstrap: cursor=%d size=%d end=%d windowEnd=%d", p.name, p.cursor, size, end, p.windowEnd)
			}
			if !p.enableLoop {
				return nil, &partitionError{name: p.name, size: size}
			}
			p.cursor = p.windowStart
			end = p.cursor + size - 1
			if end > p.windowEnd {
				return nil, fmt.Errorf("%s loop enabled but window too small: size=%d windowLen=%d", p.name, size, (IntRange{Start: p.windowStart, End: p.windowEnd}).Len())
			}
		}
		seg := IntRange{Start: p.cursor, End: end}
		segs = append(segs, seg)
		p.cursor = end + 1
	}
	return segs, nil
}

// peek returns a random segment from the requested tier, refilling/splitting if needed.
func (p *tierPartition) peek(size int64) (IntRange, error) {
	idx, ok := p.idx[size]
	if !ok {
		return IntRange{}, fmt.Errorf("%s tier %d not configured", p.name, size)
	}
	if err := p.ensure(idx); err != nil {
		return IntRange{}, err
	}
	tier := p.tiers[idx]
	if len(tier.segments) == 0 {
		return IntRange{}, fmt.Errorf("%s tier %d exhausted", p.name, size)
	}
	sel := p.rand.Intn(len(tier.segments))
	return tier.segments[sel], nil
}

// consume removes and returns a random segment from the requested tier, refilling/splitting if needed.
func (p *tierPartition) consume(size int64) (IntRange, error) {
	idx, ok := p.idx[size]
	if !ok {
		return IntRange{}, fmt.Errorf("%s tier %d not configured", p.name, size)
	}
	if err := p.ensure(idx); err != nil {
		return IntRange{}, err
	}
	tier := p.tiers[idx]
	if len(tier.segments) == 0 {
		return IntRange{}, fmt.Errorf("%s tier %d exhausted", p.name, size)
	}
	sel := p.rand.Intn(len(tier.segments))
	r := tier.segments[sel]
	tier.segments[sel] = tier.segments[len(tier.segments)-1]
	tier.segments = tier.segments[:len(tier.segments)-1]
	return r, nil
}

// removeExact removes the exact matching segment from the tier that corresponds to r.Len().
func (p *tierPartition) removeExact(r IntRange) bool {
	idx, ok := p.idx[r.Len()]
	if !ok {
		return false
	}
	tier := p.tiers[idx]
	for i := range tier.segments {
		if tier.segments[i].Start == r.Start && tier.segments[i].End == r.End {
			tier.segments[i] = tier.segments[len(tier.segments)-1]
			tier.segments = tier.segments[:len(tier.segments)-1]
			return true
		}
	}
	return false
}

// ensure makes sure tier idx has at least one segment available for use.
// It tries (1) splitting from upper tiers (bigger segments) and then (2) calling refill.
func (p *tierPartition) ensure(idx int) error {
	tier := p.tiers[idx]
	if len(tier.segments) > tier.cfg.Threshold {
		return nil
	}
	if idx == len(p.tiers)-1 {
		// Only the top tier can grow from the allocation window; if the window is not
		// configured, this partition only relies on injected segments.
		window := IntRange{Start: p.windowStart, End: p.windowEnd}
		if window.Valid() || p.refill != nil {
			target := tier.cfg.MaxCount
			if target <= 0 {
				target = tier.cfg.Threshold + 1
			}
			needCount := target - len(tier.segments)
			if needCount <= 0 {
				needCount = 1
			}
			segs, err := p.allocateSequential(tier.cfg.Size, needCount, false)
			if err != nil {
				return err
			}
			tier.segments = append(tier.segments, segs...)
		}
		return nil
	}
	if p.splitFromUpper(idx) {
		return nil
	}
	if idx+1 < len(p.tiers) {
		if err := p.ensure(idx + 1); err == nil {
			if p.splitFromUpper(idx) {
				return nil
			}
		} else if err != nil {
			if _, upperErr := err.(*partitionError); !upperErr {
				return err
			}
		}
	}
	if len(tier.segments) == 0 {
		return &partitionError{name: p.name, size: tier.cfg.Size}
	}
	return nil
}

// partitionError indicates a tier is unavailable after ensure() attempts (split/refill).
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
	sel := p.rand.Intn(len(upper.segments))
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

// addRange splits the input range into tier-sized segments (greedy from largest to smallest)
// and stores those segments in their corresponding tiers.
func (p *tierPartition) addRange(r IntRange) error {
	if !r.Valid() {
		return nil
	}
	remain := r.Len()
	cursor := r.Start
	for i := len(p.tiers) - 1; i >= 0; i-- {
		size := p.tiers[i].cfg.Size
		for remain >= size {
			seg := IntRange{Start: cursor, End: cursor + size - 1}
			p.tiers[i].segments = append(p.tiers[i].segments, seg)
			cursor += size
			remain -= size
		}
	}
	if remain != 0 {
		return fmt.Errorf("range length %d not aligned with tier sizes", r.Len())
	}
	return nil
}

// addExactRange appends r as-is to the tier that matches r.Len().
func (p *tierPartition) addExactRange(r IntRange) error {
	idx, ok := p.idx[r.Len()]
	if !ok {
		return fmt.Errorf("%s tier %d not configured", p.name, r.Len())
	}
	p.tiers[idx].segments = append(p.tiers[idx].segments, r)
	return nil
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
