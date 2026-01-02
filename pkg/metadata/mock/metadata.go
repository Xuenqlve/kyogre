package mock

import (
	"context"
	"fmt"

	"github.com/mitchellh/mapstructure"
	"github.com/xuenqlve/common/errors"
	"github.com/xuenqlve/common/schema_store"
	"github.com/xuenqlve/kyogre/internal/plugin/iquery"
	"github.com/xuenqlve/kyogre/internal/plugin/metadata"
)

const Mock metadata.MetadataType = "mock"

type Config struct {
	Name string `mapstructure:"name"`
	// IQueryEnabled 控制该 metadata 是否参与 lookup（默认 true）。
	IQueryEnabled bool `mapstructure:"lookup-enabled" json:"lookup-enabled"`
}

type Metadata struct {
	pipeline string
	cfg      Config
	schema   schema_store.SchemaStore
	keys     []schema_store.SchemaKey
}

func init() {
	metadata.RegisterMetadata(Mock, &Metadata{}, false)
}

func (m *Metadata) Configure(pipeline string, data map[string]any) error {
	m.pipeline = pipeline
	m.cfg = Config{IQueryEnabled: true}
	if err := mapstructure.Decode(data, &m.cfg); err != nil {
		return errors.Trace(err)
	}
	if m.cfg.Name == "" {
		m.cfg.Name = "mock"
	}
	m.keys = []schema_store.SchemaKey{mockSchemaKey{name: fmt.Sprintf("%s_table", m.cfg.Name)}}
	loader := &mockSchemaLoader{
		schemas: map[string]any{
			m.keys[0].UniqueID(): map[string]any{
				"pipeline": pipeline,
				"name":     m.cfg.Name,
			},
		},
	}
	m.schema = schema_store.NewBaseSchemaStore(loader)
	return nil
}

func (m *Metadata) Initialize(ctx context.Context) error {
	return nil
}

func (m *Metadata) SchemaKeys() []schema_store.SchemaKey {
	return m.keys
}

func (m *Metadata) SchemaPrimaryField(key schema_store.SchemaKey) ([]iquery.BoundParam, error) {
	// mock metadata 默认使用 id 作为主键字段
	return []iquery.BoundParam{{Column: "id", Type: "int"}}, nil
}

func (m *Metadata) IQueryEnabled() bool {
	return m.cfg.IQueryEnabled
}

func (m *Metadata) SchemaStore() schema_store.SchemaStore {
	return m.schema
}

func (m *Metadata) Close() error {
	if m.schema != nil {
		return m.schema.Close()
	}
	return nil
}

type mockSchemaKey struct {
	name string
}

func (k mockSchemaKey) UniqueID() string {
	return k.name
}

type mockSchemaLoader struct {
	schemas map[string]any
}

func (l *mockSchemaLoader) LoadSchema(key schema_store.SchemaKey) (any, error) {
	if key == nil {
		return nil, fmt.Errorf("schema key is nil")
	}
	if v, ok := l.schemas[key.UniqueID()]; ok {
		return v, nil
	}
	return map[string]any{"name": key.UniqueID()}, nil
}

func (l *mockSchemaLoader) Close() error { return nil }
