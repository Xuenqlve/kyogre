package iquery

import (
	"context"
	"fmt"
	"reflect"
	"sync"

	"github.com/xuenqlve/common/schema_store"
)

type LookupRequest struct {
	Items []LookupRequestItem
}

type LookupRequestItem struct {
	Schema schema_store.SchemaKey
	Field  string
}

type LookupResult struct {
	Schema schema_store.SchemaKey
	Field  string
	Max    int64
	Rows   int64
	Extras map[string]any
}

type Lookup interface {
	Configure(pipeline string, cfg map[string]any) error
	Lookup(ctx context.Context, req LookupRequest) ([]LookupResult, error)
	Close() error
}

type (
	LookupType    string
	LookupFactory func() Lookup
)

var (
	_iquery_registry map[LookupType]LookupFactory
	_iquery_mutex    sync.Mutex
)

func RegisterIQueryPlugin(iQueryType LookupType, factory LookupFactory) {
	_iquery_mutex.Lock()
	defer _iquery_mutex.Unlock()
	if _iquery_registry == nil {
		_iquery_registry = make(map[LookupType]LookupFactory)
	}
	if _, ok := _iquery_registry[iQueryType]; ok {
		panic("pressure plugin already registered")
	}
	_iquery_registry[iQueryType] = factory
}

func RegisterIQuery(iQueryType LookupType, v Lookup, singleton bool) {
	var pf LookupFactory
	if singleton {
		pf = func() Lookup { return v }
	} else {
		pf = func() Lookup { return reflect.New(reflect.TypeOf(v).Elem()).Interface().(Lookup) }
	}
	RegisterIQueryPlugin(iQueryType, pf)
}

func GetIQueryModule(iQueryType LookupType) (Lookup, error) {
	_iquery_mutex.Lock()
	defer _iquery_mutex.Unlock()
	pf, ok := _iquery_registry[iQueryType]
	if !ok {
		return nil, fmt.Errorf("pressure plugin not registered type:%v", iQueryType)
	}
	return pf(), nil
}
