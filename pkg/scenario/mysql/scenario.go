package mysql

import (
	"context"
	"fmt"

	"github.com/mitchellh/mapstructure"
	"github.com/xuenqlve/kyogre/internal/config"
	"github.com/xuenqlve/kyogre/internal/message"
	"github.com/xuenqlve/kyogre/internal/plugin/generator"
	"github.com/xuenqlve/kyogre/internal/plugin/iquery"
	"github.com/xuenqlve/kyogre/internal/plugin/metadata"
)

type Scenario struct {
	pipeline  string
	cfg       *Config
	ctx       context.Context
	metadata  metadata.Metadata
	generator []generator.Generator
	workers   []*Worker

	lookup    iquery.Lookup
	sequencer *iquery.Sequencer
}

func NewScenario() *Scenario {
	return &Scenario{
		generator: make([]generator.Generator, 0),
		workers:   make([]*Worker, 0),
	}
}

func (s *Scenario) Configure(pipeline string, data map[string]any) (err error) {
	s.pipeline = pipeline
	s.cfg = &Config{}
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
	if s.cfg.EnableIQuery {
		if s.cfg.IQueryModule == nil {
			return fmt.Errorf("iquery-module config is required when enable-iquery is true")
		}
		if err := iquery.IQueryManager.Configure(s.pipeline, map[string]config.ConfigureMold{
			s.cfg.IQueryModule.Type: *s.cfg.IQueryModule,
		}); err != nil {
			return err
		}
		lookup, err := iquery.IQueryManager.GetIQueryLookup(s.cfg.IQueryModule.Type)
		if err != nil {
			return err
		}
		sequencer, err := iquery.NewSequencer()
		if err != nil {
			return err
		}
		if err = sequencer.Configure(
			s.pipeline,
			iquery.WithWorkerCount(s.cfg.WorkerCount),
			iquery.WithLookUpKey(s.cfg.IQueryModule.Type),
		); err != nil {
			return err
		}

		// 预分段
		if s.metadata != nil &&
			s.cfg.GenerationStrategy != nil &&
			s.cfg.GenerationStrategy.SequenceConfig != nil &&
			s.cfg.GenerationStrategy.SequenceConfig.Enabled {
			specs := make([]iquery.SequenceSpec, 0, len(s.metadata.SchemaKeys()))
			for _, key := range s.metadata.SchemaKeys() {
				specs = append(specs, iquery.SequenceSpec{
					Schema: key,
					Field:  s.cfg.GenerationStrategy.SequenceConfig.Field,
					Step:   s.cfg.GenerationStrategy.SequenceConfig.Step,
					Width:  s.cfg.GenerationStrategy.SequenceConfig.Width,
				})
			}
			if len(specs) > 0 {
				if err = sequencer.Preallocate(ctx, specs); err != nil {
					return err
				}
			}
		}

		s.lookup = lookup
		s.sequencer = sequencer
	}

	return nil
}

func (s *Scenario) Start(msgChan message.InPoint) error {
	if len(s.generator) == 0 {
		return fmt.Errorf("no generators initialized")
	}

	// 从配置中获取依赖配置
	dependencyConfig, err := s.cfg.DependencyConfig.GetDependencyConfig()
	if err != nil {
		return fmt.Errorf("failed to get dependency config: %w", err)
	}

	for i := 0; i < s.cfg.WorkerCount; i++ {
		worker := NewWorker(
			s.ctx,
			msgChan,
			s.generator[i],
			s.lookup,
			s.sequencer,
			i,                        // worker ID
			dependencyConfig,         // 传入依赖配置
			s.cfg.GenerationStrategy, // 传入生成策略
		)
		worker.Start()
		s.workers = append(s.workers, worker)
	}
	return nil
}

func (s *Scenario) Close() error {
	for _, w := range s.workers {
		w.Close()
	}

	if s.sequencer != nil {
		if err := s.sequencer.Close(); err != nil {
			return err
		}
	}

	if s.lookup != nil {
		if err := s.lookup.Close(); err != nil {
			return err
		}
	}

	return nil
}

func (s *Scenario) RegisterMetadata(md metadata.Metadata) {
	s.metadata = md
}
