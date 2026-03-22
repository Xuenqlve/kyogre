package base_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/xuenqlve/kyogre/internal/message"
	pluginScenario "github.com/xuenqlve/kyogre/internal/plugin/scenario"
	base "github.com/xuenqlve/kyogre/pkg/scenario/base"
)

func TestDirectorCapturesBaseScenarioRuntimeError(t *testing.T) {
	builderType := base.BuilderType("test-base-scenario-director-runtime")
	builder := &recordingBuilder{
		targets:  []base.Target{{Key: "db.users", Schema: struct{}{}}},
		buildErr: errors.New("build failed"),
	}
	base.RegisterBuilder(builderType, builder, true)

	director := pluginScenario.NewDirector()
	if err := director.Configure("pipeline-runtime", string(base.ScenarioType), map[string]any{
		"builder":       string(builderType),
		"message-count": 1,
	}); err != nil {
		t.Fatalf("configure director: %v", err)
	}
	director.RegisterMetadata(&stubMetadata{})
	director.EnableFinite()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	point := make(message.Point, 1)
	if err := director.Start(ctx, point.InPoint()); err != nil {
		t.Fatalf("start director: %v", err)
	}

	select {
	case <-director.Done():
	case <-time.After(2 * time.Second):
		t.Fatalf("director did not finish in time")
	}

	if err := director.Err(); err == nil || err.Error() != "build failed" {
		t.Fatalf("unexpected director runtime error: %v", err)
	}

	summary := director.Summary()
	if summary["error"] != "build failed" {
		t.Fatalf("unexpected summary error: %v", summary["error"])
	}
}
