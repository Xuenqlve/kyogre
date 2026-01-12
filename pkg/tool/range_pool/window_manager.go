package range_pool

import (
	"fmt"
	"sync"
)

type windowState struct {
	init   bool
	start  int64
	end    int64
	cursor int64
	loop   bool
}

type windowManager struct {
	name string

	mu    sync.RWMutex
	state windowState
}

func newWindowManager(name string) *windowManager {
	return &windowManager{name: name}
}

func (w *windowManager) stateSnapshot() windowState {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.state
}

func (w *windowManager) windowState() (start, cursor, end int64, enableLoop bool) {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.state.start, w.state.cursor, w.state.end, w.state.loop
}

func (w *windowManager) withWrite(fn func(*windowState) error) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	ws := w.state
	if err := fn(&ws); err != nil {
		return err
	}
	w.state = ws
	return nil
}

func (w *windowManager) applyRefillWindow(enableLoop bool, win IntRange) error {
	if !win.Valid() {
		return fmt.Errorf("%s refill returned invalid window", w.name)
	}
	return w.withWrite(func(ws *windowState) error {
		if !ws.init {
			ws.init = true
			ws.start = win.Start
			ws.end = win.End
			ws.cursor = ws.start
			ws.loop = ws.loop || enableLoop
			return nil
		}
		if win.Start != ws.start {
			return fmt.Errorf("%s refill window start changed: %d -> %d", w.name, ws.start, win.Start)
		}
		ws.loop = ws.loop || enableLoop
		if win.End > ws.end {
			ws.end = win.End
		}
		return nil
	})
}

// ensureCapacity 确保当前分配窗口至少具备 need 的容量。
func (w *windowManager) ensureCapacity(need int64, bootstrap bool, refill func(need int64, reason string, wait bool) error) error {
	if need <= 0 {
		return nil
	}

	for {
		state := w.stateSnapshot()
		if !state.init {
			if err := refill(need, "bootstrap-init", true); err != nil {
				return err
			}
			continue
		}

		window := IntRange{Start: state.start, End: state.end}
		if window.Len() >= need {
			return nil
		}

		missing := need - window.Len()
		prevEnd := state.end
		if err := refill(missing, "expand-window", true); err != nil {
			return err
		}
		state = w.stateSnapshot()
		if bootstrap && state.end <= prevEnd {
			return fmt.Errorf("%s allocation window capacity insufficient after refill: need=%d have=%d", w.name, need, (IntRange{Start: state.start, End: state.end}).Len())
		}
	}
}

// allocateSequential 从当前 cursor 开始顺序切分出 count 个 size 长度的连续区间，并推进 cursor。
// 当 cursor 越界时会尝试通过 refill 扩展 windowEnd；若 enableLoop=true 则允许回绕到 windowStart。
func (w *windowManager) allocateSequential(size int64, count int, bootstrap bool, refill func(need int64, reason string, wait bool) error) ([]IntRange, error) {
	if count <= 0 {
		return nil, nil
	}
	if size <= 0 {
		return nil, fmt.Errorf("%s allocate invalid size %d", w.name, size)
	}

	for {
		var (
			segs        []IntRange
			startCursor int64
			desiredEnd  int64
			prevEnd     int64
			needExtra   int64
			needRefill  bool
		)
		err := w.withWrite(func(ws *windowState) error {
			if !ws.init {
				return fmt.Errorf("%s allocation window not initialized", w.name)
			}
			window := IntRange{Start: ws.start, End: ws.end}
			if !window.Valid() {
				return fmt.Errorf("%s allocation window not initialized", w.name)
			}
			if window.Len() < size {
				return fmt.Errorf("%s allocation window too small for size=%d", w.name, size)
			}

			if ws.loop && ws.cursor+size-1 > ws.end {
				ws.cursor = ws.start
			}

			startCursor = ws.cursor
			desiredEnd = startCursor + size*int64(count) - 1
			prevEnd = ws.end

			if desiredEnd <= ws.end {
				segs = make([]IntRange, 0, count)
				cursor := ws.cursor
				for i := 0; i < count; i++ {
					end := cursor + size - 1
					segs = append(segs, IntRange{Start: cursor, End: end})
					cursor = end + 1
				}
				ws.cursor = cursor
				return nil
			}

			needExtra = desiredEnd - prevEnd
			needRefill = true
			return nil
		})
		if err != nil {
			return nil, err
		}
		if segs != nil {
			return segs, nil
		}
		if !needRefill {
			continue
		}

		if err := refill(needExtra, "allocate-sequential", true); err != nil {
			return nil, err
		}

		state := w.stateSnapshot()
		if state.end > prevEnd {
			continue
		}
		if bootstrap {
			return nil, fmt.Errorf("%s allocation window exhausted during bootstrap: cursor=%d size=%d desiredEnd=%d windowEnd=%d", w.name, startCursor, size, desiredEnd, state.end)
		}
		if !state.loop {
			return nil, &partitionError{name: w.name, size: size}
		}
		_ = w.withWrite(func(ws *windowState) error {
			ws.cursor = ws.start
			return nil
		})
	}
}
