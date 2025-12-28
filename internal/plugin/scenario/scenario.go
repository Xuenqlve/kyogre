package scenario

import (
	"fmt"
	"sync"

	"github.com/xuenqlve/kyogre/internal/config"
	"github.com/xuenqlve/kyogre/internal/plugin/generator"
	"github.com/xuenqlve/kyogre/internal/plugin/iquery"
)

func NewManager() *Manager {
	return &Manager{
		storage: make(map[string]Scenario),
	}
}

type Manager struct {
	pipeline string
	mux      sync.RWMutex
	storage  map[string]Scenario
}

func (m *Manager) Configure(pipeline string, data map[string]config.ScenarioConfigureMold, gManager *generator.Manager, iqManager *iquery.Manager) error {
	m.mux.Lock()
	defer m.mux.Unlock()
	m.pipeline = pipeline
	for key, cfg := range data {
		scenario, err := GetScenario(Type(cfg.Type))
		if err != nil {
			return err
		}
		if err = scenario.Configure(pipeline, cfg.Config); err != nil {
			return err
		}
		if cfg.IQuery != "" {
			var lookup iquery.Lookup
			if lookup, err = iqManager.GetIQueryLookup(cfg.IQuery); err != nil {
				return err
			}
			scenario.RegisterIQueryLookup(lookup)
		}

		for _, generatorKey := range cfg.Generators {
			var g generator.Generator
			if g, err = gManager.GetGenerator(generatorKey); err != nil {
				return err
			}
			scenario.RegisterGenerator(g)
		}

		m.storage[key] = scenario
	}
	return nil
}

func (m *Manager) GetScenario(key string) (Scenario, error) {
	m.mux.RLock()
	defer m.mux.RUnlock()
	scenario, ok := m.storage[key]
	if !ok {
		return nil, fmt.Errorf("scenario '%s' not found", key)
	}
	return scenario, nil
}
