package mock

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/xuenqlve/common/errors"
	"github.com/xuenqlve/common/log"
	"github.com/xuenqlve/kyogre/internal/message"
	"github.com/xuenqlve/kyogre/internal/plugin/generator"
	"github.com/xuenqlve/kyogre/internal/plugin/iquery"
	"github.com/xuenqlve/kyogre/internal/plugin/scenario"
)

const ScenarioType scenario.Type = "mock"

type Config struct {
	MessageCount int          `mapstructure:"message-count"`
	IntervalMS   int          `mapstructure:"interval-ms"`
	WorkerCount  int          `mapstructure:"worker-count"`
	IQuery       IQueryConfig `mapstructure:"iquery"`
}

type IQueryConfig struct {
	Enabled bool   `mapstructure:"enabled"`
	Key     string `mapstructure:"key"`
}

type Scenario struct {
	pipeline string
	cfg      Config

	ctx        context.Context
	generators []generator.Generator
	lookup     iquery.Lookup
	workers    sync.WaitGroup
}

func init() {
	scenario.RegisterScenario(ScenarioType, &Scenario{}, false)
}

func (s *Scenario) Configure(pipeline string, data map[string]any) error {
	s.pipeline = pipeline
	if err := mapstructure.Decode(data, &s.cfg); err != nil {
		return errors.Trace(err)
	}
	if s.cfg.MessageCount <= 0 {
		s.cfg.MessageCount = 5
	}
	if s.cfg.IntervalMS <= 0 {
		s.cfg.IntervalMS = 100
	}
	if s.cfg.WorkerCount <= 0 {
		s.cfg.WorkerCount = 1
	}
	return nil
}

func (s *Scenario) RegisterGenerator(gen generator.Generator) {
	if gen != nil {
		s.generators = append(s.generators, gen)
	}
}

func (s *Scenario) RegisterIQueryLookup(lookup iquery.Lookup) {
	s.lookup = lookup
}

func (s *Scenario) Preparation(ctx context.Context) error {
	s.ctx = ctx
	if len(s.generators) == 0 {
		return fmt.Errorf("no generators registered")
	}
	if s.cfg.WorkerCount > len(s.generators) {
		s.cfg.WorkerCount = len(s.generators)
	}
	if s.cfg.IQuery.Enabled && s.lookup == nil {
		return fmt.Errorf("scenario %s requires iquery '%s' but lookup is nil", s.pipeline, s.cfg.IQuery.Key)
	}
	return nil
}

func (s *Scenario) Start(msgChan message.InPoint) error {
	if s.ctx == nil {
		return fmt.Errorf("scenario %s not prepared", s.pipeline)
	}
	interval := time.Duration(s.cfg.IntervalMS) * time.Millisecond
	for workerID := 0; workerID < s.cfg.WorkerCount; workerID++ {
		gen := s.generators[workerID]
		s.workers.Add(1)
		go s.runWorker(workerID, gen, msgChan, interval)
	}
	return nil
}

func (s *Scenario) Close() error {
	s.workers.Wait()
	return nil
}

func (s *Scenario) runWorker(workerID int, gen generator.Generator, msgChan message.InPoint, interval time.Duration) {
	defer s.workers.Done()
	for i := 0; i < s.cfg.MessageCount; i++ {
		select {
		case <-s.ctx.Done():
			return
		default:
		}

		depReq := generator.NewDependencyRequest(nil, nil)
		dep, err := gen.CollectDependencies(depReq)
		if err != nil {
			log.Warnf("mock scenario generator failure worker=%d: %v", workerID, err)
			continue
		}
		msg, err := gen.MockMessage(generator.NewMessageGenerationRequest(dep, nil, nil))
		if err != nil {
			log.Warnf("mock scenario message failure worker=%d: %v", workerID, err)
			continue
		}

		select {
		case msgChan <- msg:
		case <-s.ctx.Done():
			return
		}

		select {
		case <-s.ctx.Done():
			return
		case <-time.After(interval):
		}
	}
}
