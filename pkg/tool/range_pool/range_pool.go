package range_pool

import (
	"fmt"
	"math/rand"
	"sort"
	"time"
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
// and a "free" partition (insert consumers). Reserve operations immediately
// transition ranges between partitions without requiring explicit commit/rollback.
type RangePool struct {
	cfg RangePoolConfig

	live *tierPartition
	free *tierPartition

	existMin int64
	existMax int64
	insertHi int64
}

// RangePoolRefillFunc is invoked when a tier needs additional segments.
type RangePoolRefillFunc func(partition string, tierSize int64, deficit int) ([]IntRange, error)

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

	LiveRefill RangePoolRefillFunc
	FreeRefill RangePoolRefillFunc

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
func WithLiveRefill(fn RangePoolRefillFunc) RangePoolOption {
	return func(cfg *RangePoolConfig) { cfg.LiveRefill = fn }
}

// WithFreeRefill sets the free partition refill callback.
func WithFreeRefill(fn RangePoolRefillFunc) RangePoolOption {
	return func(cfg *RangePoolConfig) { cfg.FreeRefill = fn }
}

// WithRandSource overrides the random source used when selecting segments.
func WithRandSource(src rand.Source) RangePoolOption {
	return func(cfg *RangePoolConfig) { cfg.RandSource = src }
}

// NewRangePool builds a RangePool seeded with existing ranges.
func NewRangePool(existMin, existMax, insertHi int64, opts ...RangePoolOption) (*RangePool, error) {
	cfg := defaultRangePoolConfig()
	for _, opt := range opts {
		opt(&cfg)
	}
	if err := cfg.ValidateAndSetDefault(); err != nil {
		return nil, err
	}
	rnd := cfg.rand()
	live, err := newTierPartition("live", cfg.LiveTiers, cfg.LiveRefill, rnd)
	if err != nil {
		return nil, err
	}
	free, err := newTierPartition("free", cfg.FreeTiers, cfg.FreeRefill, rnd)
	if err != nil {
		return nil, err
	}
	pool := &RangePool{
		cfg:      cfg,
		live:     live,
		free:     free,
		existMin: existMin,
		existMax: existMax,
		insertHi: insertHi,
	}
	if existMax >= existMin && existMax > 0 {
		if err := pool.live.addRange(IntRange{Start: existMin, End: existMax}); err != nil {
			return nil, err
		}
	}
	if insertHi < existMax {
		pool.insertHi = existMax
	}
	return pool, nil
}

// ReserveFromFreeForInsert pulls a segment from the free tiers and immediately
// records it as live.
func (p *RangePool) ReserveFromFreeForInsert(size int64) (IntRange, bool) {
	r, err := p.free.consume(size)
	if err != nil {
		return IntRange{}, false
	}
	_ = p.live.addExactRange(r)
	p.bumpBounds(r)
	if r.End > p.insertHi {
		p.insertHi = r.End
	}
	return r, true
}

// AddLiveRange registers a freshly allocated range (e.g. from high-water insert).
func (p *RangePool) AddLiveRange(r IntRange) error {
	if !r.Valid() {
		return nil
	}
	if err := p.live.addRange(r); err != nil {
		return err
	}
	p.bumpBounds(r)
	if r.End > p.insertHi {
		p.insertHi = r.End
	}
	return nil
}

// AddFreeRange injects a range into the free partition (used for seeding/testing).
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
	_ = p.free.addExactRange(r)
	p.recalcLiveBounds()
	return nil
}

// ExistingRange provides a best-effort range using known min/max bounds.
func (p *RangePool) ExistingRange(n int64) (IntRange, bool) {
	if n <= 0 {
		n = 1
	}
	if p.existMax < p.existMin || p.existMax <= 0 {
		return IntRange{}, false
	}
	start := p.existMin
	end := start + n - 1
	if end > p.existMax {
		end = p.existMax
	}
	r := IntRange{Start: start, End: end}
	if !r.Valid() {
		return IntRange{}, false
	}
	return r, true
}

// ExistMin returns the smallest known live value.
func (p *RangePool) ExistMin() int64 { return p.existMin }

// ExistMax returns the largest known live value.
func (p *RangePool) ExistMax() int64 { return p.existMax }

// InsertHi returns the highest id ever reserved for insert.
func (p *RangePool) InsertHi() int64 { return p.insertHi }

func (p *RangePool) bumpBounds(r IntRange) {
	if p.existMin == 0 || r.Start < p.existMin {
		p.existMin = r.Start
	}
	if r.End > p.existMax {
		p.existMax = r.End
	}
}

func (p *RangePool) recalcLiveBounds() {
	var minv, maxv int64
	for _, tier := range p.live.tiers {
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
	p.existMin = minv
	p.existMax = maxv
}

// tierPartition represents the tier hierarchy backing either the live or free pool.
type tierPartition struct {
	name   string
	tiers  []*tierState
	idx    map[int64]int
	rand   *rand.Rand
	refill RangePoolRefillFunc
}

type tierState struct {
	cfg      TierConfig
	segments []IntRange
}

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
		copy := tiers[i]
		tp.tiers = append(tp.tiers, &tierState{cfg: copy})
		index[copy.Size] = len(tp.tiers) - 1
	}
	return tp, nil
}

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

func (p *tierPartition) ensure(idx int) error {
	tier := p.tiers[idx]
	if len(tier.segments) > tier.cfg.Threshold {
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
	if p.refill != nil {
		need := tier.cfg.MaxCount - len(tier.segments)
		if need <= 0 {
			need = tier.cfg.Threshold - len(tier.segments) + 1
			if need <= 0 {
				need = 1
			}
		}
		segs, err := p.refill(p.name, tier.cfg.Size, need)
		if err != nil {
			return err
		}
		for _, seg := range segs {
			if seg.Len() != tier.cfg.Size {
				return fmt.Errorf("refill for %s tier %d returned len=%d", p.name, tier.cfg.Size, seg.Len())
			}
			tier.segments = append(tier.segments, seg)
		}
	}
	if len(tier.segments) == 0 {
		return &partitionError{name: p.name, size: tier.cfg.Size}
	}
	return nil
}

type partitionError struct {
	name string
	size int64
}

func (e *partitionError) Error() string {
	return fmt.Sprintf("%s tier %d unavailable", e.name, e.size)
}

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

func (p *tierPartition) addExactRange(r IntRange) error {
	idx, ok := p.idx[r.Len()]
	if !ok {
		return fmt.Errorf("%s tier %d not configured", p.name, r.Len())
	}
	p.tiers[idx].segments = append(p.tiers[idx].segments, r)
	return nil
}

func defaultRangePoolConfig() RangePoolConfig {
	tiers := []TierConfig{
		{Size: 1, Threshold: 2, MaxCount: 64},
		{Size: 3, Threshold: 2, MaxCount: 48},
		{Size: 5, Threshold: 2, MaxCount: 32},
		{Size: 10, Threshold: 1, MaxCount: 16},
		{Size: 20, Threshold: 1, MaxCount: 8},
		{Size: 50, Threshold: 1, MaxCount: 4},
		{Size: 100, Threshold: 1, MaxCount: 2},
	}
	return RangePoolConfig{LiveTiers: tiers, FreeTiers: tiers}
}

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
	return nil
}
