package mock

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/xuenqlve/common/errors"
	"github.com/xuenqlve/common/log"
	"github.com/xuenqlve/kyogre/internal/message"
	"github.com/xuenqlve/kyogre/internal/plugin/generator"
	"github.com/xuenqlve/kyogre/internal/plugin/scenario"
)

const ScenarioType scenario.Type = "mock"

type Config struct {
	MessageCount int          `mapstructure:"message-count"`
	IntervalMS   int          `mapstructure:"interval-ms"`
	WorkerCount  int          `mapstructure:"worker-count"`
	IQuery       IQueryConfig `mapstructure:"lookup"`
}

type IQueryConfig struct {
	Enabled bool   `mapstructure:"enabled"`
	Key     string `mapstructure:"key"`
}

type Scenario struct {
	*scenario.BaseScenario
	cfg Config

	ctx context.Context
	wg  sync.WaitGroup

	startAt      time.Time
	sentCount    atomic.Int64
	completeOnce sync.Once
}

func init() {
	scenario.RegisterScenario(ScenarioType, &Scenario{}, false)
}

func (s *Scenario) Configure(pipeline string, data map[string]any) error {
	s.BaseScenario = scenario.NewBaseScenario(pipeline)
	s.EnableFinite()
	s.sentCount.Store(0)
	s.completeOnce = sync.Once{}

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

func (s *Scenario) Preparation(ctx context.Context) error {
	s.ctx = ctx
	return nil
}

func (s *Scenario) Start(msgChan message.InPoint) error {
	if s.ctx == nil {
		return fmt.Errorf("scenario %s not prepared", s.Pipeline())
	}
	s.startAt = time.Now()
	interval := time.Duration(s.cfg.IntervalMS) * time.Millisecond
	for workerID := 0; workerID < s.cfg.WorkerCount; workerID++ {
		gen := s.Generators()[workerID]
		s.wg.Add(1)
		go s.runWorker(workerID, gen, msgChan, interval)
	}
	go s.waitAndNotify()
	return nil
}

func (s *Scenario) Close() error {
	s.wg.Wait()
	return nil
}

func (s *Scenario) runWorker(workerID int, gen generator.Generator, msgChan message.InPoint, interval time.Duration) {
	defer s.wg.Done()
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
			s.sentCount.Add(1)
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

func (s *Scenario) waitAndNotify() {
	s.wg.Wait()
	s.notifyComplete()
}

func (s *Scenario) notifyComplete() {
	s.completeOnce.Do(func() {
		total := s.sentCount.Load()
		duration := time.Since(s.startAt)
		summary := map[string]any{
			"pipeline":     s.Pipeline(),
			"workers":      s.cfg.WorkerCount,
			"messages":     total,
			"duration":     duration.String(),
			"interval_ms":  s.cfg.IntervalMS,
			"message_type": message.MockType,
		}
		log.Infof("[%s] mock scenario completed summary=%v", s.Pipeline(), summary)
		if helper := s.EnableFinite(); helper != nil {
			helper.NotifyDone(summary)
		}
	})
}
