package scenario

import "sync"

// Finite 表示能够自行确定完成时机的场景，实现后 Pipeline 可以监听 Done() 做收尾。
type Finite interface {
	Done() <-chan struct{}
	Summary() map[string]any
}

// FiniteHelper 提供通用的完成通知与摘要存储，供有限场景复用。
type FiniteHelper struct {
	mu      sync.RWMutex
	done    chan struct{}
	once    sync.Once
	summary map[string]any
}

func NewFiniteHelper() *FiniteHelper {
	return &FiniteHelper{done: make(chan struct{})}
}

func (h *FiniteHelper) Done() <-chan struct{} {
	if h == nil {
		return nil
	}
	return h.done
}

// NotifyDone 关闭完成通道并记录摘要，保证只触发一次。
func (h *FiniteHelper) NotifyDone(summary map[string]any) {
	if h == nil {
		return
	}
	h.once.Do(func() {
		h.mu.Lock()
		h.summary = summary
		d := h.done
		h.mu.Unlock()
		if d != nil {
			close(d)
		}
	})
}

// Summary 返回一次性的执行摘要副本。
func (h *FiniteHelper) Summary() map[string]any {
	if h == nil {
		return nil
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	if h.summary == nil {
		return nil
	}
	snap := make(map[string]any, len(h.summary))
	for k, v := range h.summary {
		snap[k] = v
	}
	return snap
}
