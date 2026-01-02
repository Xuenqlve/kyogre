package scenario

import (
	"sync"

	"github.com/xuenqlve/kyogre/internal/plugin/generator"
	"github.com/xuenqlve/kyogre/internal/plugin/iquery"
)

type BaseScenario struct {
	mu       sync.Mutex
	pipeline string
	//iQueryLookup iquery.Lookup
	generators   []generator.Generator
	finite       *FiniteHelper
	sequencer    *iquery.Sequencer
	sequenceSpec []iquery.SequenceSpec
}

func NewBaseScenario(pipeline string) *BaseScenario {
	return &BaseScenario{
		pipeline:   pipeline,
		generators: make([]generator.Generator, 0),
	}
}

func (s *BaseScenario) Pipeline() string {
	return s.pipeline
}

//func (s *BaseScenario) RegisterIQueryLookup(lookup lookup.Lookup) {
//	s.mu.Lock()
//	defer s.mu.Unlock()
//	s.iQueryLookup = lookup
//}

// RegisterSequencer 在 internal 组装阶段注入 sequencer 以及需要管理的序列信息。
// 场景实现可在 Start 阶段从 Sequencer() 获取并按 worker 分配区间。
func (s *BaseScenario) RegisterSequencer(seq *iquery.Sequencer, specs []iquery.SequenceSpec) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sequencer = seq
	s.sequenceSpec = specs
}

func (s *BaseScenario) RegisterGenerator(gen generator.Generator) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.generators = append(s.generators, gen)
}

func (s *BaseScenario) Generators() (list []generator.Generator) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.generators
}

//func (s *BaseScenario) IQueryLookup() lookup.Lookup {
//	s.mu.Lock()
//	defer s.mu.Unlock()
//	return s.iQueryLookup
//}

func (s *BaseScenario) Sequencer() *iquery.Sequencer {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.sequencer
}

func (s *BaseScenario) SequenceSpecs() []iquery.SequenceSpec {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]iquery.SequenceSpec(nil), s.sequenceSpec...)
}

// EnableFinite 显式开启有限场景能力，返回可用的 FiniteHelper。
func (s *BaseScenario) EnableFinite() *FiniteHelper {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.finite == nil {
		s.finite = NewFiniteHelper()
	}
	return s.finite
}

// Done 返回完成信号，未开启有限能力时返回 nil。
func (s *BaseScenario) Done() <-chan struct{} {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.finite == nil {
		return nil
	}
	return s.finite.Done()
}

// Summary 返回一次性的执行摘要，未开启有限能力时返回 nil。
func (s *BaseScenario) Summary() map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.finite == nil {
		return nil
	}
	return s.finite.Summary()
}
