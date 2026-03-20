package base_test

import (
	"context"
	"testing"

	"github.com/xuenqlve/kyogre/internal/plugin/generator"
	pluginMetadata "github.com/xuenqlve/kyogre/internal/plugin/metadata"
	base "github.com/xuenqlve/kyogre/pkg/scenario/base"
)

type recordingBuilder struct {
	targets []base.Target
	plans   []base.Plan
}

func (b *recordingBuilder) Configure(_ string, _ map[string]any) error { return nil }

func (b *recordingBuilder) LoadTargets(_ pluginMetadata.Metadata, _ []string) ([]base.Target, error) {
	out := make([]base.Target, 0, len(b.targets))
	for _, target := range b.targets {
		out = append(out, target.Clone())
	}
	return out, nil
}

func (b *recordingBuilder) Build(plan base.Plan) (generator.GenerationContext, error) {
	b.plans = append(b.plans, plan.Clone())
	return &stubContext{}, nil
}

func (b *recordingBuilder) Close() error { return nil }

func TestBaseScenarioStartRow(t *testing.T) {
	builderType := base.BuilderType("test-base-scenario-row")
	builder := &recordingBuilder{
		targets: []base.Target{
			{Key: "db.users", Schema: struct{}{}},
			{Key: "db.orders", Schema: struct{}{}},
		},
	}
	base.RegisterBuilder(builderType, builder, true)

	sc := &base.Scenario{}
	err := sc.Configure("pipeline-row", map[string]any{
		"builder":       string(builderType),
		"message-count": 2,
		"mode":          base.ModeRow,
		"operation-selector": map[string]any{
			"values": []string{"insert"},
		},
		"row-count-selector": map[string]any{
			"fixed": 3,
		},
	})
	if err != nil {
		t.Fatalf("configure scenario: %v", err)
	}

	ctxChan := make(chan generator.GenerationContext, 2)
	sc.Start(context.Background(), nil, nil, ctxChan)

	if len(builder.plans) != 2 {
		t.Fatalf("expected 2 plans, got %d", len(builder.plans))
	}
	if builder.plans[0].RowsPerMessage != 3 {
		t.Fatalf("unexpected rows per message: %d", builder.plans[0].RowsPerMessage)
	}
	if builder.plans[0].Target.Key != "db.users" || builder.plans[1].Target.Key != "db.orders" {
		t.Fatalf("unexpected target order: %s -> %s", builder.plans[0].Target.Key, builder.plans[1].Target.Key)
	}
	if len(ctxChan) != 2 {
		t.Fatalf("expected 2 contexts, got %d", len(ctxChan))
	}
	summary := sc.Summary()
	if summary["messages"] != 2 {
		t.Fatalf("unexpected summary messages: %v", summary["messages"])
	}
}

func TestBaseScenarioStartTransaction(t *testing.T) {
	builderType := base.BuilderType("test-base-scenario-transaction")
	builder := &recordingBuilder{
		targets: []base.Target{
			{Key: "db.orders", Schema: struct{}{}},
		},
	}
	base.RegisterBuilder(builderType, builder, true)

	sc := &base.Scenario{}
	err := sc.Configure("pipeline-tx", map[string]any{
		"builder":       string(builderType),
		"message-count": 1,
		"mode":          base.ModeTransaction,
		"operation-selector": map[string]any{
			"values": []string{"insert"},
		},
		"row-count-selector": map[string]any{
			"fixed": 2,
		},
		"transaction-size-selector": map[string]any{
			"fixed": 4,
		},
	})
	if err != nil {
		t.Fatalf("configure scenario: %v", err)
	}

	ctxChan := make(chan generator.GenerationContext, 1)
	sc.Start(context.Background(), nil, nil, ctxChan)

	if len(builder.plans) != 1 {
		t.Fatalf("expected 1 plan, got %d", len(builder.plans))
	}
	if builder.plans[0].TransactionSize != 4 {
		t.Fatalf("unexpected transaction size: %d", builder.plans[0].TransactionSize)
	}
	if len(ctxChan) != 1 {
		t.Fatalf("expected 1 context, got %d", len(ctxChan))
	}
}

func TestBaseScenarioStartWithNoTargets(t *testing.T) {
	builderType := base.BuilderType("test-base-scenario-empty")
	builder := &recordingBuilder{}
	base.RegisterBuilder(builderType, builder, true)

	sc := &base.Scenario{}
	err := sc.Configure("pipeline-empty", map[string]any{
		"builder": string(builderType),
	})
	if err != nil {
		t.Fatalf("configure scenario: %v", err)
	}

	ctxChan := make(chan generator.GenerationContext, 1)
	sc.Start(context.Background(), nil, nil, ctxChan)

	if len(builder.plans) != 0 {
		t.Fatalf("expected no plans, got %d", len(builder.plans))
	}
	if len(ctxChan) != 0 {
		t.Fatalf("expected no contexts, got %d", len(ctxChan))
	}
}
