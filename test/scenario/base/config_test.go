package base_test

import (
	"testing"

	base "github.com/xuenqlve/kyogre/pkg/scenario/base"
)

func TestConfigNormalizeDefaults(t *testing.T) {
	cfg := base.Config{
		Builder: "mysql",
	}
	if err := cfg.Normalize(); err != nil {
		t.Fatalf("normalize config: %v", err)
	}
	if cfg.Mode != base.ModeRow {
		t.Fatalf("unexpected mode: %s", cfg.Mode)
	}
	if cfg.MessageCount != 1 {
		t.Fatalf("unexpected message count: %d", cfg.MessageCount)
	}
	if cfg.TargetSelector.Strategy == "" {
		t.Fatalf("expected target selector strategy defaulted")
	}
	if len(cfg.OperationSelector.Values) != 1 || cfg.OperationSelector.Values[0] != "insert" {
		t.Fatalf("unexpected operation selector defaults: %#v", cfg.OperationSelector.Values)
	}
	if len(cfg.RowCountSelector.Values) != 1 || cfg.RowCountSelector.Values[0] != 1 {
		t.Fatalf("unexpected row count selector defaults: %#v", cfg.RowCountSelector.Values)
	}
}

func TestConfigNormalizeTransactionDefaults(t *testing.T) {
	cfg := base.Config{
		Builder: "mysql",
		Mode:    base.ModeTransaction,
	}
	if err := cfg.Normalize(); err != nil {
		t.Fatalf("normalize transaction config: %v", err)
	}
	if len(cfg.TransactionSizeSelector.Values) != 1 || cfg.TransactionSizeSelector.Values[0] != 2 {
		t.Fatalf("unexpected transaction selector defaults: %#v", cfg.TransactionSizeSelector.Values)
	}
}

func TestConfigNormalizeRangeValues(t *testing.T) {
	cfg := base.Config{
		Builder: "mysql",
		RowCountSelector: base.IntSelectorConfig{
			Min: 1,
			Max: 3,
		},
	}
	if err := cfg.Normalize(); err != nil {
		t.Fatalf("normalize range config: %v", err)
	}
	if len(cfg.RowCountSelector.Values) != 3 {
		t.Fatalf("unexpected range values: %#v", cfg.RowCountSelector.Values)
	}
}

func TestConfigNormalizeRejectsInvalidInput(t *testing.T) {
	tests := []base.Config{
		{},
		{
			Builder: "mysql",
			Mode:    "invalid",
		},
		{
			Builder: "mysql",
			OperationSelector: base.ValueSelectorConfig{
				Strategy: "weighted",
			},
		},
		{
			Builder: "mysql",
			RowCountSelector: base.IntSelectorConfig{
				Strategy: "weighted",
			},
		},
		{
			Builder: "mysql",
			RowCountSelector: base.IntSelectorConfig{
				Min: 3,
				Max: 1,
			},
		},
		{
			Builder: "mysql",
			Lookup: base.LookupConfig{
				Enabled:    true,
				Operations: []string{""},
			},
		},
	}
	for i := range tests {
		if err := tests[i].Normalize(); err == nil {
			t.Fatalf("expected normalize error for test[%d]", i)
		}
	}
}
