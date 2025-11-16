package mysql

import (
	"context"
	"sync"

	"github.com/xuenqlve/common/log"
	"github.com/xuenqlve/kyogre/internal/message"
	"github.com/xuenqlve/kyogre/internal/plugin"
)

type Worker struct {
	ctx       context.Context
	cancel    context.CancelFunc
	wg        sync.WaitGroup
	msgQueue  message.InPoint
	generator plugin.Generator
	once      sync.Once
}

func NewWorker(ctx context.Context, msgChan message.InPoint, generator plugin.Generator) *Worker {
	lctx, cancel := context.WithCancel(ctx)
	return &Worker{
		ctx:       lctx,
		cancel:    cancel,
		wg:        sync.WaitGroup{},
		msgQueue:  msgChan,
		generator: generator,
		once:      sync.Once{},
	}
}

func (w *Worker) Start() {
	w.wg.Add(1)
	go func() {
		defer func() {
			w.wg.Done()
			if r := recover(); r != nil {
				log.Errorf("MySQL ScanWork.run panicked: %v", r)
			}
		}()

		if err := w.run(); err != nil {
			log.Errorf("ScanWork.run err:%s", err.Error())
			return
		}
	}()
}

func (w *Worker) run() (err error) {
	for {
		select {
		case <-w.ctx.Done():
			log.Infof("[scan work] ctx Done")
			return
		default:
		}
		//todo
		param := plugin.MockParam{}

		mockMessage := w.generator.MockMessage(param)
		w.msgQueue <- mockMessage
	}
}

func (w *Worker) Done() {
	w.once.Do(func() {
		w.cancel()
	})
}

func (w *Worker) Close() {
	w.wg.Wait()
}
