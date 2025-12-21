package mock

import (
	"context"

	"github.com/mitchellh/mapstructure"
	"github.com/xuenqlve/common/errors"
	"github.com/xuenqlve/common/schema_store"
	"github.com/xuenqlve/kyogre/internal/plugin/metadata"
)

const Mock metadata.MetadataType = "mock"

type Config struct {
	Name string `mapstructure:"name"`
}

type Metadata struct {
	pipeline string
	cfg      Config
}

func init() {
	metadata.RegisterMetadata(Mock, &Metadata{}, false)
}

func (m *Metadata) Configure(pipeline string, data map[string]any) error {
	m.pipeline = pipeline
	if err := mapstructure.Decode(data, &m.cfg); err != nil {
		return errors.Trace(err)
	}
	if m.cfg.Name == "" {
		m.cfg.Name = "mock"
	}
	return nil
}

func (m *Metadata) Initialize(ctx context.Context) error {
	return nil
}

func (m *Metadata) SchemaKeys() []schema_store.SchemaKey {
	return nil
}

func (m *Metadata) SchemaStore() schema_store.SchemaStore {
	return nil
}

func (m *Metadata) Close() error {
	return nil
}
