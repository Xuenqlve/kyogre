package scenario

import (
	"context"
	"fmt"
	"sync"

	"github.com/mitchellh/mapstructure"
	"github.com/xuenqlve/kyogre/internal/message"
	"github.com/xuenqlve/kyogre/internal/plugin/generator"
	"github.com/xuenqlve/kyogre/internal/plugin/iquery"
	"github.com/xuenqlve/kyogre/internal/plugin/metadata"
)

type Director struct {
	mu           sync.Mutex
	pipeline     string
	cfg          DirectorConfig
	ctx          context.Context
	cancel       context.CancelFunc
	scenario     Scenario
	ctxChan      chan generator.GenerationContext
	generators   map[string]generator.Generator
	kindRouting  map[string]string
	metadata     metadata.Metadata
	finite       *FiniteHelper
	sequencer    *iquery.Sequencer
	sequenceSpec []iquery.SequenceSpec

	workers []*Worker
	errOnce sync.Once
	errMu   sync.Mutex
	err     error
}

type runtimeErrorProvider interface {
	RuntimeError() error
}

type DirectorConfig struct {
	WorkerCount   int `mapstructure:"worker-count"`
	ContextLength int `mapstructure:"context-length"`
}

func (c *DirectorConfig) Normalize() {
	if c.WorkerCount <= 0 {
		c.WorkerCount = 1
	}
}

func NewDirector() *Director {
	return &Director{
		cfg:         DirectorConfig{WorkerCount: 1},
		generators:  make(map[string]generator.Generator),
		kindRouting: make(map[string]string),
		finite:      NewFiniteHelper(),
	}
}

func (s *Director) Pipeline() string {
	return s.pipeline
}

func (s *Director) Preparation(ctx context.Context) error {
	s.ctx = ctx
	return nil
}

// ConfigureBase 解析通用配置（例如 worker-count）。
func (s *Director) Configure(pipeline string, key string, data map[string]any) error {
	s.pipeline = pipeline
	s.cfg = DirectorConfig{WorkerCount: 1}
	if err := mapstructure.Decode(data, &s.cfg); err != nil {
		return err
	}
	s.cfg.Normalize()
	scenario, err := GetScenario(Type(key))
	if err != nil {
		return err
	}
	if err = scenario.Configure(s.pipeline, data); err != nil {
		return err
	}
	s.workers = make([]*Worker, 0, s.cfg.WorkerCount)
	s.scenario = scenario
	s.ctxChan = make(chan generator.GenerationContext, s.cfg.ContextLength)
	return nil
}

// RegisterSequencer 在 internal 组装阶段注入 sequencer 以及需要管理的序列信息。
// 场景实现可在 Start 阶段从 Sequencer() 获取并按 worker 分配区间。
func (s *Director) RegisterSequencer(seq *iquery.Sequencer, specs []iquery.SequenceSpec) {
	s.sequencer = seq
	s.sequenceSpec = specs
}

func (s *Director) RegisterGenerator(key string, gen generator.Generator) error {
	if key == "" {
		return fmt.Errorf("generator key is empty")
	}
	if gen == nil {
		return fmt.Errorf("generator %s is nil", key)
	}
	kinds := gen.Kinds()
	if len(kinds) == 0 {
		return fmt.Errorf("generator %s has no kinds", key)
	}
	for _, kind := range kinds {
		if kind == "" {
			return fmt.Errorf("generator %s has empty kind", key)
		}
		if existing, ok := s.kindRouting[kind]; ok && existing != key {
			return fmt.Errorf("kind %s already bound to generator %s", kind, existing)
		}
		s.kindRouting[kind] = key
	}
	s.generators[key] = gen
	return nil
}

func (s *Director) RegisterMetadata(meta metadata.Metadata) {
	s.metadata = meta
}

// EnableFinite 显式开启有限场景能力，返回可用的 FiniteHelper。
func (s *Director) EnableFinite() *FiniteHelper {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.finite == nil {
		s.finite = NewFiniteHelper()
	}
	return s.finite
}

func (s *Director) Start(ctx context.Context, out message.InPoint) error {
	s.errOnce = sync.Once{}
	s.errMu.Lock()
	s.err = nil
	s.errMu.Unlock()
	runCtx := ctx
	if ctx == nil {
		runCtx = context.Background()
	}
	runCtx, s.cancel = context.WithCancel(runCtx)
	for i := 0; i < s.cfg.WorkerCount; i++ {
		w := NewWorker(runCtx, s.pipeline, s.generators, s.kindRouting, s.ctxChan, out, s.reportError)
		w.Start()
		s.workers = append(s.workers, w)
	}
	go func() {
		s.scenario.Start(runCtx, s.metadata, s.sequencer, s.ctxChan)
		if reporter, ok := s.scenario.(runtimeErrorProvider); ok {
			if err := reporter.RuntimeError(); err != nil {
				s.reportError(err)
			}
		}
		if s.ctxChan != nil {
			close(s.ctxChan)
		}
		if s.finite != nil {
			summary := s.scenario.Summary()
			if summary == nil {
				summary = map[string]any{
					"pipeline": s.pipeline,
					"reason":   "scenario finished",
				}
			}
			s.finite.NotifyDone(summary)
		}
	}()
	s.errMu.Lock()
	defer s.errMu.Unlock()
	return s.err
}

func (s *Director) Err() error {
	s.errMu.Lock()
	defer s.errMu.Unlock()
	return s.err
}

func (s *Director) Close(forceExit bool) error {
	if s.cancel != nil {
		s.cancel()
	}
	if forceExit {
		for index := range s.workers {
			s.workers[index].Done()
		}
	}
	for index := range s.workers {
		s.workers[index].Close()
	}
	return nil
}

func (s *Director) reportError(err error) {
	if err == nil {
		return
	}
	s.errOnce.Do(func() {
		s.errMu.Lock()
		s.err = err
		s.errMu.Unlock()
		if s.cancel != nil {
			s.cancel()
		}
	})
}

// Done 返回完成信号，未开启有限能力时返回 nil。
func (s *Director) Done() <-chan struct{} {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.finite == nil {
		return nil
	}
	return s.finite.Done()
}

// Summary 返回一次性的执行摘要，未开启有限能力时返回 nil。
func (s *Director) Summary() map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.finite == nil {
		return nil
	}
	return s.finite.Summary()
}
