package scenario

import (
	"context"
	"fmt"
	"sync"

	"github.com/xuenqlve/kyogre/internal/event"
	"github.com/xuenqlve/kyogre/internal/message"
	"github.com/xuenqlve/kyogre/internal/plugin/generator"
)

// Worker 消费 GenerationContext 并生成消息。
// kind 与 generatorKey 由 director 路由绑定。
type Worker struct {
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
	once   sync.Once

	in       <-chan generator.GenerationContext
	out      message.InPoint
	gens     map[string]generator.Generator
	kindMap  map[string]string
	pipeline string
	onError  func(error)
}

func NewWorker(ctx context.Context, pipeline string, gens map[string]generator.Generator, kindMap map[string]string, in <-chan generator.GenerationContext, out message.InPoint, onError func(error)) *Worker {
	lctx, cancel := context.WithCancel(ctx)
	return &Worker{
		ctx:      lctx,
		cancel:   cancel,
		in:       in,
		out:      out,
		gens:     gens,
		kindMap:  kindMap,
		pipeline: pipeline,
		onError:  onError,
	}
}

func (w *Worker) Start() {
	w.wg.Add(1)
	go func() {
		defer func() {
			w.wg.Done()
			if r := recover(); r != nil {
				w.reportError(fmt.Errorf("panic: %v", r))
			}
		}()
		if err := w.run(); err != nil {
			w.reportError(err)
			return
		}
	}()
}

func (w *Worker) run() (err error) {
	for {
		select {
		case <-w.ctx.Done():
			return
		case gctx, ok := <-w.in:
			if !ok {
				return
			}
			if gctx == nil {
				err = fmt.Errorf("scenario worker: context is nil")
				return
			}
			if err = gctx.Validate(); err != nil {
				err = fmt.Errorf("scenario worker: context validate failed: %v", err)
				return
			}
			kind := gctx.Kind()
			if kind == "" {
				err = fmt.Errorf("scenario worker: context kind is empty")
				return
			}
			genKey, ok := w.kindMap[kind]
			if !ok || genKey == "" {
				err = fmt.Errorf("scenario worker: generator key not found for kind=%s", kind)
				return
			}
			gen, exist := w.gens[genKey]
			if gen == nil || !exist {
				err = fmt.Errorf("scenario worker: generator not found kind=%s key=%s", kind, genKey)
				return
			}
			var msg message.Message
			if msg, err = gen.Generate(gctx); err != nil {
				err = fmt.Errorf("scenario worker: generate failed kind=%s err=%v", kind, err)
				return
			}
			select {
			case <-w.ctx.Done():
				return
			case w.out <- msg:
			}
		}
	}
}

func (w *Worker) reportError(err error) {
	if err == nil {
		return
	}
	if w.onError != nil {
		w.onError(err)
	}
	event.EventAdmin.Upload(event.Event{
		Type: event.WorkerError,
		Key:  w.pipeline,
		Value: map[string]any{
			"component": "scenario-worker",
			"err":       err.Error(),
		},
	})
}

func (w *Worker) Done() {
	w.once.Do(func() {
		w.cancel()
	})
}

func (w *Worker) Close() {
	w.wg.Wait()
}
