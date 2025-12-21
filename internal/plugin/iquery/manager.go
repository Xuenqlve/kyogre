package iquery

import (
	"fmt"
	"sync"

	"github.com/xuenqlve/kyogre/internal/config"
)

// IQueryManager 负责管理多个反查实例，便于按 key 使用
var IQueryManager = &Manager{
	engines: make(map[string]Lookup),
}

type Manager struct {
	pipeline string
	mux      sync.RWMutex
	engines  map[string]Lookup
}

// Configure 根据配置初始化多个反查插件
func (e *Manager) Configure(pipeline string, data map[string]config.ConfigureMold) error {
	e.mux.Lock()
	defer e.mux.Unlock()
	e.pipeline = pipeline
	if e.engines == nil {
		e.engines = make(map[string]Lookup)
	}

	for key, cfg := range data {
		iq, err := GetIQueryModule(LookupType(cfg.Type))
		if err != nil {
			return err
		}
		if err = iq.Configure(pipeline, cfg.Config); err != nil {
			return err
		}
		e.engines[key] = iq
	}
	return nil
}

// GetIQueryLookup 按 key 获取已初始化的反查实例
func (e *Manager) GetIQueryLookup(key string) (Lookup, error) {
	e.mux.RLock()
	defer e.mux.RUnlock()
	m, exist := e.engines[key]
	if !exist {
		return nil, fmt.Errorf("iquery not found by key: %s", key)
	}
	return m, nil
}
