package mysql

import (
	"context"
	"database/sql"

	"github.com/mitchellh/mapstructure"
	"github.com/xuenqlve/common/errors"
	"github.com/xuenqlve/common/log"
	mysql_schema "github.com/xuenqlve/common/relational_database/mysql"
	"github.com/xuenqlve/common/schema_store"
	"github.com/xuenqlve/kyogre/internal/plugin"
	"github.com/xuenqlve/kyogre/pkg/data_source/mysql"
)

var (
	MySQL             plugin.MetadataType = "mysql"
	ConfigMode        plugin.MetadataMode = "config"
	DefaultDataSource string              = "mock"
)

type Config struct {
	DataSource string   `mapstructure:"data-source" json:"data-source"`
	Databases  Database `mapstructure:"databases" json:"databases"`
}

func (c *Config) Validate() error {
	if c.DataSource == "" {
		c.DataSource = DefaultDataSource
	}
	return c.Databases.Validate()
}

func init() {
	plugin.RegisterMetadata(MySQL, ConfigMode, &Metadata{}, true)
}

type Metadata struct {
	pipeline       string
	cfg            *Config
	conn           *sql.DB
	schema         schema_store.SchemaStore
	keys           []schema_store.SchemaKey
	mockDatasource bool
}

func (m *Metadata) Configure(pipeline string, data map[string]any) (err error) {
	m.pipeline = pipeline
	if err = mapstructure.Decode(data, &m.cfg); err != nil {
		return errors.Trace(err)
	}
	if err = m.cfg.Validate(); err != nil {
		return errors.Trace(err)
	}
	m.keys = m.cfg.Databases.SchemaKeys()
	return nil
}

func (m *Metadata) Initialize(ctx context.Context) error {
	if m.cfg.DataSource != DefaultDataSource {
		return m.initializeByDb(ctx)
	}
	m.initializeByMock()
	return nil
}

func (m *Metadata) initializeByMock() {
	// 构建mock tables
	mockTables := BuildMockTables(m.cfg.Databases)
	// 创建mock加载工具
	loader := &MockLoadSchemaTool{
		mockTables: mockTables,
	}
	// 使用mock loader创建SchemaStore
	m.schema = schema_store.NewBaseSchemaStore(loader)
}

func (m *Metadata) initializeByDb(ctx context.Context) error {
	conn, err := mysql.Connection(m.cfg.DataSource)
	m.schema = schema_store.NewBaseSchemaStore(mysql_schema.NewSchema(conn))
	if err != nil {
		return errors.Trace(err)
	}
	sqlSets := CreateTableSQLs(m.cfg.Databases)
	for _, query := range sqlSets {
		log.Infof("Initialize SQL: %s", query)
		_, err = m.conn.ExecContext(ctx, query)
		if err != nil {
			return errors.Trace(err)
		}
	}
	return nil
}

func (m *Metadata) SchemaKeys() []schema_store.SchemaKey {
	return m.keys
}

func (m *Metadata) SchemaStore() schema_store.SchemaStore {
	return m.schema
}

func (m *Metadata) Close() error {
	if m.conn != nil {
		if err := m.conn.Close(); err != nil {
			return errors.Trace(err)
		}
	}
	return nil
}
