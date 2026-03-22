package base_test

import (
	"testing"

	base "github.com/xuenqlve/kyogre/pkg/scenario/base"
)

type stubRand struct {
	values []int
	index  int
}

func (s *stubRand) Intn(n int) int {
	if len(s.values) == 0 || n <= 0 {
		return 0
	}
	value := s.values[s.index%len(s.values)]
	s.index++
	if value < 0 {
		value = -value
	}
	return value % n
}

func TestSelectorFactoryTarget(t *testing.T) {
	factory := base.NewSelectorFactory(&stubRand{values: []int{1}})
	sel, err := factory.Target(base.TargetSelectorConfig{
		Strategy: "random",
		Schemas:  []string{"db.users", "db.orders"},
	}, []base.Target{
		{Key: "db.users", Schema: struct{}{}},
		{Key: "db.orders", Schema: struct{}{}},
		{Key: "db.logs", Schema: struct{}{}},
	})
	if err != nil {
		t.Fatalf("new target selector: %v", err)
	}
	target, err := sel.Pick()
	if err != nil {
		t.Fatalf("pick target: %v", err)
	}
	if target.Key != "db.orders" {
		t.Fatalf("unexpected target key: %s", target.Key)
	}
}

func TestSelectorFactoryWeightedTarget(t *testing.T) {
	factory := base.NewSelectorFactory(&stubRand{values: []int{3}})
	sel, err := factory.Target(base.TargetSelectorConfig{
		Strategy: "weighted",
		Weights: map[string]int{
			"db.users":  3,
			"db.orders": 1,
		},
	}, []base.Target{
		{Key: "db.users", Schema: struct{}{}},
		{Key: "db.orders", Schema: struct{}{}},
	})
	if err != nil {
		t.Fatalf("new weighted target selector: %v", err)
	}
	target, err := sel.Pick()
	if err != nil {
		t.Fatalf("pick target: %v", err)
	}
	if target.Key != "db.orders" {
		t.Fatalf("unexpected weighted target key: %s", target.Key)
	}
}

func TestSelectorFactoryOperation(t *testing.T) {
	factory := base.NewSelectorFactory(&stubRand{values: []int{2}})
	sel, err := factory.Operation(base.ValueSelectorConfig{
		Strategy: "random",
		Values:   []string{"insert", "update", "delete"},
	})
	if err != nil {
		t.Fatalf("new operation selector: %v", err)
	}
	value, err := sel.Pick()
	if err != nil {
		t.Fatalf("pick operation: %v", err)
	}
	if value != "delete" {
		t.Fatalf("unexpected operation: %s", value)
	}
}

func TestSelectorFactoryInt(t *testing.T) {
	factory := base.NewSelectorFactory(&stubRand{values: []int{2}})
	sel, err := factory.Int(base.IntSelectorConfig{
		Strategy: "random",
		Values:   []int{1, 3, 5},
	})
	if err != nil {
		t.Fatalf("new int selector: %v", err)
	}
	value, err := sel.Pick()
	if err != nil {
		t.Fatalf("pick int: %v", err)
	}
	if value != 5 {
		t.Fatalf("unexpected int selector value: %d", value)
	}
}

func TestSelectorFactoryRejectsEmptyTargets(t *testing.T) {
	factory := base.NewSelectorFactory(nil)
	_, err := factory.Target(base.TargetSelectorConfig{Strategy: "round-robin"}, nil)
	if err == nil {
		t.Fatalf("expected empty targets error")
	}
}
