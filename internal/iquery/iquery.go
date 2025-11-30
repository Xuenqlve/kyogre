package iquery

import (
	"context"
	"fmt"
	"reflect"
	"sync"

	"github.com/xuenqlve/common/schema_store"
)

type QueryResult struct {
	TableKey        schema_store.SchemaKey // 表的唯一标识符 "db.table"
	Field           string                 // 反查的字段
	MaxValue        int64                  // 字段的最大值
	CurrentRowCount int64                  // 表的当前行数
	Metadata        map[string]any         // 其他元数据
}

type IQuery interface {
	Configure(pipeline string, data map[string]any) (err error)
	QueryMaxValue(ctx context.Context, key schema_store.SchemaKey, field string) (int64, error)
	QueryRowCount(ctx context.Context, key schema_store.SchemaKey) (int64, error)
	BatchQuery(ctx context.Context, keys []schema_store.SchemaKey) ([]*QueryResult, error)
	GetQueryResult(ctx context.Context, key schema_store.SchemaKey, field string) (*QueryResult, error)
	Close() error
}

type (
	IQueryType    string
	IQueryFactory func() IQuery
)

var (
	_iquery_registry map[IQueryType]IQueryFactory
	_iquery_mutex    sync.Mutex
)

func RegisterIQueryPlugin(iQueryType IQueryType, factory IQueryFactory) {
	_iquery_mutex.Lock()
	defer _iquery_mutex.Unlock()
	if _iquery_registry == nil {
		_iquery_registry = make(map[IQueryType]IQueryFactory)
	}
	if _, ok := _iquery_registry[iQueryType]; ok {
		panic("pressure plugin already registered")
	}
	_iquery_registry[iQueryType] = factory
}

func RegisterIQuery(iQueryType IQueryType, v IQuery, singleton bool) {
	var pf IQueryFactory
	if singleton {
		pf = func() IQuery { return v }
	} else {
		pf = func() IQuery { return reflect.New(reflect.TypeOf(v).Elem()).Interface().(IQuery) }
	}
	RegisterIQueryPlugin(iQueryType, pf)
}

func GetIQueryModule(iQueryType IQueryType) (IQuery, error) {
	_iquery_mutex.Lock()
	defer _iquery_mutex.Unlock()
	pf, ok := _iquery_registry[iQueryType]
	if !ok {
		return nil, fmt.Errorf("pressure plugin not registered type:%v", iQueryType)
	}
	return pf(), nil
}
