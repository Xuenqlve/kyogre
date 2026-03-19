package base_test

import (
	"context"
	"testing"

	mysql_schema "github.com/xuenqlve/common/relational_database/mysql"
	"github.com/xuenqlve/kyogre/internal/plugin/iquery"
	"github.com/xuenqlve/kyogre/pkg/message"
	base "github.com/xuenqlve/kyogre/pkg/scenario/base"
)

type stubProvider struct{}

func (s *stubProvider) Columns() []string      { return []string{"id"} }
func (s *stubProvider) Rows() []map[string]any { return []map[string]any{{"id": 1}} }

type stubSequencer struct {
	lastMethod string
	lastNeed   int64
	provider   iquery.Provider
	err        error
}

func (s *stubSequencer) ReserveInsert(_ context.Context, _ iquery.SequenceSpec, need int64) (iquery.Provider, error) {
	s.lastMethod = "insert"
	s.lastNeed = need
	return s.provider, s.err
}

func (s *stubSequencer) ReserveUpdate(_ context.Context, _ iquery.SequenceSpec, need int64) (iquery.Provider, error) {
	s.lastMethod = "update"
	s.lastNeed = need
	return s.provider, s.err
}

func (s *stubSequencer) ReserveDelete(_ context.Context, _ iquery.SequenceSpec, need int64) (iquery.Provider, error) {
	s.lastMethod = "delete"
	s.lastNeed = need
	return s.provider, s.err
}

func TestLookupBinderProvidersDisabled(t *testing.T) {
	binder := base.NewLookupBinder(base.LookupConfig{}, nil)
	providers, err := binder.Providers(context.Background(), base.Plan{Operation: message.Update})
	if err != nil {
		t.Fatalf("providers: %v", err)
	}
	if providers != nil {
		t.Fatalf("expected nil providers when lookup disabled")
	}
}

func TestLookupBinderProvidersByOperation(t *testing.T) {
	tests := []struct {
		name      string
		operation string
		mode      string
		rows      int
		txSize    int
		wantCall  string
		wantNeed  int64
	}{
		{name: "insert", operation: message.Insert, mode: base.ModeRow, rows: 2, wantCall: "insert", wantNeed: 2},
		{name: "update", operation: message.Update, mode: base.ModeRow, rows: 3, wantCall: "update", wantNeed: 3},
		{name: "delete", operation: message.Delete, mode: base.ModeTransaction, rows: 2, txSize: 4, wantCall: "delete", wantNeed: 8},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			seq := &stubSequencer{provider: &stubProvider{}}
			binder := base.NewLookupBinder(base.LookupConfig{
				Enabled:    true,
				Operations: []string{message.Insert, message.Update, message.Delete},
			}, seq)
			providers, err := binder.Providers(context.Background(), buildLookupPlan(tt.operation, tt.mode, tt.rows, tt.txSize))
			if err != nil {
				t.Fatalf("providers: %v", err)
			}
			if seq.lastMethod != tt.wantCall {
				t.Fatalf("unexpected sequencer call: %s", seq.lastMethod)
			}
			if seq.lastNeed != tt.wantNeed {
				t.Fatalf("unexpected provider need: %d", seq.lastNeed)
			}
			if len(providers) != 1 {
				t.Fatalf("expected one provider, got %d", len(providers))
			}
		})
	}
}

func TestLookupBinderProvidersRejectsMissingSpec(t *testing.T) {
	seq := &stubSequencer{provider: &stubProvider{}}
	binder := base.NewLookupBinder(base.LookupConfig{
		Enabled:    true,
		Operations: []string{message.Update},
	}, seq)
	_, err := binder.Providers(context.Background(), base.Plan{
		Operation:      message.Update,
		RowsPerMessage: 1,
		Target:         base.Target{Key: "db.users", Schema: struct{}{}},
	})
	if err == nil {
		t.Fatalf("expected missing sequence spec error")
	}
}

func TestLookupConfigEnabledFor(t *testing.T) {
	cfg := base.LookupConfig{Enabled: true, Operations: []string{message.Update, message.Delete}}
	if !cfg.EnabledFor(message.Update) {
		t.Fatalf("expected update enabled")
	}
	if cfg.EnabledFor(message.Insert) {
		t.Fatalf("expected insert disabled")
	}
}

func buildLookupPlan(operation, mode string, rows, txSize int) base.Plan {
	index := &mysql_schema.Index{Database: "db", Table: "users"}
	spec := iquery.SequenceSpec{
		Schema: index,
		Field:  "id",
		Fields: []iquery.BoundParam{{Column: "id", Type: "bigint"}},
	}
	return base.Plan{
		Builder:         "mysql",
		Mode:            mode,
		Operation:       operation,
		Target:          base.Target{Key: "db.users", Schema: struct{}{}, Extras: map[string]any{base.TargetExtraSequenceSpec: spec}},
		RowsPerMessage:  rows,
		TransactionSize: txSize,
	}
}
