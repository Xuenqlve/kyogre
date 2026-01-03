package range_pool

import (
	"math/rand"
	"testing"
)

func TestRangePoolInsertAndRecycle(t *testing.T) {
	pool, err := NewRangePool(0, 0, 0, WithRandSource(rand.NewSource(1)))
	if err != nil {
		t.Fatalf("new range pool: %v", err)
	}
	if err := pool.AddFreeRange(IntRange{Start: 10, End: 14}); err != nil {
		t.Fatalf("add free range: %v", err)
	}

	r, ok := pool.ReserveFromFreeForInsert(5)
	if !ok {
		t.Fatalf("expected reserve insert to succeed")
	}
	if r != (IntRange{Start: 10, End: 14}) {
		t.Fatalf("unexpected insert range %#v", r)
	}

	take, ok := pool.TakeFromLive(5)
	if !ok || take != r {
		t.Fatalf("expected live peek to see inserted range, got %#v ok=%v", take, ok)
	}

	if err := pool.ReserveDelete(r); err != nil {
		t.Fatalf("reserve delete failed: %v", err)
	}

	reused, ok := pool.ReserveFromFreeForInsert(5)
	if !ok || reused != r {
		t.Fatalf("expected recycled range, got %#v ok=%v", reused, ok)
	}
}

func TestRangePoolAddLiveRangeAndExisting(t *testing.T) {
	pool, err := NewRangePool(0, 0, 0)
	if err != nil {
		t.Fatalf("new range pool: %v", err)
	}
	if err := pool.AddLiveRange(IntRange{Start: 100, End: 119}); err != nil {
		t.Fatalf("add live range: %v", err)
	}
	if pool.ExistMin() != 100 || pool.ExistMax() != 119 {
		t.Fatalf("unexpected bounds min=%d max=%d", pool.ExistMin(), pool.ExistMax())
	}
	r, ok := pool.ExistingRange(10)
	if !ok {
		t.Fatalf("expected existing range")
	}
	if r.Start != 100 || r.End != 109 {
		t.Fatalf("unexpected existing fallback %#v", r)
	}
}
