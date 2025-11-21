package plugin

import (
	"encoding/json"
	"fmt"

	"github.com/mitchellh/mapstructure"
)

// ConfigLoader 配置加载器
// 支持从YAML/JSON等配置格式加载DependencyConfig
type ConfigLoader struct{}

// LoadDependencyConfig 从map中加载DependencyConfig
// 该函数支持动态加载不同类型的配置
func (cl *ConfigLoader) LoadDependencyConfig(data map[string]any) (DependencyConfig, error) {
	// 首先获取type字段来确定配置类型
	configType, ok := data["type"].(string)
	if !ok {
		return nil, fmt.Errorf("missing or invalid 'type' field in config")
	}

	switch configType {
	case "dml":
		return cl.loadDMLConfig(data)
	case "transaction":
		return cl.loadTransactionConfig(data)
	case "ddl":
		return cl.loadDDLConfig(data)
	default:
		return nil, fmt.Errorf("unknown config type: %s", configType)
	}
}

// loadDMLConfig 加载DMLConfig
func (cl *ConfigLoader) loadDMLConfig(data map[string]any) (*DMLConfig, error) {
	config := &DMLConfig{}

	if err := mapstructure.Decode(data, config); err != nil {
		return nil, fmt.Errorf("failed to decode DMLConfig: %w", err)
	}

	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("DMLConfig validation failed: %w", err)
	}

	return config, nil
}

// loadTransactionConfig 加载TransactionConfig
func (cl *ConfigLoader) loadTransactionConfig(data map[string]any) (*TransactionConfig, error) {
	config := &TransactionConfig{}

	if err := mapstructure.Decode(data, config); err != nil {
		return nil, fmt.Errorf("failed to decode TransactionConfig: %w", err)
	}

	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("TransactionConfig validation failed: %w", err)
	}

	return config, nil
}

// loadDDLConfig 加载DDLConfig
func (cl *ConfigLoader) loadDDLConfig(data map[string]any) (*DDLConfig, error) {
	config := &DDLConfig{}

	if err := mapstructure.Decode(data, config); err != nil {
		return nil, fmt.Errorf("failed to decode DDLConfig: %w", err)
	}

	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("DDLConfig validation failed: %w", err)
	}

	return config, nil
}

// LoadDependencyConfigFromJSON 从JSON字符串加载DependencyConfig
func (cl *ConfigLoader) LoadDependencyConfigFromJSON(jsonStr string) (DependencyConfig, error) {
	var data map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	return cl.LoadDependencyConfig(data)
}

// NewConfigLoader 创建配置加载器
func NewConfigLoader() *ConfigLoader {
	return &ConfigLoader{}
}
