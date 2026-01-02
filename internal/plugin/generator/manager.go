package generator

import (
	"fmt"
	"sync"

	"github.com/xuenqlve/common/errors"
	"github.com/xuenqlve/kyogre/internal/config"
)

//func ManagerGenerator(pipeline string, data map[string]config.ConfigureMold, metadataHit map[string]string, manager *metadata.Manager) (*Manager, error) {
//	m := &Manager{
//		pipeline: pipeline,
//		storage:  map[string]Generator{},
//	}
//	for key, cfg := range data {
//		generator, err := GetGenerator(Type(cfg.Type))
//		if err != nil {
//			return nil, errors.Trace(err)
//		}
//		metadataKey, exist := metadataHit[key]
//		if !exist {
//			return nil, errors.Errorf("metadata generator not found for key: %s", key)
//		}
//		md, err := manager.GetMetadata(metadataKey)
//		if err != nil {
//			return nil, errors.Trace(err)
//		}
//		if err = generator.Configure(pipeline, cfg.Config); err != nil {
//			return nil, err
//		}
//		generator.RegisterMetadata(md)
//		m.storage[key] = generator
//	}
//	return m, nil
//}

func NewManager() *Manager {
	return &Manager{
		storage: map[string]Generator{},
	}
}

type Manager struct {
	pipeline string
	mu       sync.RWMutex
	storage  map[string]Generator
}

func (m *Manager) Configure(pipeline string, data map[string]config.ConfigureMold) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.pipeline = pipeline
	for key, cfg := range data {
		generator, err := GetGenerator(Type(cfg.Type))
		if err != nil {
			return errors.Trace(err)
		}
		if err = generator.Configure(pipeline, cfg.Config); err != nil {
			return err
		}
		m.storage[key] = generator
	}
	return nil
}

func (m *Manager) GetGenerator(key string) (Generator, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	generator, ok := m.storage[key]
	if !ok {
		return nil, fmt.Errorf("generator %s not found", key)
	}
	return generator, nil
}
