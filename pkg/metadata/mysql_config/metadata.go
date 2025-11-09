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
)

var (
	MySQL      plugin.MetadataType = "mysql"
	ConfigMode plugin.MetadataMode = "config"
)

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
	m.keys = m.cfg.SchemaKeys()
	return nil
}

func (m *Metadata) Initialize(ctx context.Context) error {
	sqlSets := m.cfg.CreateTableSQLs()
	for _, sql := range sqlSets {
		log.Infof("Initialize SQL: %s", sql)
		_, err := m.conn.ExecContext(ctx, sql)
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
