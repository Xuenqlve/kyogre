package mysql_row

import (
	"github.com/xuenqlve/common/errors"
)

type Config struct {
	DataSource        string `mapstructure:"data-source" json:"data-source" yaml:"data-source" toml:"data-source"`
	WorkerCount       int    `mapstructure:"worker-count" json:"worker-count" yaml:"worker-count" toml:"worker-count" `
	WorkerQueueLength int    `mapstructure:"worker-queue-length" json:"worker-queue-length" toml:"worker-queue-length" yaml:"worker-queue-length"`
}

func (cfg *Config) Validate() error {
	if cfg.DataSource == "" {
		return errors.New("data-source is required")
	}
	if cfg.WorkerCount == 0 {
		cfg.WorkerCount = 1
	}
	if cfg.WorkerQueueLength == 0 {
		cfg.WorkerQueueLength = 10
	}
	return nil
}
