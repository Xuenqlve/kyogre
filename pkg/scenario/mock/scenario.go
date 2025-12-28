package mock

import (
	"context"
	"fmt"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/xuenqlve/common/errors"
	"github.com/xuenqlve/common/log"
	"github.com/xuenqlve/kyogre/internal/message"
	"github.com/xuenqlve/kyogre/internal/plugin/generator"
	"github.com/xuenqlve/kyogre/internal/plugin/iquery"
	"github.com/xuenqlve/kyogre/internal/plugin/scenario"
	mock "github.com/xuenqlve/kyogre/pkg/generator/mock"
)

const ScenarioType scenario.Type = "mock"

type Config struct {
	MessageCount int          `mapstructure:"message-count"`
	IntervalMS   int          `mapstructure:"interval-ms"`
	WorkerCount  int          `mapstructure:"worker-count"`
	IQuery       IQueryConfig `mapstructure:"iquery"`
}

// IQueryConfig 开关化的反查配置，尽量自动推导表/字段
type IQueryConfig struct {
	Enabled bool   `mapstructure:"enabled"`
	Key     string `mapstructure:"key"`
}

type Scenario struct {
	pipeline   string
	cfg        Config
	ctx        context.Context
	generators []generator.Generator
	lookup     iquery.Lookup
	sequencer  *iquery.Sequencer
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
	s.generators = append(s.generators, gen)
}

func (s *Scenario) Preparation(ctx context.Context) error {
	s.ctx = ctx
	if len(s.generators) == 0 {
		return fmt.Errorf("no generators registered")
	}
	if s.cfg.WorkerCount > len(s.generators) {
		// 一个 worker 绑定一个 generator，避免重复复用
		s.cfg.WorkerCount = len(s.generators)
	}
	if s.cfg.IQuery.Enabled {
		if err := s.setupIQuery(ctx); err != nil {
			return err
		}
	}
	return nil
}

func (s *Scenario) Start(msgChan message.InPoint) error {
	interval := time.Duration(s.cfg.IntervalMS) * time.Millisecond
	for workerID := 0; workerID < s.cfg.WorkerCount; workerID++ {
		gen := s.generators[workerID]
		go s.runWorker(workerID, gen, msgChan, interval)
	}
	return nil
}

func (s *Scenario) Close() error { return nil }

func (s *Scenario) runWorker(workerID int, gen generator.Generator, msgChan message.InPoint, interval time.Duration) {
	for i := 0; i < s.cfg.MessageCount; i++ {
		req := generator.NewDependencyRequest(nil, nil)
		if s.cfg.IQuery.Enabled && s.sequencer != nil {
			if depReq := s.nextSequenceDependency(workerID); depReq != nil {
				req = depReq
			}
		}

		dep, err := gen.CollectDependencies(req)
		if err != nil {
			log.Warnf("worker %d collect dependencies failed: %v", workerID, err)
			continue
		}
		msg, err := gen.MockMessage(generator.NewMessageGenerationRequest(dep, nil, nil))
		if err != nil {
			log.Warnf("worker %d mock message failed: %v", workerID, err)
			continue
		}
		log.Infof("mock worker=%d msg: %+v", workerID, msg)
		select {
		case msgChan <- msg:
		case <-s.ctx.Done():
			return
		}
		time.Sleep(interval)
	}

	// 发送 nil 表示链路完成，触发 pipeline 结束
	select {
	case msgChan <- nil:
	case <-s.ctx.Done():
	}
}

func (s *Scenario) setupIQuery(ctx context.Context) error {
	var (
		lookupKey string
		err       error
	)
	switch {
	case s.cfg.IQuery.Key != "":
		lookupKey = s.cfg.IQuery.Key
		//s.lookup, err = iquery.IQueryManager.GetIQueryLookup(lookupKey)
	case s.cfg.IQuery.Module != nil:
		lookupKey = s.cfg.IQuery.Module.Type
		//err = iquery.IQueryManager.Configure(s.pipeline, map[string]config.ConfigureMold{
		//	lookupKey: *s.cfg.IQuery.Module,
		//})
		//if err == nil {
		//	s.lookup, err = iquery.IQueryManager.GetIQueryLookup(lookupKey)
		//}
	default:
		// 默认走内存反查，省配置
		lookupKey = "memory"
		//err = iquery.IQueryManager.Configure(s.pipeline, map[string]config.ConfigureMold{
		//	lookupKey: {Type: lookupKey},
		//})
		//if err == nil {
		//	s.lookup, err = iquery.IQueryManager.GetIQueryLookup(lookupKey)
		//}
	}
	if err != nil {
		return fmt.Errorf("init iquery failed: %w", err)
	}

	seq, err := iquery.NewSequencer()
	if err != nil {
		return err
	}
	if err = seq.Configure(s.pipeline, iquery.WithWorkerCount(s.cfg.WorkerCount), iquery.WithLookUpKey(lookupKey)); err != nil {
		return err
	}

	specs := s.buildSequenceSpecs()
	if len(specs) == 0 {
		return fmt.Errorf("no sequence spec derived from metadata")
	}
	if err = seq.Preallocate(ctx, specs); err != nil {
		return err
	}
	s.sequencer = seq
	return nil
}

func (s *Scenario) buildSequenceSpecs() []iquery.SequenceSpec {
	if s.metadata == nil {
		return nil
	}
	field := s.cfg.IQuery.Field
	if field == "" {
		field = "id"
	}
	keys := s.metadata.SchemaKeys()
	specs := make([]iquery.SequenceSpec, 0, len(keys))
	for _, key := range keys {
		specs = append(specs, iquery.SequenceSpec{
			Schema: key,
			Field:  field,
			Step:   1,
			Width:  s.cfg.IQuery.Width,
		})
	}
	return specs
}

func (s *Scenario) nextSequenceDependency(workerID int) *generator.DependencyRequest {
	if s.sequencer == nil {
		return nil
	}
	specs := s.buildSequenceSpecs()
	if len(specs) == 0 {
		return nil
	}
	// 简化：取第一个表作为演示
	start, _, err := s.sequencer.Reserve(s.ctx, workerID, specs[0], 1)
	if err != nil {
		log.Warnf("worker %d reserve sequence failed: %v", workerID, err)
		return nil
	}
	val := fmt.Sprintf("%s-%d", s.cfg.IQuery.Field, start)
	cfg := &mock.MockDependencyConfig{ForceValue: val}
	return generator.NewDependencyRequest(cfg, nil)
}
