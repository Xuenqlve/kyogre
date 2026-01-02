package scenario

import (
	"fmt"
	"sync"

	"github.com/xuenqlve/kyogre/internal/config"
	"github.com/xuenqlve/kyogre/internal/models"
	"github.com/xuenqlve/kyogre/internal/plugin/generator"
	"github.com/xuenqlve/kyogre/internal/plugin/iquery"
	"github.com/xuenqlve/kyogre/internal/plugin/metadata"
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
	metadata   *metadata.Manager
}

func (m *Manager) Configure(pipeline string, data map[string]config.ScenarioConfigureMold, gManager *generator.Manager, iqManager *iquery.Manager, mdManager *metadata.Manager) error {
	m.mux.Lock()
	defer m.mux.Unlock()
	m.pipeline = pipeline
	m.generators = gManager
	m.iquery = iqManager
	m.metadata = mdManager
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

	for _, generatorKey := range cfg.Generators {
		if m.generators == nil {
			return nil, fmt.Errorf("generator manager not initialized when binding generator '%s'", generatorKey)
		}
		var g generator.Generator
		g, err = m.generators.GetGenerator(generatorKey)
		if err != nil {
			return nil, err
		}
		sc.RegisterGenerator(g)
	}

	if cfg.IQuery != "" {
		if err = m.wireSequencer(sc, cfg); err != nil {
			return nil, err
		}
	}
	return sc, nil
}

//type sequencerRegistrar interface {
//	RegisterSequencer(seq *iquery.Sequencer, specs []iquery.SequenceSpec)
//}

func (m *Manager) wireSequencer(sc Scenario, cfg config.ScenarioConfigureMold) error {
	if m.iquery == nil || m.metadata == nil || m.generators == nil {
		return fmt.Errorf("sequencer wiring requires iquery/metadata/generator managers initialized")
	}
	lookup, err := m.iquery.GetIQueryLookup(cfg.IQuery)
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
	metadataKeys := make(map[string]struct{})
	for _, generatorKey := range cfg.Generators {
		mold, ok := m.generators.GetConfigureMold(generatorKey)
		if !ok || mold.Metadata == "" {
			continue
		}
		metadataKeys[mold.Metadata] = struct{}{}
	}
	specs := make([]iquery.SequenceSpec, 0)
	for metadataKey := range metadataKeys {
		md, err := m.metadata.GetMetadata(metadataKey)
		if err != nil {
			return nil, err
		}
		if !md.IQueryEnabled() {
			continue
		}
		for _, schemaKey := range md.SchemaKeys() {
			fields := []models.FieldParam{}
			if fields, err = md.SchemaPrimaryField(schemaKey); err != nil {
				return nil, err
			}
			for _, field := range fields {
				specs = append(specs, iquery.SequenceSpec{
					Schema: schemaKey,
					Field:  field,
				})
			}
		}
	}
	return specs, nil
}
