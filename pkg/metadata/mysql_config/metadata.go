package mysql_config

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
	"github.com/xuenqlve/kyogre/pkg/metadata/common"
)

var (
	MySQL      plugin.MetadataType = "mysql"
	ConfigMode plugin.MetadataMode = "config"
)

type Config struct {
	DataSource string          `mapstructure:"data-source" json:"data-source"`
	Databases  common.Database `mapstructure:"databases" json:"databases"`
}

func init() {
	plugin.RegisterMetadata(MySQL, ConfigMode, &Metadata{}, true)
}

type Metadata struct {
	pipeline string
	cfg      *Config
	conn     *sql.DB
	schema   schema_store.SchemaStore
	keys     []schema_store.SchemaKey
}

func (m *Metadata) Configure(pipeline string, data map[string]any) (err error) {
	m.pipeline = pipeline
	if err = mapstructure.Decode(data, &m.cfg); err != nil {
		return errors.Trace(err)
	}
	m.conn, err = mysql.Connection(m.cfg.DataSource)
	if err != nil {
		return errors.Trace(err)
	}
	m.schema = schema_store.NewBaseSchemaStore(mysql_schema.NewSchema(m.conn))
	m.keys = m.cfg.Databases.SchemaKeys()
	return nil
}

func (m *Metadata) Initialize(ctx context.Context) error {
	sqlSets := common.CreateTableSQLs(m.cfg.Databases)
	for _, query := range sqlSets {
		log.Infof("Initialize SQL: %s", query)
		_, err := m.conn.ExecContext(ctx, query)
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
