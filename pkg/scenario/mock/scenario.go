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
	"github.com/xuenqlve/kyogre/internal/plugin/metadata"
	"github.com/xuenqlve/kyogre/internal/plugin/scenario"
)

const ScenarioType scenario.Type = "mock"

type Config struct {
	MessageCount int `mapstructure:"message-count"`
	IntervalMS   int `mapstructure:"interval-ms"`
}

type Scenario struct {
	pipeline   string
	cfg        Config
	ctx        context.Context
	generators []generator.Generator
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
	return nil
}

func (s *Scenario) RegisterMetadata(metadata.Metadata) {}

func (s *Scenario) RegisterGenerator(gen generator.Generator) {
	s.generators = append(s.generators, gen)
}

func (s *Scenario) Preparation(ctx context.Context) error {
	s.ctx = ctx
	if len(s.generators) == 0 {
		return fmt.Errorf("no generators registered")
	}
	return nil
}

func (s *Scenario) Start(msgChan message.InPoint) error {
	interval := time.Duration(s.cfg.IntervalMS) * time.Millisecond
	go func() {
		for i := 0; i < s.cfg.MessageCount; i++ {
			for _, gen := range s.generators {
				dep, err := gen.CollectDependencies(generator.NewDependencyRequest(nil, nil))
				if err != nil {
					continue
				}
				msg, err := gen.MockMessage(generator.NewMessageGenerationRequest(dep, nil, nil))
				if err != nil {
					continue
				}
				log.Infof("mock generator msg: %+v", msg)
				select {
				case msgChan <- msg:
				case <-s.ctx.Done():
					return
				}
			}
			time.Sleep(interval)
		}
		// 发送 nil 表示链路完成，触发 pipeline 结束
		select {
		case msgChan <- nil:
		case <-s.ctx.Done():
		}
	}()
	return nil
}

func (s *Scenario) Close() error { return nil }
