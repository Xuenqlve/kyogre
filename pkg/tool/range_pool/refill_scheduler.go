package range_pool

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/xuenqlve/common/log"
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

	lastRefill time.Time
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
		select {
		case res := <-resp:
			return res.err
		case <-s.ctx.Done():
			return fmt.Errorf("%s refill canceled", s.name)
		}
	}

	select {
	case s.reqCh <- req:
		log.Infof("---------- Push refill:%v", req.need)
	default:
		// Drop if queue is busy; next consume will retry.
		log.Warnf("---------- Drop if queue is busy: name=%s need=%d reason=%s", s.name, req.need, req.reason)
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
			log.Infof("---------- [doRefill] req:%v", req.need)
			err := s.doRefill(req.need)
			if req.resp != nil {
				req.resp <- refillResult{err: err}
			}
		}
	}
}

func (s *refillScheduler) doRefill(need int64) error {
	const minRefillInterval = 50 * time.Millisecond

	if !s.lastRefill.IsZero() {
		elapsed := time.Since(s.lastRefill)
		if elapsed < minRefillInterval {
			time.Sleep(minRefillInterval - elapsed)
		}
	}
	enableLoop, win, err := s.refill(s.name, need)
	if err != nil {
		return err
	}
	s.lastRefill = time.Now()
	return s.window.applyRefillWindow(enableLoop, win)
}
