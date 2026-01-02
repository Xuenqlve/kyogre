package metadata

import (
	"context"
	"fmt"
	"reflect"
	"sync"

	"github.com/xuenqlve/common/schema_store"
	"github.com/xuenqlve/kyogre/internal/plugin/iquery"
)

// Metadata 元数据接口
// 职责：初始化压测环境，建立表结构，提供基础元信息
type Metadata interface {
	// Configure 配置元数据，初始化表结构、字段定义等
	Configure(pipeline string, data map[string]any) error

	// Initialize 执行初始化操作，在真实数据库中创建表、初始化种子数据等
	Initialize(ctx context.Context) error

	SchemaKeys() []schema_store.SchemaKey

	SchemaPrimaryField(key schema_store.SchemaKey) ([]iquery.BoundParam, error)

	// IQueryEnabled 表示该 metadata 是否启用 lookup（默认应为 true，可在 metadata config 中关闭）。
	IQueryEnabled() bool

	SchemaStore() schema_store.SchemaStore

	// Close 清理资源
	Close() error
}

// MetadataType 元数据类型
type MetadataType string

//type MetadataMode string

// MetadataFactory 元数据工厂函数
type MetadataFactory func() Metadata

var (
	//_metadata_registry map[MetadataType]map[MetadataMode]MetadataFactory
	_metadata_registry map[MetadataType]MetadataFactory
	_metadata_mutex    sync.Mutex
)

// RegisterMetadataPlugin 注册元数据插件
func RegisterMetadataPlugin(metadataType MetadataType, factory MetadataFactory) {
	_metadata_mutex.Lock()
	defer _metadata_mutex.Unlock()
	if _metadata_registry == nil {
		//_metadata_registry = make(map[MetadataType]map[MetadataMode]MetadataFactory)
		_metadata_registry = make(map[MetadataType]MetadataFactory)
	}
	_, ok := _metadata_registry[metadataType]
	if ok {
		panic(fmt.Sprintf("metadata plugin already registered with type %s", metadataType))
	}
	_metadata_registry[metadataType] = factory
}

// RegisterMetadata 注册元数据实现
// singleton 为 true 时返回同一个实例，为 false 时每次返回新实例
func RegisterMetadata(metadataType MetadataType, v Metadata, singleton bool) {
	var mf MetadataFactory
	if singleton {
		mf = func() Metadata { return v }
	} else {
		mf = func() Metadata { return reflect.New(reflect.TypeOf(v).Elem()).Interface().(Metadata) }
	}
	RegisterMetadataPlugin(metadataType, mf)
}

// GetMetadata 根据类型获取元数据实例
func GetMetadata(metadataType MetadataType) (Metadata, error) {
	_metadata_mutex.Lock()
	defer _metadata_mutex.Unlock()
	plugin, ok := _metadata_registry[metadataType]
	if !ok {
		return nil, fmt.Errorf("metadata plugin not registered type:%v", metadataType)
	}
	return plugin(), nil
}
