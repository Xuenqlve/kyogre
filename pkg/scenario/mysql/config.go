package mysql

import (
	"fmt"

	"github.com/xuenqlve/kyogre/internal/config"
	"github.com/xuenqlve/kyogre/internal/plugin/generator"
	"github.com/xuenqlve/kyogre/pkg/generator/mysql"
)

type Config struct {
	WorkerCount        int                           `mapstructure:"worker-count" yaml:"worker-count"`
	IQueryKey          string                        `mapstructure:"iquery-key" yaml:"iquery-key"`
	Metadata           string                        `mapstructure:"metadata" yaml:"metadata"`
	GenerationStrategy *generator.GenerationStrategy `mapstructure:"generation-strategy" yaml:"generation-strategy"` // 生成策略
	// 依赖配置：支持DML、Transaction、DDL三种模式
	DependencyConfig DependencyConfigData `mapstructure:"dependency-config" yaml:"dependency-config"` // 依赖配置
}

// IQueryConfig 反查配置
type IQueryConfig struct {
	Enabled bool                  `mapstructure:"enabled" yaml:"enabled"`
	Key     string                `mapstructure:"key" yaml:"key"`
	Module  *config.ConfigureMold `mapstructure:"module,omitempty" yaml:"module,omitempty"`
}

// DependencyConfigData 依赖配置数据（用于YAML/JSON加载）
type DependencyConfigData struct {
	Type        string                   `mapstructure:"type" yaml:"type"`               // 配置类型: dml/transaction/ddl
	DML         *mysql.DMLConfig         `mapstructure:"dml" yaml:"dml"`                 // DML配置
	Transaction *mysql.TransactionConfig `mapstructure:"transaction" yaml:"transaction"` // 事务配置
	DDL         *mysql.DDLConfig         `mapstructure:"ddl" yaml:"ddl"`                 // DDL配置
}

// GetDependencyConfig 获取对应类型的DependencyConfig
func (d *DependencyConfigData) GetDependencyConfig() (generator.DependencyConfig, error) {
	if err := d.Normalize(); err != nil {
		return nil, err
	}

	switch d.Type {
	case "dml":
		if d.DML == nil {
			return nil, fmt.Errorf("dml config is required when type is dml")
		}
		return d.DML, nil
	case "transaction":
		if d.Transaction == nil {
			return nil, fmt.Errorf("transaction config is required when type is transaction")
		}
		return d.Transaction, nil
	case "ddl":
		if d.DDL == nil {
			return nil, fmt.Errorf("ddl config is required when type is ddl")
		}
		return d.DDL, nil
	default:
		return nil, fmt.Errorf("unknown dependency config type: %s", d.Type)
	}
}

func (c *Config) Validate() error {
	if c.WorkerCount == 0 {
		c.WorkerCount = 1
	}

	// 设置默认的生成策略（如果没有指定）

	// 规范化依赖配置并做完整校验
	if err := c.DependencyConfig.Normalize(); err != nil {
		return err
	}

	return nil
}

// Validate 校验并补全IQuery配置
func (q *IQueryConfig) Validate() error {
	if !q.Enabled {
		return nil
	}

	if q.Key == "" && q.Module == nil {
		return fmt.Errorf("enable-iquery 为 true 时需要配置 key 或 module")
	}

	if q.Key != "" && q.Module != nil {
		return fmt.Errorf("iquery key 与 module 只能二选一")
	}

	if q.Module != nil && q.Module.Type == "" {
		return fmt.Errorf("iquery module.type 不能为空")
	}

	return nil
}

// mergeLegacy 将旧版的扁平字段合并进新版配置（仅在新版字段缺失时）
func (q *IQueryConfig) mergeLegacy(enabled bool, key string, module *config.ConfigureMold) {
	if !q.Enabled {
		q.Enabled = enabled
	}
	if q.Key == "" {
		q.Key = key
	}
	if q.Module == nil {
		q.Module = module
	}
}

// Normalize 填充默认值并校验依赖配置
func (d *DependencyConfigData) Normalize() error {
	if d.Type == "" {
		d.Type = "dml"
	}
	switch d.Type {
	case "dml":
		if d.DML == nil {
			d.DML = &mysql.DMLConfig{
				Operation:   "insert",
				TableSelect: generator.RandomTableSelect,
				Count:       "1",
			}
		}
		return d.DML.Validate()
	case "transaction":
		if d.Transaction == nil {
			return fmt.Errorf("transaction config is required when type is transaction")
		}
		return d.Transaction.Validate()
	case "ddl":
		if d.DDL == nil {
			return fmt.Errorf("ddl config is required when type is ddl")
		}
		return d.DDL.Validate()
	default:
		return fmt.Errorf("unknown dependency config type: %s", d.Type)
	}
}
