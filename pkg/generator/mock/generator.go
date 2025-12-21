package mock

import (
	"fmt"
	"sync/atomic"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/xuenqlve/common/errors"
	"github.com/xuenqlve/common/schema_store"
	"github.com/xuenqlve/kyogre/internal/message"
	"github.com/xuenqlve/kyogre/internal/plugin/generator"
	"github.com/xuenqlve/kyogre/internal/plugin/metadata"
)

const Mock generator.Type = "mock"

type Config struct {
	Prefix string `mapstructure:"prefix"`
}

type mockDependency struct {
	value string
}

type Generator struct {
	pipeline string
	cfg      Config
	seq      atomic.Int64
}

func init() {
	generator.RegisterGenerator(Mock, &Generator{}, false)
}

func (g *Generator) Configure(pipeline string, data map[string]any) error {
	g.pipeline = pipeline
	if err := mapstructure.Decode(data, &g.cfg); err != nil {
		return errors.Trace(err)
	}
	if g.cfg.Prefix == "" {
		g.cfg.Prefix = "mock"
	}
	return nil
}

func (g *Generator) RegisterMetadata(metadata.Metadata) {}

func (g *Generator) CollectDependencies(req *generator.DependencyRequest) (generator.GenerationDependency, error) {
	next := g.seq.Add(1)
	return &mockDependency{value: fmt.Sprintf("%s-%d", g.cfg.Prefix, next)}, nil
}

func (g *Generator) MockMessage(req *generator.MessageGenerationRequest) (message.Message, error) {
	dep, ok := req.Dependency.(*mockDependency)
	if !ok {
		return nil, fmt.Errorf("unexpected dependency type %T", req.Dependency)
	}
	return &message.MockMessage{Value: dep.value, CreatedAt: time.Now()}, nil
}

func (g *Generator) Close() {}

// implement interfaces for completeness
func (d *mockDependency) GetSchemas() []schema_store.SchemaKey { return nil }
func (d *mockDependency) GetMode() generator.GenerationMode    { return "mock" }
func (d *mockDependency) Validate() error                      { return nil }
func (d *mockDependency) DependencyType() string               { return "mock" }

// ensure schema_store imported
var _ generator.GenerationDependency = (*mockDependency)(nil)
