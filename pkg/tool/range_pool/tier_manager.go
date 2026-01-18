package range_pool

import (
	"fmt"
	"math/rand"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

type tierState struct {
	cfg        TierConfig
	segments   []IntRange
	lastRefill time.Time
}

type tierManager struct {
	name  string
	tiers []*tierState
	idx   map[int64]int
	rand  *rand.Rand

	mu     sync.RWMutex
	randMu sync.Mutex

	emergency []uint32
}

func newTierManager(name string, tiers []TierConfig, rnd *rand.Rand) (*tierManager, error) {
	if len(tiers) == 0 {
		return nil, fmt.Errorf("%s partition requires at least one tier", name)
	}
	sort.Slice(tiers, func(i, j int) bool { return tiers[i].Size < tiers[j].Size })
	index := make(map[int64]int, len(tiers))

	tm := &tierManager{name: name, rand: rnd, idx: index}
	for i := range tiers {
		if tiers[i].Size <= 0 {
			return nil, fmt.Errorf("%s tier %d has invalid size", name, tiers[i].Size)
		}
		if tiers[i].Threshold < 0 {
			return nil, fmt.Errorf("%s tier %d threshold must be >= 0", name, tiers[i].Size)
		}
		copyCfg := tiers[i]
		tm.tiers = append(tm.tiers, &tierState{cfg: copyCfg})
		index[copyCfg.Size] = len(tm.tiers) - 1
	}
	tm.emergency = make([]uint32, len(tm.tiers))
	return tm, nil
}

func (t *tierManager) pickTierSize(need int64) (int64, bool) {
	if need <= 0 {
		need = 1
	}
	for _, tier := range t.tiers {
		if tier.cfg.Size >= need {
			return tier.cfg.Size, true
		}
	}
	return 0, false
}

func (t *tierManager) randInt(num int) int {
	t.randMu.Lock()
	defer t.randMu.Unlock()
	return t.rand.Intn(num)
}

func (t *tierManager) tierIndex(size int64) (int, bool) {
	idx, ok := t.idx[size]
	return idx, ok
}

func (t *tierManager) peek(size int64) (IntRange, error) {
	idx, ok := t.tierIndex(size)
	if !ok {
		return IntRange{}, fmt.Errorf("%s tier %d not configured", t.name, size)
	}

	t.mu.RLock()
	defer t.mu.RUnlock()

	tier := t.tiers[idx]
	if len(tier.segments) == 0 {
		return IntRange{}, fmt.Errorf("%s tier %d exhausted", t.name, size)
	}

	sel := t.randInt(len(tier.segments))
	return tier.segments[sel], nil
}

func (t *tierManager) consume(size int64, emergency func(idx int, target int) int) (IntRange, bool, error) {
	idx, ok := t.tierIndex(size)
	if !ok {
		return IntRange{}, false, fmt.Errorf("%s tier %d not configured", t.name, size)
	}

	t.mu.Lock()
	tier := t.tiers[idx]
	cfg := tier.cfg
	if len(tier.segments) == 0 {
		t.mu.Unlock()
		doEmergency := atomic.CompareAndSwapUint32(&t.emergency[idx], 0, 1)
		if doEmergency {
			target := cfg.MaxCount
			if target <= 0 {
				target = cfg.Threshold + 1
			}
			func() {
				defer atomic.StoreUint32(&t.emergency[idx], 0)
				emergency(idx, target)
			}()
		}
		t.mu.Lock()
		tier = t.tiers[idx]
	}

	if len(tier.segments) == 0 {
		t.mu.Unlock()
		return IntRange{}, false, fmt.Errorf("%s tier %d exhausted", t.name, size)
	}

	sel := t.randInt(len(tier.segments))
	r := tier.segments[sel]
	tier.segments[sel] = tier.segments[len(tier.segments)-1]
	tier.segments = tier.segments[:len(tier.segments)-1]

	needRefill := len(tier.segments) <= cfg.Threshold
	t.mu.Unlock()

	return r, needRefill, nil
}

func (t *tierManager) appendSegments(idx int, segs []IntRange) {
	if len(segs) == 0 {
		return
	}
	t.mu.Lock()
	t.tiers[idx].segments = append(t.tiers[idx].segments, segs...)
	t.mu.Unlock()
}

func (t *tierManager) splitFromUpperUnsafe(idx int) bool {
	upperIdx := idx + 1
	if upperIdx >= len(t.tiers) {
		return false
	}
	upper := t.tiers[upperIdx]
	if len(upper.segments) == 0 {
		return false
	}
	lowerSize := t.tiers[idx].cfg.Size
	if lowerSize == 0 || upper.cfg.Size%lowerSize != 0 {
		return false
	}
	ratio := int(upper.cfg.Size / lowerSize)
	if ratio <= 1 {
		return false
	}

	sel := t.randInt(len(upper.segments))
	parent := upper.segments[sel]
	upper.segments[sel] = upper.segments[len(upper.segments)-1]
	upper.segments = upper.segments[:len(upper.segments)-1]

	childSize := lowerSize
	cursor := parent.Start
	for i := 0; i < ratio; i++ {
		child := IntRange{Start: cursor, End: cursor + childSize - 1}
		t.tiers[idx].segments = append(t.tiers[idx].segments, child)
		cursor += childSize
	}
	return true
}

func (t *tierManager) refillMiddleTier(idx int, target int) int {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.refillMiddleTierUnsafe(idx, target)
}

func (t *tierManager) refillMiddleTierUnsafe(idx int, target int) int {
	tier := t.tiers[idx]
	beforeLen := len(tier.segments)
	maxAttempts := target * 2

	for attempt := 0; attempt < maxAttempts && len(tier.segments) < target; attempt++ {
		if !t.splitFromUpperUnsafe(idx) {
			if idx+1 < len(t.tiers) {
				upperTarget := t.tiers[idx+1].cfg.MaxCount
				if upperTarget <= 0 {
					upperTarget = t.tiers[idx+1].cfg.Threshold + 1
				}
				t.refillMiddleTierUnsafe(idx+1, upperTarget)
				if !t.splitFromUpperUnsafe(idx) {
					break
				}
			} else {
				break
			}
		}
	}
	return len(tier.segments) - beforeLen
}

func (t *tierManager) refillTopTier(idx int, target int, alloc func(size int64, count int) ([]IntRange, error)) int {
	t.mu.RLock()
	tier := t.tiers[idx]
	needCount := target - len(tier.segments)
	size := tier.cfg.Size
	t.mu.RUnlock()

	if needCount <= 0 {
		return 0
	}
	segs, err := alloc(size, needCount)
	if err != nil {
		return 0
	}
	t.appendSegments(idx, segs)
	return len(segs)
}

func (t *tierManager) tierSnapshot(idx int) (tierState, bool) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	if idx < 0 || idx >= len(t.tiers) {
		return tierState{}, false
	}
	return *t.tiers[idx], true
}

func (t *tierManager) tierConfig(idx int) (TierConfig, bool) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	if idx < 0 || idx >= len(t.tiers) {
		return TierConfig{}, false
	}
	return t.tiers[idx].cfg, true
}

func (t *tierManager) tierLen(idx int) int {
	t.mu.RLock()
	defer t.mu.RUnlock()
	if idx < 0 || idx >= len(t.tiers) {
		return 0
	}
	return len(t.tiers[idx].segments)
}

func (t *tierManager) setLastRefill(idx int, ts time.Time) {
	t.mu.Lock()
	if idx >= 0 && idx < len(t.tiers) {
		t.tiers[idx].lastRefill = ts
	}
	t.mu.Unlock()
}

func (t *tierManager) tiersCount() int {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return len(t.tiers)
}
