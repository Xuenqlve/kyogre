package tool

import (
	"math/rand"
	"sync"
	"testing"
	"time"

	"github.com/xuenqlve/common/log"
	"github.com/xuenqlve/kyogre/pkg/tool/range_pool"
)

type LiveRefill struct {
	Min int64
	Max int64

	LiveStart      int64
	LiveCursor     int64
	LiveEnableLoop bool

	FreeStart      int64
	FreeCursor     int64
	FreeEnableLoop bool
}

func (l *LiveRefill) Refill(partition string, req range_pool.RefillRequest) (enableLoop bool, refillWindow range_pool.IntRange, err error) {
	log.Infof("------------------start Refill ------------------")
	need := req.Need
	switch partition {
	case range_pool.RangePoolLiveName:
		if l.LiveEnableLoop {
			enableLoop = true
		}
		start := l.LiveStart
		end := l.LiveCursor + need
		if end >= l.Max {
			end = l.Max
			l.LiveEnableLoop = true
			//enableLoop = true
			l.LiveStart = l.Min
			l.LiveCursor = l.Min
		} else {
			l.LiveEnableLoop = false
			l.LiveCursor = end
		}
		refillWindow = range_pool.IntRange{
			start,
			end,
		}
		if enableLoop {
			log.Infof("min:%d max:%d live_start:%d live_cursor:%d LiveEnableLoop:%v refillWindow:%d~%d", l.Min, l.Max, l.LiveStart, l.LiveCursor, l.LiveEnableLoop, refillWindow.Start, refillWindow.End)
		}
	case range_pool.RangePoolFreeName:
		if l.FreeEnableLoop {
			enableLoop = true
		}
		start := l.FreeStart
		end := l.FreeCursor + need
		if end >= l.Max {
			end = l.Max
			l.FreeEnableLoop = true
			l.FreeCursor = l.Min
			l.FreeStart = l.Min
		} else {
			l.FreeEnableLoop = false
			l.FreeCursor = end
		}
		refillWindow = range_pool.IntRange{
			start,
			end,
		}
		//log.Infof("min:%d max:%d live_start:%d live_cursor:%d LiveEnableLoop:%v refillWindow:%d~%d", l.Min, l.Max, l.FreeStart, l.FreeCursor, l.FreeEnableLoop, refillWindow.Start, refillWindow.End)
	}
	return
}

func TestLiveRefill(t *testing.T) {
	refill := &LiveRefill{
		Min:            0,
		Max:            100,
		LiveStart:      0,
		LiveCursor:     0,
		LiveEnableLoop: false,
		FreeStart:      50,
		FreeCursor:     50,
		FreeEnableLoop: false,
	}
	t.Run("run 1", func(t *testing.T) {
		loop, window, err := refill.Refill(range_pool.RangePoolLiveName, range_pool.RefillRequest{Need: 20})
		if err != nil {
			t.Fatal(err)
			return
		}
		t.Logf("loop:%v window:%+v", loop, window)
	})

	t.Run("run 2", func(t *testing.T) {
		for i := 0; i < 10; i++ {
			loop, window, err := refill.Refill(range_pool.RangePoolLiveName, range_pool.RefillRequest{Need: 21})
			if err != nil {
				t.Fatal(err)
				return
			}
			t.Logf("loop:%v window:%+v", loop, window)
		}
	})

	t.Run("run 3", func(t *testing.T) {
		for i := 0; i < 10; i++ {
			loop, window, err := refill.Refill(range_pool.RangePoolFreeName, range_pool.RefillRequest{Need: 21})
			if err != nil {
				t.Fatal(err)
				return
			}
			t.Logf("loop:%v window:%+v", loop, window)
		}
	})
}

