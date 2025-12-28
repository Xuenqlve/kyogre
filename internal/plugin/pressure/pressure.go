package pressure

import (
	"context"
	"fmt"
	"reflect"
	"sync"

	"github.com/xuenqlve/kyogre/internal/message"
)

type Pressure interface {
	Configure(pipeline string, data map[string]any) (err error)
	MessageType() string
	Start(ctx context.Context) error
	Execute(msg message.Message)
	Close() error
}

type (
	PressureType    string
	PressureFactory func() Pressure
)

var (
	_pressure_registry map[PressureType]PressureFactory
	_pressure_mutex    sync.Mutex
)

func RegisterPressurePlugin(pressureType PressureType, factory PressureFactory) {
	_pressure_mutex.Lock()
	defer _pressure_mutex.Unlock()
	if _pressure_registry == nil {
		_pressure_registry = make(map[PressureType]PressureFactory)
	}
	if _, ok := _pressure_registry[pressureType]; ok {
		panic("pressure plugin already registered")
	}
	_pressure_registry[pressureType] = factory
}

func RegisterPressure(pressureType PressureType, v Pressure, singleton bool) {
	var pf PressureFactory
	if singleton {
		pf = func() Pressure { return v }
	} else {
		pf = func() Pressure { return reflect.New(reflect.TypeOf(v).Elem()).Interface().(Pressure) }
	}
	RegisterPressurePlugin(pressureType, pf)
}

func GetPressure(pressureType PressureType) (Pressure, error) {
	_pressure_mutex.Lock()
	defer _pressure_mutex.Unlock()
	pf, ok := _pressure_registry[pressureType]
	if !ok {
		return nil, fmt.Errorf("pressure plugin not registered type:%v", pressureType)
	}
	return pf(), nil
}
