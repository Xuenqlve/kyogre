package generator

import (
	"fmt"
	"reflect"
	"sync"

	"github.com/xuenqlve/kyogre/internal/message"
	"github.com/xuenqlve/kyogre/internal/plugin/iquery"
)

// GenerationMode 生成模式
type GenerationMode string

// GenerationContext 生成上下文（由 scenario 构造，generator 负责解析与生成消息）
// 具体实现可根据消息类型扩展，例如单表DML、事务消息等。
type GenerationContext interface {
	// Kind 用于路由到对应 generator
	Kind() string
	// Provider 返回主键/唯一键值的提供者，由 scenario 提供
	Provider(key string) iquery.Provider
	// Strategy 返回生成策略（可选）
	Strategy() *StrategySnapshot
	// Validate 校验上下文内容是否合理（由上层构造时完成）
	Validate() error
	// Extras 返回扩展参数（可选）
	Extras() map[string]any
}

// Generator 生成器接口（两阶段设计）
type Generator interface {
	Configure(pipeline string, data map[string]any) error
	// Kinds 返回该生成器支持的上下文类型（静态声明，不依赖配置）。
	Kinds() []string
	// 单阶段：基于上下文生成消息
	Generate(ctx GenerationContext) (message.Message, error)

	Close()
}

type (
	Type    string
	Factory func() Generator
)

var (
	_generatorRegistry map[Type]Factory
	_generatorMutex    sync.Mutex
)

// RegisterGenerator 注册生成器实例
func RegisterGenerator(generatorType Type, v Generator, singleton bool) {
	var gf Factory
	if singleton {
		gf = func() Generator { return v }
	} else {
		gf = func() Generator { return reflect.New(reflect.TypeOf(v).Elem()).Interface().(Generator) }
	}
	RegisterGeneratorFactory(generatorType, gf)
}

// RegisterGeneratorFactory 注册生成器工厂
func RegisterGeneratorFactory(generatorType Type, factory Factory) {
	_generatorMutex.Lock()
	defer _generatorMutex.Unlock()
	if _generatorRegistry == nil {
		_generatorRegistry = make(map[Type]Factory)
	}
	if _, ok := _generatorRegistry[generatorType]; ok {
		panic(fmt.Sprintf("generator already exists, type:%s", generatorType))
	}
	_generatorRegistry[generatorType] = factory
}

// GetGenerator 根据类型创建生成器实例
func GetGenerator(generatorType Type) (Generator, error) {
	_generatorMutex.Lock()
	defer _generatorMutex.Unlock()
	factory, ok := _generatorRegistry[generatorType]
	if !ok {
		return nil, fmt.Errorf("generator not registered type:%s", generatorType)
	}
	return factory(), nil
}
