package pressure

import (
	"fmt"
	"strings"
	"sync"

	"github.com/juju/errors"
	"github.com/xuenqlve/kyogre/internal/config"
)

func NewManager() *Manager {
	return &Manager{
		storage: make(map[string]Pressure),
	}
}

type Manager struct {
	pipeline string
	mux      sync.RWMutex
	storage  map[string]Pressure
}

func (m *Manager) Configure(pipeline string, data map[string]config.GeneratorConfigureMold) (err error) {
	m.mux.Lock()
	defer m.mux.Unlock()
	for key, cfg := range data {
		var q Pressure
		q, err = GetPressure(PressureType(cfg.Type))
		if err != nil {
			return errors.Trace(err)
		}
		if err = q.Configure(pipeline, cfg.Config); err != nil {
			return errors.Trace(err)
		}
		m.storage[key] = q
	}
	return nil
}

func (m *Manager) GetPressure(keys []string) (Pressure, error) {
	m.mux.RLock()
	defer m.mux.RUnlock()
	for _, key := range keys {
		p, ok := m.storage[key]
		if !ok {
			return nil, fmt.Errorf("pressure '%s' not found", key)
		}
	}

	return p, nil
}
