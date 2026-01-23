package iquery

import (
	"context"
	"fmt"
	"testing"

	"github.com/xuenqlve/kyogre/internal/plugin/iquery"
	"github.com/xuenqlve/kyogre/pkg/tool/range_pool"
)

type liveLookupStub struct {
	rangeCalls int
	scanCalls  int
	lastCursor any
	scanRows   [][]any
}

func (l *liveLookupStub) Configure(_ string, _ map[string]any) error { return nil }

func (l *liveLookupStub) LookupRange(_ context.Context, req iquery.Request) (iquery.RangeResult, error) {
	l.rangeCalls++
	l.lastCursor = req.Cursor
	if req.Cursor == nil {
		return iquery.RangeResult{}, fmt.Errorf("cursor is nil")
	}
	if _, ok := req.Cursor.(int64); !ok {
		return iquery.RangeResult{}, fmt.Errorf("cursor type %T", req.Cursor)
	}
	need := req.Need
	if need <= 0 {
		need = 1
	}
	return iquery.RangeResult{
		Window: range_pool.IntRange{Start: 1, End: need},
	}, nil
}

func (l *liveLookupStub) ScanValues(_ context.Context, _ iquery.Request) (iquery.ValuesResult, error) {
	l.scanCalls++
	return iquery.ValuesResult{
		Rows:    l.scanRows,
		HasMore: false,
	}, nil
}

func (l *liveLookupStub) Close() error { return nil }

func TestSequencerRoutesLiveLookupForScanValues(t *testing.T) {
	seq, err := iquery.NewSequencer()
	if err != nil {
		t.Fatalf("new sequencer err: %v", err)
	}
	live := &liveLookupStub{scanRows: [][]any{{"live-1"}}}
	seq.BindLookup(live)
	if err = seq.Configure("test", iquery.WithLookUpKey("live")); err != nil {
		t.Fatalf("configure err: %v", err)
	}

	spec := iquery.SequenceSpec{
		Schema: testSchemaKey{id: "seq-live"},
		Fields: []iquery.ColumnParam{{Column: "code", Type: "varchar"}},
	}

	provider, _, err := seq.ReserveInsertProvider(context.Background(), spec, 1)
	if err != nil {
		t.Fatalf("reserve insert err: %v", err)
	}

	if live.scanCalls != 0 {
		t.Fatalf("live lookup should not be used for insert scan, got %d", live.scanCalls)
	}

	provider, _, err = seq.ReserveUpdateProvider(context.Background(), spec, 1)
	if err != nil {
		t.Fatalf("reserve update err: %v", err)
	}
	if provider == nil {
		t.Fatalf("expected provider for update")
	}
	if live.scanCalls == 0 {
		t.Fatalf("expected live lookup ScanValues to be called")
	}
}

func TestSequencerPassesCursorToLookupRange(t *testing.T) {
	seq, err := iquery.NewSequencer()
	if err != nil {
		t.Fatalf("new sequencer err: %v", err)
	}
	live := &liveLookupStub{}
	seq.BindLookup(live)
	if err = seq.Configure("test", iquery.WithLookUpKey("live")); err != nil {
		t.Fatalf("configure err: %v", err)
	}

	spec := iquery.SequenceSpec{
		Schema: testSchemaKey{id: "seq-range"},
		Fields: []iquery.ColumnParam{{Column: "id", Type: "int"}},
	}

	if _, err = seq.ReserveUpdate(context.Background(), spec, 1); err != nil {
		t.Fatalf("reserve update err: %v", err)
	}
	if live.rangeCalls == 0 {
		t.Fatalf("expected live lookup LookupRange to be called")
	}
	if live.lastCursor == nil {
		t.Fatalf("expected cursor to be set")
	}
	if err = seq.Close(); err != nil {
		t.Fatalf("close err: %v", err)
	}
}
