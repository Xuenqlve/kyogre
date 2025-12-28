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
		storage: make(map[string]config.ScenarioConfigureMold),
	}
}

type Manager struct {
	pipeline   string
	mux        sync.RWMutex
	storage    map[string]config.ScenarioConfigureMold
	generators *generator.Manager
	iquery     *iquery.Manager
}

func (m *Manager) Configure(pipeline string, data map[string]config.ScenarioConfigureMold, gManager *generator.Manager, iqManager *iquery.Manager) error {
	m.mux.Lock()
	defer m.mux.Unlock()
	m.pipeline = pipeline
	m.generators = gManager
	m.iquery = iqManager
	for key, cfg := range data {
		if _, err := m.buildScenario(cfg); err != nil {
			return err
		}
		m.storage[key] = cfg
	}
	return nil
}

func (m *Manager) GetScenario(key string) (Scenario, error) {
	m.mux.RLock()
	cfg, ok := m.storage[key]
	m.mux.RUnlock()
	if !ok {
		return nil, fmt.Errorf("scenario '%s' not found", key)
	}
	return m.buildScenario(cfg)
}

func (m *Manager) buildScenario(cfg config.ScenarioConfigureMold) (Scenario, error) {
	sc, err := GetScenario(Type(cfg.Type))
	if err != nil {
		return nil, err
	}
	if err = sc.Configure(m.pipeline, cfg.Config); err != nil {
		return nil, err
	}
	if cfg.IQuery != "" {
		if m.iquery == nil {
			return nil, fmt.Errorf("iquery manager not initialized when binding scenario '%s'", cfg.IQuery)
		}
		lookup, err := m.iquery.GetIQueryLookup(cfg.IQuery)
		if err != nil {
			return nil, err
		}
		sc.RegisterIQueryLookup(lookup)
	}
	for _, generatorKey := range cfg.Generators {
		if m.generators == nil {
			return nil, fmt.Errorf("generator manager not initialized when binding generator '%s'", generatorKey)
		}
		g, err := m.generators.GetGenerator(generatorKey)
		if err != nil {
			return nil, err
		}
		sc.RegisterGenerator(g)
	}
	return sc, nil
}
