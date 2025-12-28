package iquery

import (
	"fmt"
	"sync"

	"github.com/xuenqlve/kyogre/internal/config"
)

func IQueryManager(pipeline string, data map[string]config.ConfigureMold) (*Manager, error) {
	manager := &Manager{
		pipeline: pipeline,
		storage:  make(map[string]Lookup),
	}

	for key, cfg := range data {
		iq, err := GetIQueryModule(LookupType(cfg.Type))
		if err != nil {
			return nil, err
		}
		if err = iq.Configure(pipeline, cfg.Config); err != nil {
			return nil, err
		}
		manager.storage[key] = iq
	}
	return manager, nil
}

type Manager struct {
	pipeline string
	mux      sync.RWMutex
	storage  map[string]Lookup
}

// GetIQueryLookup 按 key 获取已初始化的反查实例
func (e *Manager) GetIQueryLookup(key string) (Lookup, error) {
	e.mux.RLock()
	defer e.mux.RUnlock()
	m, exist := e.storage[key]
	if !exist {
		return nil, fmt.Errorf("iquery not found by key: %s", key)
	}
	return m, nil
}
