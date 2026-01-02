package iquery

import "fmt"

// IntRange is inclusive.
type IntRange struct {
	Start int64
	End   int64
}

func (r IntRange) Len() int64 {
	if r.End < r.Start {
		return 0
	}
	return r.End - r.Start + 1
}

func (r IntRange) Valid() bool { return r.Len() > 0 }

// rangePool manages ranges for a single sequencer key (e.g. one table primary key).
// It stores only segment metadata, not per-row ids.
type rangePool struct {
	existMin int64
	existMax int64
	insertHi int64 // last known high-water (inserted max) from lookup

	// live ranges represent ids that exist (inserted and not deleted).
	live []IntRange
	// free ranges represent ids that were deleted and can be reused safely.
	free []IntRange
	// pendingInsert ranges reserved for insert but not yet committed as "exists".
	pendingInsert []IntRange
	// pendingDelete ranges reserved for delete but not yet committed as "deleted".
	pendingDelete []IntRange
}

func newRangePool(existMin, existMax, insertHi int64) *rangePool {
	return &rangePool{
		existMin: existMin,
		existMax: existMax,
		insertHi: insertHi,
	}
}

func (p *rangePool) reserveFromFreeForInsert(n int64) (IntRange, bool) {
	if n <= 0 {
		n = 1
	}
	for i := range p.free {
		r := p.free[i]
		if !r.Valid() {
			continue
		}
		if r.Len() < n {
			continue
		}
		out := IntRange{Start: r.Start, End: r.Start + n - 1}
		rest := IntRange{Start: out.End + 1, End: r.End}
		if rest.Valid() {
			p.free[i] = rest
		} else {
			p.free = append(p.free[:i], p.free[i+1:]...)
		}
		p.pendingInsert = append(p.pendingInsert, out)
		return out, true
	}
	return IntRange{}, false
}

func (p *rangePool) markPendingInsert(r IntRange) {
	if !r.Valid() {
		return
	}
	p.pendingInsert = append(p.pendingInsert, r)
}

func (p *rangePool) commitInsert(r IntRange) error {
	if !r.Valid() {
		return nil
	}
	var found bool
	p.pendingInsert, found = removeExactRange(p.pendingInsert, r)
	if !found {
		return fmt.Errorf("pending insert range not found: [%d,%d]", r.Start, r.End)
	}
	p.addLive(r)
	return nil
}

func (p *rangePool) rollbackInsert(r IntRange) error {
	if !r.Valid() {
		return nil
	}
	var found bool
	p.pendingInsert, found = removeExactRange(p.pendingInsert, r)
	if !found {
		return fmt.Errorf("pending insert range not found: [%d,%d]", r.Start, r.End)
	}
	p.free = append(p.free, r)
	return nil
}

func (p *rangePool) addLive(r IntRange) {
	if !r.Valid() {
		return
	}
	p.live = append(p.live, r)
	if p.existMin == 0 || r.Start < p.existMin {
		p.existMin = r.Start
	}
	if r.End > p.existMax {
		p.existMax = r.End
	}
	if r.End > p.insertHi {
		p.insertHi = r.End
	}
}

func (p *rangePool) reserveDelete(r IntRange) error {
	if !r.Valid() {
		return nil
	}
	// We require deletes to be segment-accurate: r must be fully contained within one live range.
	for i := range p.live {
		l := p.live[i]
		if r.Start < l.Start || r.End > l.End {
			continue
		}
		// Remove r from l, potentially splitting.
		left := IntRange{Start: l.Start, End: r.Start - 1}
		right := IntRange{Start: r.End + 1, End: l.End}
		newLive := make([]IntRange, 0, len(p.live)+1)
		newLive = append(newLive, p.live[:i]...)
		if left.Valid() {
			newLive = append(newLive, left)
		}
		if right.Valid() {
			newLive = append(newLive, right)
		}
		newLive = append(newLive, p.live[i+1:]...)
		p.live = newLive
		p.pendingDelete = append(p.pendingDelete, r)
		return nil
	}
	return fmt.Errorf("delete range not found in live pool: [%d,%d]", r.Start, r.End)
}

func (p *rangePool) takeFromLive(n int64) (IntRange, bool) {
	if n <= 0 {
		n = 1
	}
	for _, l := range p.live {
		if !l.Valid() {
			continue
		}
		if l.Len() < n {
			continue
		}
		return IntRange{Start: l.Start, End: l.Start + n - 1}, true
	}
	return IntRange{}, false
}

func (p *rangePool) commitDelete(r IntRange) error {
	if !r.Valid() {
		return nil
	}
	var found bool
	p.pendingDelete, found = removeExactRange(p.pendingDelete, r)
	if !found {
		return fmt.Errorf("pending delete range not found: [%d,%d]", r.Start, r.End)
	}
	p.free = append(p.free, r)
	return nil
}

func (p *rangePool) rollbackDelete(r IntRange) error {
	if !r.Valid() {
		return nil
	}
	var found bool
	p.pendingDelete, found = removeExactRange(p.pendingDelete, r)
	if !found {
		return fmt.Errorf("pending delete range not found: [%d,%d]", r.Start, r.End)
	}
	p.addLive(r)
	return nil
}

func removeExactRange(list []IntRange, r IntRange) ([]IntRange, bool) {
	for i := range list {
		if list[i].Start == r.Start && list[i].End == r.End {
			return append(list[:i], list[i+1:]...), true
		}
	}
	return list, false
}
