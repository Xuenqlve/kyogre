package data_source

import (
	"fmt"
	"reflect"
	"sync"

	"github.com/xuenqlve/common/errors"
)

type (
	DataSourceFactory func() DataSource
	DataSourceType    string
)

func (ds DataSourceType) String() string {
	return string(ds)
}

type DataSource interface {
	Configure(pipeline string, data map[string]any) error
	CreateDataSource(dataSourceName string) (any, error)
	DataSourceConfig(dataSourceName string) (any, error)
}

var (
	_data_source_registry map[DataSourceType]DataSourceFactory
	_data_source_mutex    sync.Mutex
)

func RegisterPluginFactory(dataSourceType DataSourceType, factory DataSourceFactory) {
	_data_source_mutex.Lock()
	defer _data_source_mutex.Unlock()

	if _data_source_registry == nil {
		_data_source_registry = make(map[DataSourceType]DataSourceFactory)
	}

	_, ok := _data_source_registry[dataSourceType]
	if ok {
		panic(fmt.Sprintf("plugin already exists, type: %v", dataSourceType))
	}

	_data_source_registry[dataSourceType] = factory
}

func RegisterPlugin(dataSourceType DataSourceType, v DataSource, singleton bool) {
	var pf DataSourceFactory
	if singleton {
		pf = func() DataSource {
			return v
		}
	} else {
		pf = func() DataSource {
			return reflect.New(reflect.TypeOf(v).Elem()).Interface().(DataSource)
		}
	}
	RegisterPluginFactory(dataSourceType, pf)
}

func GetDataSource(dataSourceType DataSourceType) (DataSource, error) {
	_data_source_mutex.Lock()
	defer _data_source_mutex.Unlock()

	if _data_source_registry == nil {
		return nil, errors.Errorf("data_source empty registry")
	}

	p, ok := _data_source_registry[dataSourceType]
	if !ok {
		return nil, errors.Errorf("data_source empty plugin name: %v", dataSourceType)
	}
	return p(), nil
}