func TestRangePool(t *testing.T) {
	refill := &LiveRefill{
		Min:            0,
		Max:            15000,
		LiveStart:      0,
		LiveCursor:     0,
		LiveEnableLoop: false,
		FreeStart:      5000,
		FreeCursor:     5000,
		FreeEnableLoop: false,
	}
	pool, err := range_pool.NewRangePool(range_pool.WithRandSource(rand.NewSource(1)), range_pool.WithRefill(refill.Refill))
	if err != nil {
		t.Fatal(err)
		return
	}
	//r := rand.NewSource(time.Now().UnixNano())
	sizeMap := map[int]int64{
		0: 1,
		1: 5,
		2: 10,
		3: 20,
		4: 100,
		5: 500,
	}
	for i := 0; i < 10000; i++ {
		index := rand.Intn(6)
		size := sizeMap[index]
		reserveDelete, b := pool.ReserveDelete(size)
		if !b {
			//t.Logf("reserveDelete false")
			break
		}
		t.Logf("index:%d size:%d:%d~%d", i, size, reserveDelete.Start, reserveDelete.End)
		//pool.DebugLog(range_pool.RangePoolLiveName)
	}
}

type testRefill struct {
	mu    sync.Mutex
	start map[string]int64
	end   map[string]int64
	max   int64
	loop  bool
}

func newTestRefill(max int64, loop bool) *testRefill {
	return &testRefill{
		start: map[string]int64{
			range_pool.RangePoolLiveName: 0,
			range_pool.RangePoolFreeName: 0,
		},
		end: map[string]int64{
			range_pool.RangePoolLiveName: 0,
			range_pool.RangePoolFreeName: 0,
		},
		max:  max,
		loop: loop,
	}
}

func (r *testRefill) Refill(partition string, req range_pool.RefillRequest) (bool, range_pool.IntRange, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	start := r.start[partition]
	end := r.end[partition] + req.Need
	enableLoop := false
	if end >= r.max {
		end = r.max
		enableLoop = r.loop
	}
	r.end[partition] = end
	return enableLoop, range_pool.IntRange{Start: start, End: end}, nil
}

func TestRangePoolInvalidConfig(t *testing.T) {
	refill := newTestRefill(1000, false)
	invalidTiers := []range_pool.TierConfig{
		{Size: 5, Threshold: 1, MaxCount: 1},
		{Size: 6, Threshold: 1, MaxCount: 2},
	}
	_, err := range_pool.NewRangePool(
		range_pool.WithRefill(refill.Refill),
		range_pool.WithLiveTiers(invalidTiers),
		range_pool.WithFreeTiers(invalidTiers),
	)
	if err == nil {
		t.Fatal("expected invalid tier config error")
	}
}

func TestRangePoolReserveInsertTooLarge(t *testing.T) {
	refill := newTestRefill(1000, false)
	tiers := []range_pool.TierConfig{
		{Size: 2, Threshold: 1, MaxCount: 3},
		{Size: 4, Threshold: 1, MaxCount: 2},
	}
	pool, err := range_pool.NewRangePool(
		range_pool.WithRefill(refill.Refill),
		range_pool.WithLiveTiers(tiers),
		range_pool.WithFreeTiers(tiers),
		range_pool.WithRandSource(rand.NewSource(1)),
	)
	if err != nil {
		t.Fatalf("new pool: %v", err)
	}
	defer pool.Close()

	if _, ok := pool.ReserveInsert(10); ok {
		t.Fatal("expected reserve insert to fail for oversized need")
	}
	r, ok := pool.ReserveInsert(3)
	if !ok {
		t.Fatal("expected reserve insert to succeed")
	}
	if r.Len() != 4 {
		t.Fatalf("expected length 4, got %d", r.Len())
	}
}

func TestRangePoolConcurrentReserveInsert(t *testing.T) {
	refill := newTestRefill(1_000_000, true)
	tiers := []range_pool.TierConfig{
		{Size: 2, Threshold: 1, MaxCount: 4},
		{Size: 4, Threshold: 1, MaxCount: 2},
	}
	pool, err := range_pool.NewRangePool(
		range_pool.WithRefill(refill.Refill),
		range_pool.WithLiveTiers(tiers),
		range_pool.WithFreeTiers(tiers),
		range_pool.WithRandSource(rand.NewSource(1)),
	)
	if err != nil {
		t.Fatalf("new pool: %v", err)
	}
	defer pool.Close()

	const goroutines = 20
	const perG = 50
	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < perG; j++ {
				pool.ReserveInsert(2)
			}
		}()
	}

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("concurrent reserve insert timed out")
	}
}
