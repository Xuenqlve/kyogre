package pressure

import (
	"fmt"
	"sync"

	"github.com/xuenqlve/common/errors"
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

func (m *Manager) Configure(pipeline string, data map[string]config.ConfigureMold) error {
	m.mux.Lock()
	defer m.mux.Unlock()
	m.pipeline = pipeline
	for key, cfg := range data {
		pressure, err := m.newPressure(cfg)
		if err != nil {
			return errors.Trace(err)
		}
		m.storage[key] = pressure
	}
	return nil
}

func (m *Manager) GetPressureController(keys []string) (*Controller, error) {
	m.mux.RLock()
	defer m.mux.RUnlock()
	pressures := make([]Pressure, 0, len(keys))
	for _, key := range keys {
		pressure, ok := m.storage[key]
		if !ok {
			return nil, fmt.Errorf("pressure '%s' not found", key)
		}
		pressures = append(pressures, pressure)
	}
	return NewController(pressures), nil
}

func (m *Manager) newPressure(cfg config.ConfigureMold) (Pressure, error) {
	p, err := GetPressure(PressureType(cfg.Type))
	if err != nil {
		return nil, errors.Trace(err)
	}
	if err = p.Configure(m.pipeline, cfg.Config); err != nil {
		return nil, errors.Trace(err)
	}
	return p, nil
}
