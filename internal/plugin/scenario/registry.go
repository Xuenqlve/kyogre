package scenario

import (
	"context"
	"fmt"
	"reflect"
	"sync"

	"github.com/xuenqlve/kyogre/internal/message"
	"github.com/xuenqlve/kyogre/internal/plugin/generator"
	"github.com/xuenqlve/kyogre/internal/plugin/iquery"
)

type Scenario interface {
	Configure(pipeline string, data map[string]any) (err error)
	//RegisterIQueryLookup(lookup iquery.Lookup)
	RegisterSequencer(seq *iquery.Sequencer, specs []iquery.SequenceSpec)
	RegisterGenerator(gen generator.Generator)
	Preparation(ctx context.Context) error
	Start(msgChan message.InPoint) error
	Summary() map[string]any
	Done() <-chan struct{}
	Close() error
}

type (
	Type    string
	Factory func() Scenario
)

var (
	_scenarioRegistry map[Type]Factory
	_scenarioMutex    sync.Mutex
)

func RegisterScenario(scenarioType Type, v Scenario, singleton bool) {
	var sf Factory
	if singleton {
		sf = func() Scenario { return v }
	} else {
		sf = func() Scenario { return reflect.New(reflect.TypeOf(v).Elem()).Interface().(Scenario) }
	}
	RegisterScenarioFactory(scenarioType, sf)
}

func RegisterScenarioFactory(scenarioType Type, factory Factory) {
	_scenarioMutex.Lock()
	defer _scenarioMutex.Unlock()
	if _scenarioRegistry == nil {
		_scenarioRegistry = make(map[Type]Factory)
	}
	if _, ok := _scenarioRegistry[scenarioType]; ok {
		panic(fmt.Sprintf("scenario already exists, type:%s", scenarioType))
	}
	_scenarioRegistry[scenarioType] = factory
}

func GetScenario(scenarioType Type) (Scenario, error) {
	_scenarioMutex.Lock()
	defer _scenarioMutex.Unlock()
	factory, ok := _scenarioRegistry[scenarioType]
	if !ok {
		return nil, fmt.Errorf("scenario not registered type:%s", scenarioType)
	}
	return factory(), nil
}
