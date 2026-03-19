package base

import "testing"

func TestConfigNormalizeDefaults(t *testing.T) {
	cfg := Config{
		Builder: "mysql",
	}
	if err := cfg.Normalize(); err != nil {
		t.Fatalf("normalize config: %v", err)
	}
	if cfg.Mode != ModeRow {
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
	cfg := Config{
		Builder: "mysql",
		Mode:    ModeTransaction,
	}
	if err := cfg.Normalize(); err != nil {
		t.Fatalf("normalize transaction config: %v", err)
	}
	if len(cfg.TransactionSizeSelector.Values) != 1 || cfg.TransactionSizeSelector.Values[0] != 2 {
		t.Fatalf("unexpected transaction selector defaults: %#v", cfg.TransactionSizeSelector.Values)
	}
}

func TestConfigNormalizeRangeValues(t *testing.T) {
	cfg := Config{
		Builder: "mysql",
		RowCountSelector: IntSelectorConfig{
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
	tests := []Config{
		{},
		{
			Builder: "mysql",
			Mode:    "invalid",
		},
		{
			Builder: "mysql",
			OperationSelector: ValueSelectorConfig{
				Strategy: "weighted",
			},
		},
		{
			Builder: "mysql",
			RowCountSelector: IntSelectorConfig{
				Strategy: "weighted",
			},
		},
		{
			Builder: "mysql",
			RowCountSelector: IntSelectorConfig{
				Min: 3,
				Max: 1,
			},
		},
		{
			Builder: "mysql",
			Lookup: LookupConfig{
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
