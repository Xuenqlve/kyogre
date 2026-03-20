package base_test

import (
	"context"
	"testing"

	"github.com/xuenqlve/common/schema_store"
	"github.com/xuenqlve/kyogre/internal/plugin/generator"
	"github.com/xuenqlve/kyogre/internal/plugin/iquery"
	"github.com/xuenqlve/kyogre/internal/plugin/metadata"
	base "github.com/xuenqlve/kyogre/pkg/scenario/base"
)

type stubBuilder struct {
	configured bool
}

func (s *stubBuilder) Configure(_ string, _ map[string]any) error {
	s.configured = true
	return nil
}

func (s *stubBuilder) LoadTargets(_ metadata.Metadata, _ []string) ([]base.Target, error) {
	return []base.Target{{Key: "db.users", Schema: struct{}{}}}, nil
}

func (s *stubBuilder) Build(_ base.Plan) (generator.GenerationContext, error) {
	return &stubContext{}, nil
}

type stubContext struct{}

func (s *stubContext) Kind() string                          { return "stub" }
func (s *stubContext) Provider(string) iquery.Provider       { return nil }
func (s *stubContext) Providers() map[string]iquery.Provider { return nil }
func (s *stubContext) Strategy() *generator.StrategySnapshot { return nil }
func (s *stubContext) Validate() error                       { return nil }
func (s *stubContext) Extras() map[string]any                { return nil }

func TestRegisterAndGetBuilder(t *testing.T) {
	builderType := base.BuilderType("test-builder-register")
	base.RegisterBuilder(builderType, &stubBuilder{}, false)

	builder, err := base.GetBuilder(builderType)
	if err != nil {
		t.Fatalf("get builder: %v", err)
	}
	if builder == nil {
		t.Fatalf("builder is nil")
	}

	targets, err := builder.LoadTargets(&stubMetadata{}, nil)
	if err != nil {
		t.Fatalf("load targets: %v", err)
	}
	if len(targets) != 1 || targets[0].Key != "db.users" {
		t.Fatalf("unexpected targets: %#v", targets)
	}
}

func TestGetBuilderReturnsFreshInstanceForNonSingleton(t *testing.T) {
	builderType := base.BuilderType("test-builder-factory")
	base.RegisterBuilder(builderType, &stubBuilder{}, false)

	builder1, err := base.GetBuilder(builderType)
	if err != nil {
		t.Fatalf("get builder1: %v", err)
	}
	builder2, err := base.GetBuilder(builderType)
	if err != nil {
		t.Fatalf("get builder2: %v", err)
	}
	if builder1 == builder2 {
		t.Fatalf("expected distinct builder instances for non-singleton registration")
	}
}

func TestGetBuilderReturnsErrorWhenMissing(t *testing.T) {
	if _, err := base.GetBuilder("missing-builder"); err == nil {
		t.Fatalf("expected missing builder error")
	}
}

type stubMetadata struct{}

func (s *stubMetadata) Configure(string, map[string]any) error { return nil }
func (s *stubMetadata) Initialize(context.Context) error       { return nil }
func (s *stubMetadata) SchemaKeys() []schema_store.SchemaKey   { return nil }
func (s *stubMetadata) SchemaPrimaryField(schema_store.SchemaKey) ([]iquery.BoundParam, error) {
	return nil, nil
}
func (s *stubMetadata) IQueryEnabled() bool                   { return false }
func (s *stubMetadata) SchemaStore() schema_store.SchemaStore { return nil }
func (s *stubMetadata) Close() error                          { return nil }
