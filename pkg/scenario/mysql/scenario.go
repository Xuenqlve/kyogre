package mysql

import (
	"context"
	"fmt"

	"github.com/mitchellh/mapstructure"
	"github.com/xuenqlve/kyogre/internal/message"
	"github.com/xuenqlve/kyogre/internal/plugin/generator"
	"github.com/xuenqlve/kyogre/internal/plugin/iquery"
	"github.com/xuenqlve/kyogre/internal/plugin/metadata"
	"github.com/xuenqlve/kyogre/internal/plugin/scenario"
)

const ScenarioType scenario.Type = "mysql"

func init() {
	scenario.RegisterScenario(ScenarioType, NewScenario(), false)
}

type Scenario struct {
	*scenario.BaseScenario
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
	s.BaseScenario = scenario.NewBaseScenario(pipeline)
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
	if len(s.generator) == 0 {
		return fmt.Errorf("no generators registered")
	}
	if len(s.generator) < s.cfg.WorkerCount {
		return fmt.Errorf("generator count %d less than worker count %d", len(s.generator), s.cfg.WorkerCount)
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

func (s *Scenario) RegisterGenerator(gen generator.Generator) {
	s.generator = append(s.generator, gen)
}
