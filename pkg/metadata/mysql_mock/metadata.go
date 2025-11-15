package mysql_mock

import (
	"context"

	"github.com/mitchellh/mapstructure"
	"github.com/xuenqlve/common/errors"
	mysql_schema "github.com/xuenqlve/common/relational_database/mysql"
	"github.com/xuenqlve/common/schema_store"
	"github.com/xuenqlve/kyogre/internal/plugin"
	"github.com/xuenqlve/kyogre/pkg/metadata/common"
)

var (
	MySQL    plugin.MetadataType = "mysql"
	MockMode plugin.MetadataMode = "mock"
)

func init() {
	plugin.RegisterMetadata(MySQL, MockMode, &Metadata{}, true)
}

// MockLoadSchemaTool 是一个不依赖数据库连接的schema加载工具
// 从内存中的mock数据返回Table对象
type MockLoadSchemaTool struct {
	mockTables map[string]*mysql_schema.Table
}

// LoadSchema 根据SchemaKey从mock数据中返回Table
func (m *MockLoadSchemaTool) LoadSchema(key schema_store.SchemaKey) (any, error) {
	// key 应该是 mysql_schema.Index 类型
	idx, ok := key.(*mysql_schema.Index)
	if !ok {
		return nil, errors.Errorf("invalid schema key type, expected *mysql_schema.Index, got %T", key)
	}

	tableKey := idx.Database + "." + idx.Table
	table, exists := m.mockTables[tableKey]
	if !exists {
		return nil, errors.Errorf("mock table not found: %s", tableKey)
	}

	return table, nil
}

// Close 关闭资源（mock版本不需要实际操作）
func (m *MockLoadSchemaTool) Close() error {
	return nil
}

type Config struct {
	Databases common.Database `mapstructure:"databases" json:"databases"`
}

// Metadata 是mysql mock模式的元数据实现
// 不通过查询实际MySQL数据库来获取schema信息
type Metadata struct {
	pipeline string
	cfg      *Config
	schema   schema_store.SchemaStore
	keys     []schema_store.SchemaKey
}

// Configure 配置元数据
// 解析配置并构建mock的Table对象
func (m *Metadata) Configure(pipeline string, data map[string]any) error {
	m.pipeline = pipeline

	// 解析配置到Config结构体
	m.cfg = &Config{}
	if err := mapstructure.Decode(data, m.cfg); err != nil {
		return errors.Trace(err)
	}

	// 构建mock tables
	mockTables := common.BuildMockTables(m.cfg.Databases)

	// 创建mock加载工具
	loader := &MockLoadSchemaTool{
		mockTables: mockTables,
	}

	// 使用mock loader创建SchemaStore
	m.schema = schema_store.NewBaseSchemaStore(loader)

	// 从配置中获取schema keys
	m.keys = m.cfg.Databases.SchemaKeys()
	return nil
}

// Initialize 初始化元数据
// mock版本不需要创建表，只做简单的验证
func (m *Metadata) Initialize(ctx context.Context) error {
	if len(m.keys) == 0 {
		return errors.New("no tables configured in mock metadata")
	}
	return nil
}

// SchemaKeys 返回所有配置的表的schema keys
func (m *Metadata) SchemaKeys() []schema_store.SchemaKey {
	return m.keys
}

// SchemaStore 返回schema存储对象
func (m *Metadata) SchemaStore() schema_store.SchemaStore {
	return m.schema
}

// Close 关闭资源
func (m *Metadata) Close() error {
	if m.schema != nil {
		if err := m.schema.Close(); err != nil {
			return errors.Trace(err)
		}
	}
	return nil
}
