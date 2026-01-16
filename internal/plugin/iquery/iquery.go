package iquery

import (
	"context"
	"fmt"
	"reflect"
	"sync"

	"github.com/xuenqlve/common/schema_store"
	"github.com/xuenqlve/kyogre/pkg/tool/range_pool"
)

// LookupRequest 描述一次反查任务需要查询的表及字段及分配窗口需求。
//type LookupRequest struct {
//	Items []LookupRequestItem
//}

type LookupRequest struct {
	Schema    schema_store.SchemaKey
	Params    []BoundParam
	Partition string
	Need      int64
	WrapAt    int64
}

// LookupResult 承载反查模块返回的分配窗口信息
type LookupResult struct {
	EnableLoop bool
	Window     range_pool.IntRange
}

type ValuesRequest struct {
	Schema  schema_store.SchemaKey
	Columns []BoundParam
	Cursor  any
	Limit   int
}

type ValuesResult struct {
	Rows       [][]any
	NextCursor any
	HasMore    bool
}

type BoundParam struct {
	Column string
	Type   string
}

type Bound struct {
	BoundParam
	MinValue any
	MaxValue any
	Count    int
}

type Lookup interface {
	// Configure 根据配置初始化反查插件
	Configure(pipeline string, cfg map[string]any) error
	// LookupBounds 返回分配窗口信息，用于 range_pool 的 refill 回调。
	LookupBounds(ctx context.Context, req LookupRequest) (LookupResult, error)
	// ScanValues 按游标分页扫描值集合，用于 AliveSet 等“非 int/联合唯一”场景。
	ScanValues(ctx context.Context, req ValuesRequest) (ValuesResult, error)
	// Close 释放资源
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

// RegisterIQueryPlugin 注册反查工厂方法
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

// RegisterIQuery 注册反查插件，支持单例/多例
func RegisterIQuery(iQueryType LookupType, v Lookup, singleton bool) {
	var pf LookupFactory
	if singleton {
		pf = func() Lookup { return v }
	} else {
		pf = func() Lookup { return reflect.New(reflect.TypeOf(v).Elem()).Interface().(Lookup) }
	}
	RegisterIQueryPlugin(iQueryType, pf)
}

// GetIQueryModule 根据类型创建反查实例
func GetIQueryModule(iQueryType LookupType) (Lookup, error) {
	_iquery_mutex.Lock()
	defer _iquery_mutex.Unlock()
	pf, ok := _iquery_registry[iQueryType]
	if !ok {
		return nil, fmt.Errorf("pressure plugin not registered type:%v", iQueryType)
	}
	return pf(), nil
}
