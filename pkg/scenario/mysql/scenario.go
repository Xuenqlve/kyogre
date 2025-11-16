package mysql

import (
	"context"

	"github.com/mitchellh/mapstructure"
	"github.com/xuenqlve/kyogre/internal/message"
	"github.com/xuenqlve/kyogre/internal/plugin"
)

type Scenario struct {
	pipeline  string
	cfg       *Config
	ctx       context.Context
	generator []plugin.Generator
	workers   []*Worker
}

func (s *Scenario) Configure(pipeline string, data map[string]any) (err error) {
	s.pipeline = pipeline
	if err = mapstructure.Decode(data, s.cfg); err != nil {
		return
	}
	if err = s.cfg.Validate(); err != nil {
		return
	}
	return
}

func (s *Scenario) Preparation(ctx context.Context) error {
	s.ctx = ctx
	// todo 初始化对应用 metadata
	// todo 初始化对应 generator
	return nil
}

func (s *Scenario) Start(msgChan message.InPoint) error {
	for i := 0; i < s.cfg.WorkerCount; i++ {
		worker := NewWorker(s.ctx, msgChan, s.generator[i])
		worker.Start()
		s.workers = append(s.workers, worker)
	}
	return nil
}

func (s *Scenario) Close() error {
	for _, w := range s.workers {
		w.Close()
	}
	return nil
}
