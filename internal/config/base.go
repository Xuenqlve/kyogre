package config

import (
	"fmt"
	"reflect"
	"sync"
)

type Config struct {
	Name       string                    `mapstructure:"name" json:"name" yaml:"name" toml:"name"`
	LogLevel   string                    `mapstructure:"log-level" json:"log-level" yaml:"log-level" toml:"log-level"`
	LogFile    string                    `mapstructure:"log-file" json:"log-file" yaml:"log-file" toml:"log-file"`
	ApiConfig  map[string]any            `mapstructure:"api-config" json:"api-config" yaml:"api-config" toml:"api-config"`
	DataSource map[string]map[string]any `mapstructure:"data-source" json:"data-source" yaml:"data-source" toml:"data-source"`
	Metadata   map[string]ConfigureMold  `mapstructure:"metadata" json:"metadata" yaml:"metadata" toml:"metadata"`
	// 场景配置
	Scenario ConfigureMold `mapstructure:"scenario" json:"scenario" yaml:"scenario" toml:"scenario"`
	// 压测配置
	Pressure ConfigureMold `mapstructure:"pressure" json:"pressure" yaml:"pressure" toml:"pressure"`
	// 监控和报告配置
	Monitor ConfigureMold `mapstructure:"monitor" json:"monitor" yaml:"monitor" toml:"monitor"`
}

type ConfigureMold struct {
	Type   string         `json:"type,omitempty" yaml:"type,omitempty" toml:"type,omitempty"`
	Config map[string]any `json:"config,omitempty" yaml:"config,omitempty" toml:"config,omitempty"`
}

type Manager interface {
	Configure(data map[string]any) error
	Load() (Config, error)
}

type (
	Factory func() Manager
	Type    string
)

var (
	_config_registry map[Type]Factory
	_config_mutext   sync.Mutex
)

func RegisterConfigFactory(configType Type, configFactory Factory) {
	_config_mutext.Lock()
	defer _config_mutext.Unlock()
	if _config_registry == nil {
		_config_registry = make(map[Type]Factory)
	}
	_, ok := _config_registry[configType]
	if ok {
		panic(fmt.Sprintf("config factory already exists, type:%v ", configType))
	}

	_config_registry[configType] = configFactory
}

func RegisterConfig(configType Type, config Manager, singleton bool) {
	var configFactory Factory
	if singleton {
		configFactory = func() Manager {
			return config
		}
	} else {
		configFactory = func() Manager {
			return reflect.New(reflect.TypeOf(config).Elem()).Interface().(Manager)
		}
	}
	RegisterConfigFactory(configType, configFactory)
}

func GetConfig(configType Type) (Manager, error) {
	_config_mutext.Lock()
	defer _config_mutext.Unlock()

	factory, ok := _config_registry[configType]
	if !ok {
		return nil, fmt.Errorf("config empty plugin name: %v", configType)
	}
	return factory(), nil
}
