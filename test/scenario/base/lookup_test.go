package base_test

import (
	"context"
	"testing"

	"github.com/xuenqlve/kyogre/internal/plugin/iquery"
	"github.com/xuenqlve/kyogre/pkg/message"
	base "github.com/xuenqlve/kyogre/pkg/scenario/base"
	"github.com/xuenqlve/kyogre/pkg/tool/range_pool"
)

type stubProvider struct{}

func (s *stubProvider) Columns() []string      { return []string{"id"} }
func (s *stubProvider) Rows() []map[string]any { return []map[string]any{{"id": 1}} }

type liveLookupStub struct {
	rangeCalls int
	scanCalls  int
	lastNeed   int64
	lastCursor int64
}

func (l *liveLookupStub) Configure(_ string, _ map[string]any) error { return nil }

func (l *liveLookupStub) LookupRange(_ context.Context, req iquery.RangeRequest) (iquery.RangeResult, error) {
	l.rangeCalls++
	l.lastNeed = req.Need
	l.lastCursor = req.Cursor
	need := req.Need
	if need <= 0 {
		need = 1
	}
	return iquery.RangeResult{
		Window: range_pool.IntRange{Start: 1, End: need},
	}, nil
}

func (l *liveLookupStub) ScanValues(_ context.Context, req iquery.ValueRequest) (iquery.ValuesResult, error) {
	l.scanCalls++
	l.lastNeed = req.Need
	rows := make([]map[string]any, 0, req.Need)
	for i := int64(0); i < req.Need; i++ {
		rows = append(rows, map[string]any{"id": i + 1})
	}
	return iquery.ValuesResult{Rows: rows, HasMore: false}, nil
}

func (l *liveLookupStub) Close() error { return nil }

type testSchemaKey struct{ id string }

func (k testSchemaKey) UniqueID() string { return k.id }

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
		assert    func(t *testing.T, live *liveLookupStub, providers map[string]iquery.Provider)
	}{
		{
			name:      "insert",
			operation: message.Insert,
			mode:      base.ModeRow,
			rows:      2,
			assert: func(t *testing.T, live *liveLookupStub, providers map[string]iquery.Provider) {
				if len(providers) != 1 {
					t.Fatalf("expected one provider, got %d", len(providers))
				}
			},
		},
		{
			name:      "update",
			operation: message.Update,
			mode:      base.ModeRow,
			rows:      3,
			assert: func(t *testing.T, live *liveLookupStub, providers map[string]iquery.Provider) {
				if live.rangeCalls == 0 {
					t.Fatalf("expected update to use live range lookup")
				}
				if live.lastNeed <= 0 {
					t.Fatalf("expected positive live lookup need, got %d", live.lastNeed)
				}
				if len(providers) != 1 {
					t.Fatalf("expected one provider, got %d", len(providers))
				}
			},
		},
		{
			name:      "delete",
			operation: message.Delete,
			mode:      base.ModeTransaction,
			rows:      2,
			txSize:    4,
			assert: func(t *testing.T, live *liveLookupStub, providers map[string]iquery.Provider) {
				if live.rangeCalls == 0 {
					t.Fatalf("expected delete to use live range lookup")
				}
				if live.lastNeed <= 0 {
					t.Fatalf("expected positive live lookup need, got %d", live.lastNeed)
				}
				if len(providers) != 1 {
					t.Fatalf("expected one provider, got %d", len(providers))
				}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			seq, err := iquery.NewSequencer()
			if err != nil {
				t.Fatalf("new sequencer: %v", err)
			}
			live := &liveLookupStub{}
			seq.BindLookup(live)
			if err = seq.Configure("test", iquery.WithLookUpKey("live")); err != nil {
				t.Fatalf("configure sequencer: %v", err)
			}
			binder := base.NewLookupBinder(base.LookupConfig{
				Enabled:    true,
				Operations: []string{message.Insert, message.Update, message.Delete},
			}, seq)
			providers, err := binder.Providers(context.Background(), buildLookupPlan(tt.operation, tt.mode, tt.rows, tt.txSize))
			if err != nil {
				t.Fatalf("providers: %v", err)
			}
			tt.assert(t, live, providers)
		})
	}
}

func TestLookupBinderProvidersRejectsMissingSpec(t *testing.T) {
	seq, err := iquery.NewSequencer()
	if err != nil {
		t.Fatalf("new sequencer: %v", err)
	}
	if err = seq.Configure("test", iquery.WithLookUpKey("live")); err != nil {
		t.Fatalf("configure sequencer: %v", err)
	}
	binder := base.NewLookupBinder(base.LookupConfig{
		Enabled:    true,
		Operations: []string{message.Update},
	}, seq)
	_, err = binder.Providers(context.Background(), base.Plan{
		Operation:      message.Update,
		RowsPerMessage: 1,
		Target:         base.Target{Key: "db.users", Schema: struct{}{}},
	})
	if err == nil {
		t.Fatalf("expected missing sequence spec error")
	}
}

func TestLookupBinderProvidersRejectsNilSequencer(t *testing.T) {
	binder := base.NewLookupBinder(base.LookupConfig{
		Enabled:    true,
		Operations: []string{message.Update},
	}, nil)
	_, err := binder.Providers(context.Background(), buildLookupPlan(message.Update, base.ModeRow, 1, 0))
	if err == nil {
		t.Fatalf("expected nil sequencer error")
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
	spec := &iquery.SequenceSpec{
		Schema: testSchemaKey{id: "db.users"},
		Field:  "id",
		Fields: []iquery.BoundParam{{Column: "id", Type: "bigint"}},
	}
	return base.Plan{
		Builder:         "mysql",
		Mode:            mode,
		Operation:       operation,
		Target:          base.Target{Key: "db.users", Schema: struct{}{}, SequenceSpec: spec},
		RowsPerMessage:  rows,
		TransactionSize: txSize,
	}
}
