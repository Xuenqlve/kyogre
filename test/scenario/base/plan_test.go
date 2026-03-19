package base_test

import (
	"testing"

	base "github.com/xuenqlve/kyogre/pkg/scenario/base"
)

func TestPlanValidate(t *testing.T) {
	plan := base.Plan{
		Builder:        "mysql",
		Mode:           base.ModeRow,
		Operation:      "insert",
		Target:         base.Target{Key: "db.users", Schema: struct{}{}},
		RowsPerMessage: 3,
	}
	if err := plan.Validate(); err != nil {
		t.Fatalf("validate plan: %v", err)
	}
}

func TestPlanValidateTransaction(t *testing.T) {
	plan := base.Plan{
		Builder:         "mysql",
		Mode:            base.ModeTransaction,
		Operation:       "insert",
		Target:          base.Target{Key: "db.orders", Schema: struct{}{}},
		RowsPerMessage:  1,
		TransactionSize: 2,
	}
	if err := plan.Validate(); err != nil {
		t.Fatalf("validate transaction plan: %v", err)
	}
}

func TestPlanValidateRejectsInvalidInput(t *testing.T) {
	tests := []base.Plan{
		{
			Mode:           base.ModeRow,
			Operation:      "insert",
			Target:         base.Target{Key: "db.users", Schema: struct{}{}},
			RowsPerMessage: 1,
		},
		{
			Builder:        "mysql",
			Mode:           "invalid",
			Operation:      "insert",
			Target:         base.Target{Key: "db.users", Schema: struct{}{}},
			RowsPerMessage: 1,
		},
		{
			Builder:        "mysql",
			Mode:           base.ModeRow,
			Target:         base.Target{Key: "db.users", Schema: struct{}{}},
			RowsPerMessage: 1,
		},
		{
			Builder:        "mysql",
			Mode:           base.ModeRow,
			Operation:      "insert",
			Target:         base.Target{Schema: struct{}{}},
			RowsPerMessage: 1,
		},
		{
			Builder:        "mysql",
			Mode:           base.ModeRow,
			Operation:      "insert",
			Target:         base.Target{Key: "db.users", Schema: struct{}{}},
			RowsPerMessage: 0,
		},
		{
			Builder:        "mysql",
			Mode:           base.ModeTransaction,
			Operation:      "insert",
			Target:         base.Target{Key: "db.orders", Schema: struct{}{}},
			RowsPerMessage: 1,
		},
	}
	for i := range tests {
		if err := tests[i].Validate(); err == nil {
			t.Fatalf("expected plan[%d] validate error", i)
		}
	}
}

func TestPlanClone(t *testing.T) {
	plan := base.Plan{
		Builder:         "mysql",
		Mode:            base.ModeTransaction,
		Operation:       "insert",
		Target:          base.Target{Key: "db.orders", Schema: struct{}{}, Extras: map[string]any{"kind": "table"}},
		RowsPerMessage:  2,
		TransactionSize: 3,
		Columns:         []string{"id", "name"},
		Extras:          map[string]any{"mode": "stress"},
	}

	cloned := plan.Clone()
	cloned.Columns[0] = "changed"
	cloned.Target.Extras["kind"] = "changed"
	cloned.Extras["mode"] = "changed"

	if plan.Columns[0] != "id" {
		t.Fatalf("expected original columns unchanged, got %q", plan.Columns[0])
	}
	if plan.Target.Extras["kind"] != "table" {
		t.Fatalf("expected original target extras unchanged, got %v", plan.Target.Extras["kind"])
	}
	if plan.Extras["mode"] != "stress" {
		t.Fatalf("expected original extras unchanged, got %v", plan.Extras["mode"])
	}
}
