package config

import (
	"fmt"
	"sync"

	"github.com/mitchellh/mapstructure"
	"github.com/xuenqlve/common/transform"
	"github.com/xuenqlve/kyogre/internal/config"
)

type FileConfig struct {
	mu       sync.Mutex
	filePath string
	cfg      map[string]any
}

func init() {
	config.RegisterConfig(config.DefaultConfigType, &FileConfig{}, true)
}

func (f *FileConfig) Configure(data map[string]any) (err error) {
	filePath, ok := data[config.FilePathKey]
	if !ok {
		return fmt.Errorf("file-config not found key:%s", config.FilePathKey)
	}
	f.filePath, ok = filePath.(string)
	if !ok {
		return fmt.Errorf("file-config key:%s %v to string fail", config.FilePathKey, filePath)
	}
	f.cfg, err = transform.ConfigFromFile(f.filePath)
	if err != nil {
		return err
	}
	return nil
}

func (f *FileConfig) Load() (config.Config, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	cfg := config.Config{}
	err := mapstructure.Decode(f.cfg, &cfg)
	return cfg, err
}

func (f *FileConfig) Close() error {
	return nil
}
