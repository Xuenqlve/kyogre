package metadata

import (
	"fmt"
	"sync"

	"github.com/xuenqlve/kyogre/internal/config"
)

var MetaData StorageEngine

type StorageEngine struct {
	pipeline string
	mux      sync.RWMutex
	engines  map[string]Metadata
}

func (e *StorageEngine) Configure(pipeline string, data map[string]config.ConfigureMold) error {
	e.mux.Lock()
	defer e.mux.Unlock()
	e.pipeline = pipeline
	for key, cfg := range data {
		metadata, err := GetMetadata(MetadataType(cfg.Type))
		if err != nil {
			return err
		}
		if err = metadata.Configure(pipeline, cfg.Config); err != nil {
			return err
		}
		e.engines[key] = metadata
	}
	return nil
}

func (e *StorageEngine) GetMetaData(key string) (Metadata, error) {
	e.mux.RLock()
	defer e.mux.RUnlock()
	m, exist := e.engines[key]
	if !exist {
		return nil, fmt.Errorf("metadata not found by key: %s", key)
	}
	return m, nil
}
