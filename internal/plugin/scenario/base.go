package scenario

import (
	"sync"

	"github.com/xuenqlve/kyogre/internal/plugin/generator"
	"github.com/xuenqlve/kyogre/internal/plugin/iquery"
)

type BaseScenario struct {
	mu           sync.Mutex
	pipeline     string
	iQueryLookup iquery.Lookup
	generators   []generator.Generator
}

func (s *BaseScenario) Configure(pipeline string) (err error) {
	s.pipeline = pipeline
	s.generators = make([]generator.Generator, 0)
	return
}

func (s *BaseScenario) Pipeline() string {
	return s.pipeline
}

func (s *BaseScenario) RegisterIQueryLookup(lookup iquery.Lookup) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.iQueryLookup = lookup
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

func (s *BaseScenario) IQueryLookup() iquery.Lookup {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.iQueryLookup
}
