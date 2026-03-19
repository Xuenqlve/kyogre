package base

import (
	"fmt"
	"reflect"
	"sync"

	"github.com/xuenqlve/kyogre/internal/plugin/generator"
	"github.com/xuenqlve/kyogre/internal/plugin/metadata"
)

type BuilderType string

// ContextBuilder 负责将通用 Plan 转换为数据库专属的 GenerationContext。
// base scenario 只做策略与计划编排，具体上下文由 builder 处理。
type ContextBuilder interface {
	Configure(pipeline string, data map[string]any) error
	LoadTargets(md metadata.Metadata, names []string) ([]Target, error)
	Build(plan Plan) (generator.GenerationContext, error)
	Close() error
}

type BuilderFactory func() ContextBuilder

var (
	builderRegistry map[BuilderType]BuilderFactory
	builderMutex    sync.Mutex
)

func RegisterBuilder(builderType BuilderType, v ContextBuilder, singleton bool) {
	var factory BuilderFactory
	if singleton {
		factory = func() ContextBuilder { return v }
	} else {
		factory = func() ContextBuilder { return reflect.New(reflect.TypeOf(v).Elem()).Interface().(ContextBuilder) }
	}
	RegisterBuilderFactory(builderType, factory)
}

func RegisterBuilderFactory(builderType BuilderType, factory BuilderFactory) {
	builderMutex.Lock()
	defer builderMutex.Unlock()
	if builderRegistry == nil {
		builderRegistry = make(map[BuilderType]BuilderFactory)
	}
	if _, ok := builderRegistry[builderType]; ok {
		panic(fmt.Sprintf("scenario builder already exists, type:%s", builderType))
	}
	builderRegistry[builderType] = factory
}

func GetBuilder(builderType BuilderType) (ContextBuilder, error) {
	builderMutex.Lock()
	defer builderMutex.Unlock()
	factory, ok := builderRegistry[builderType]
	if !ok {
		return nil, fmt.Errorf("scenario builder not registered type:%s", builderType)
	}
	return factory(), nil
}
