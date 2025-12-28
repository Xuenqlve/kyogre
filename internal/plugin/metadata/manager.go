package metadata

import (
	"fmt"
	"sync"

	"github.com/xuenqlve/kyogre/internal/config"
)

func NewManager() *Manager {
	return &Manager{
		storage: map[string]Metadata{},
	}
}

type Manager struct {
	pipeline string
	mu       sync.RWMutex
	storage  map[string]Metadata
}

func (m *Manager) Configure(pipeline string, data map[string]config.ConfigureMold) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.pipeline = pipeline
	for key, cfg := range data {
		iq, err := GetMetadata(MetadataType(cfg.Type))
		if err != nil {
			return err
		}
		if err = iq.Configure(pipeline, cfg.Config); err != nil {
			return err
		}
		m.storage[key] = iq
	}
	return nil
}

func (m *Manager) GetMetadata(key string) (Metadata, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	iq, ok := m.storage[key]
	if !ok {
		return nil, fmt.Errorf("metadata not found by key: %s", key)
	}
	return iq, nil
}
