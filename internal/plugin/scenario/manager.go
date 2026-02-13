package scenario

import (
	"fmt"
	"sync"

	"github.com/xuenqlve/kyogre/internal/config"
	"github.com/xuenqlve/kyogre/internal/plugin/generator"
	"github.com/xuenqlve/kyogre/internal/plugin/iquery"
	"github.com/xuenqlve/kyogre/internal/plugin/metadata"
)

func NewManager() *Manager {
	return &Manager{
		storage: make(map[string]*Director),
	}
}

type Manager struct {
	pipeline   string
	mux        sync.RWMutex
	storage    map[string]*Director
	generators *generator.Manager
	iQuery     *iquery.Manager
	metadata   *metadata.Manager
}

func (m *Manager) Configure(pipeline string, data map[string]config.ScenarioConfigureMold, gManager *generator.Manager, iqManager *iquery.Manager, mdManager *metadata.Manager) error {
	m.mux.Lock()
	defer m.mux.Unlock()
	m.pipeline = pipeline
	m.generators = gManager
	m.iQuery = iqManager
	m.metadata = mdManager
	for key, cfg := range data {
		supervisor, err := m.buildScenario(cfg)
		if err != nil {
			return err
		}
		m.storage[key] = supervisor
	}
	return nil
}

func (m *Manager) GetDirector(key string) (*Director, error) {
	m.mux.RLock()
	defer m.mux.RUnlock()
	supervisor, ok := m.storage[key]
	if !ok {
		return nil, fmt.Errorf("scenario '%s' not found", key)
	}
	return supervisor, nil
}

func (m *Manager) buildScenario(cfg config.ScenarioConfigureMold) (*Director, error) {
	director := NewDirector()
	err := director.Configure(m.pipeline, cfg.Type, cfg.Config)
	if err != nil {
		return nil, err
	}
	for _, generatorKey := range cfg.Generators {
		if m.generators == nil {
			return nil, fmt.Errorf("generator manager not initialized when binding generator '%s'", generatorKey)
		}
		var g generator.Generator
		if g, err = m.generators.GetGenerator(generatorKey); err != nil {
			return nil, err
		}
		director.RegisterGenerator(generatorKey, g)
	}
	meta, err := m.metadata.GetMetadata(cfg.Metadata)
	if err != nil {
		return nil, err
	}
	director.RegisterMetadata(meta)

	if cfg.IQuery != "" {
		if err = m.wireSequencer(director, cfg); err != nil {
			return nil, err
		}
	}
	return director, nil
}

func (m *Manager) wireSequencer(sc *Director, cfg config.ScenarioConfigureMold) error {
	if m.iQuery == nil || m.metadata == nil || m.generators == nil {
		return fmt.Errorf("sequencer wiring requires lookup/metadata/generator managers initialized")
	}
	lookup, err := m.iQuery.GetIQueryLookup(cfg.IQuery)
	if err != nil {
		return err
	}

	seq, err := iquery.NewSequencer()
	if err != nil {
		return err
	}
	seq.BindLookup(lookup)
	// worker-count 在不同 scenario 中可能来源不同；此处仅做 best-effort 兜底，后续可抽象为通用接口。
	//workerCount := getWorkerCount(cfg.Config)
	if err = seq.Configure(m.pipeline, iquery.WithLookUpKey(cfg.IQuery)); err != nil {
		return err
	}

	specs, err := m.buildSequenceSpecs(cfg)
	if err != nil {
		return err
	}
	sc.RegisterSequencer(seq, specs)
	return nil
}

func (m *Manager) buildSequenceSpecs(cfg config.ScenarioConfigureMold) ([]iquery.SequenceSpec, error) {
	md, err := m.metadata.GetMetadata(cfg.Metadata)
	if err != nil {
		return nil, err
	}
	if !md.IQueryEnabled() {
		return nil, nil
	}
	specs := make([]iquery.SequenceSpec, 0, len(md.SchemaKeys()))
	for _, schemaKey := range md.SchemaKeys() {
		fields := []iquery.BoundParam{}
		if fields, err = md.SchemaPrimaryField(schemaKey); err != nil {
			return nil, err
		}
		specs = append(specs, iquery.MakeSequenceSpec(schemaKey, fields))
	}
	return specs, nil
}
