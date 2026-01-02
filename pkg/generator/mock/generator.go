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

// MockDependencyConfig 允许外部（如 Scenario）直接指定一次性值，用于对接 lookup 生成有序数据
type MockDependencyConfig struct {
	ForceValue string `mapstructure:"force-value" json:"force-value"`
}

type mockDependency struct {
	value string
}

type Generator struct {
	pipeline string
	cfg      Config
	seq      atomic.Int64
	md       metadata.Metadata
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

func (g *Generator) RegisterMetadata(md metadata.Metadata) {
	g.md = md
}

func (g *Generator) CollectDependencies(req *generator.DependencyRequest) (generator.GenerationDependency, error) {
	if cfg, ok := req.Config.(*MockDependencyConfig); ok && cfg.ForceValue != "" {
		return &mockDependency{value: cfg.ForceValue}, nil
	}
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

// ensure interface implementation for custom dependency config
func (c *MockDependencyConfig) Validate() error { return nil }
func (c *MockDependencyConfig) Type() string    { return "mock" }

// ensure schema_store imported
var _ generator.GenerationDependency = (*mockDependency)(nil)
var _ generator.DependencyConfig = (*MockDependencyConfig)(nil)
