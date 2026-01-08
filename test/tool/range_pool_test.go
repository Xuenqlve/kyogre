package tool

import (
	"math/rand"
	"testing"

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

func (l *LiveRefill) Refill(partition string, need int64) (enableLoop bool, refillWindow range_pool.IntRange, err error) {
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
			l.LiveCursor = l.Min
			l.LiveCursor = l.Min
		} else {
			l.LiveEnableLoop = false
			l.LiveCursor = end
		}
		refillWindow = range_pool.IntRange{
			start,
			end,
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
	}
	log.Infof("min:%d max:%d live_start:%d live_cursor:%d LiveEnableLoop:%v free_start:%d free_cursor:%d FreeEnableLoop:%v", l.Min, l.Max, l.LiveStart, l.LiveCursor, l.LiveEnableLoop, l.FreeStart, l.FreeCursor, l.FreeEnableLoop)
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
		loop, window, err := refill.Refill(range_pool.RangePoolLiveName, 20)
		if err != nil {
			t.Fatal(err)
			return
		}
		t.Logf("loop:%v window:%+v", loop, window)
	})

	t.Run("run 2", func(t *testing.T) {
		for i := 0; i < 10; i++ {
			loop, window, err := refill.Refill(range_pool.RangePoolLiveName, 21)
			if err != nil {
				t.Fatal(err)
				return
			}
			t.Logf("loop:%v window:%+v", loop, window)
		}
	})

	t.Run("run 3", func(t *testing.T) {
		for i := 0; i < 10; i++ {
			loop, window, err := refill.Refill(range_pool.RangePoolFreeName, 21)
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
		Max:            10000,
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
	pool.DebugLog(range_pool.RangePoolFreeName)
	r, b := pool.ReserveInsert(5)
	t.Logf("1 r:%v b:%v", r, b)
	r, b = pool.ReserveInsert(5)
	t.Logf("2 r:%v b:%v", r, b)
	r, b = pool.ReserveInsert(5)
	t.Logf("3 r:%v b:%v", r, b)
	r, b = pool.ReserveInsert(5)
	t.Logf("4 r:%v b:%v", r, b)
	r, b = pool.ReserveInsert(5)
	t.Logf("5 r:%v b:%v", r, b)
	pool.DebugLog(range_pool.RangePoolFreeName)
}
