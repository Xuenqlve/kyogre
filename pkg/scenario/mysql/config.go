package mysql

import (
	"fmt"

	"github.com/xuenqlve/kyogre/internal/plugin"
	"github.com/xuenqlve/kyogre/pkg/generator/query_module"
)

type Config struct {
	WorkerCount        int                                `mapstructure:"worker-count" yaml:"worker-count"`
	QueryModule        *query_module.QueryModuleConfig   `mapstructure:"query-module" yaml:"query-module"`                   // 反查模块配置（可选）
	GenerationStrategy *plugin.GenerationStrategy        `mapstructure:"generation-strategy" yaml:"generation-strategy"`     // 生成策略

	// 依赖配置：支持DML、Transaction、DDL三种模式
	DependencyConfig   DependencyConfigData              `mapstructure:"dependency-config" yaml:"dependency-config"`         // 依赖配置
}

// DependencyConfigData 依赖配置数据（用于YAML/JSON加载）
type DependencyConfigData struct {
	Type              string         `mapstructure:"type" yaml:"type"`                       // 配置类型: dml/transaction/ddl
	DML               *plugin.DMLConfig                   `mapstructure:"dml" yaml:"dml"`                                     // DML配置
	Transaction       *plugin.TransactionConfig          `mapstructure:"transaction" yaml:"transaction"`                     // 事务配置
	DDL               *plugin.DDLConfig                   `mapstructure:"ddl" yaml:"ddl"`                                     // DDL配置
}

// GetDependencyConfig 获取对应类型的DependencyConfig
func (d *DependencyConfigData) GetDependencyConfig() (plugin.DependencyConfig, error) {
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
	if c.GenerationStrategy == nil {
		c.GenerationStrategy = &plugin.GenerationStrategy{
			RandomConfig: &plugin.RandomConfig{},
		}
	}

	// 设置默认的依赖配置（如果没有指定）
	if c.DependencyConfig.Type == "" {
		c.DependencyConfig.Type = "dml"
		if c.DependencyConfig.DML == nil {
			c.DependencyConfig.DML = &plugin.DMLConfig{
				Operation:   "insert",
				TableSelect: "random",
				Count:       "1",
			}
		}
	}

	return nil
}
