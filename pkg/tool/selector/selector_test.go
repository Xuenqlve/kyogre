package selector

import "testing"

type stubRand struct {
	values []int
	index  int
}

func (s *stubRand) Intn(n int) int {
	if len(s.values) == 0 {
		return 0
	}
	value := s.values[s.index%len(s.values)]
	s.index++
	if n <= 0 {
		return 0
	}
	if value < 0 {
		value = -value
	}
	return value % n
}

func TestRoundRobinSelector(t *testing.T) {
	sel, err := New[string](Config[string]{
		Strategy: RoundRobin,
		Items:    OptionsFromValues("users", "orders"),
	})
	if err != nil {
		t.Fatalf("new selector: %v", err)
	}

	got1, _ := sel.Pick()
	got2, _ := sel.Pick()
	got3, _ := sel.Pick()
	if got1 != "users" || got2 != "orders" || got3 != "users" {
		t.Fatalf("unexpected round-robin order: %q, %q, %q", got1, got2, got3)
	}
}

func TestRandomSelector(t *testing.T) {
	sel, err := New[string](Config[string]{
		Strategy: Random,
		Items:    OptionsFromValues("insert", "update", "delete"),
		Rand:     &stubRand{values: []int{2, 0, 1}},
	})
	if err != nil {
		t.Fatalf("new selector: %v", err)
	}

	got1, _ := sel.Pick()
	got2, _ := sel.Pick()
	got3, _ := sel.Pick()
	t.Logf("got1: %v, got2: %v, got3: %v", got1, got2, got3)
	if got1 != "delete" || got2 != "insert" || got3 != "update" {
		t.Fatalf("unexpected random picks: %q, %q, %q", got1, got2, got3)
	}
}

func TestWeightedSelector(t *testing.T) {
	sel, err := New[string](Config[string]{
		Strategy: Weighted,
		Items: []Option[string]{
			{Value: "insert", Weight: 3},
			{Value: "update", Weight: 1},
			{Value: "delete", Weight: 2},
		},
		Rand: &stubRand{values: []int{0, 3, 5}},
	})
	if err != nil {
		t.Fatalf("new selector: %v", err)
	}

	got1, _ := sel.Pick()
	got2, _ := sel.Pick()
	got3, _ := sel.Pick()
	if got1 != "insert" || got2 != "update" || got3 != "delete" {
		t.Fatalf("unexpected weighted picks: %q, %q, %q", got1, got2, got3)
	}
}

func TestWeightedSelectorRejectsInvalidWeight(t *testing.T) {
	_, err := New[string](Config[string]{
		Strategy: Weighted,
		Items: []Option[string]{
			{Value: "insert", Weight: 0},
		},
	})
	if err == nil {
		t.Fatalf("expected invalid weight error")
	}
}

func TestSelectorRejectsEmptyItems(t *testing.T) {
	_, err := New[string](Config[string]{Strategy: RoundRobin})
	if err == nil {
		t.Fatalf("expected empty items error")
	}
}
