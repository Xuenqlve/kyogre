package range_pool

import (
	"context"
	"fmt"
	"sync"
)

type refillRequest struct {
	need   int64
	reason string
	resp   chan refillResult
}

type refillResult struct {
	err error
}

type refillScheduler struct {
	name   string
	refill RangePoolRefillFunc
	window *windowManager

	ctx    context.Context
	cancel context.CancelFunc

	reqCh chan refillRequest
	wg    sync.WaitGroup
}

func newRefillScheduler(name string, refill RangePoolRefillFunc, window *windowManager) *refillScheduler {
	return &refillScheduler{
		name:   name,
		refill: refill,
		window: window,
	}
}

func (s *refillScheduler) start(ctx context.Context) {
	if ctx == nil {
		ctx = context.Background()
	}
	if s.ctx == nil {
		s.ctx, s.cancel = context.WithCancel(ctx)
	}
	if s.reqCh == nil {
		s.reqCh = make(chan refillRequest, 1)
		s.wg.Add(1)
		go s.loop()
	}
}

func (s *refillScheduler) stop() {
	if s.cancel != nil {
		s.cancel()
	}
	s.wg.Wait()
}

func (s *refillScheduler) schedule(need int64, reason string, wait bool) error {
	if need <= 0 {
		need = 1
	}
	if s.reqCh == nil {
		return fmt.Errorf("%s refill worker not started", s.name)
	}

	var resp chan refillResult
	if wait {
		resp = make(chan refillResult, 1)
	}
	req := refillRequest{
		need:   need,
		reason: reason,
		resp:   resp,
	}

	if wait {
		select {
		case s.reqCh <- req:
		case <-s.ctx.Done():
			return fmt.Errorf("%s refill canceled", s.name)
		}
		res := <-resp
		return res.err
	}

	select {
	case s.reqCh <- req:
	default:
		// Drop if queue is busy; next consume will retry.
	}
	return nil
}

func (s *refillScheduler) loop() {
	defer s.wg.Done()
	for {
		select {
		case <-s.ctx.Done():
			return
		case req := <-s.reqCh:
			err := s.doRefill(req.need)
			if req.resp != nil {
				req.resp <- refillResult{err: err}
			}
		}
	}
}

func (s *refillScheduler) doRefill(need int64) error {
	enableLoop, win, err := s.refill(s.name, need)
	if err != nil {
		return err
	}
	return s.window.applyRefillWindow(enableLoop, win)
}
