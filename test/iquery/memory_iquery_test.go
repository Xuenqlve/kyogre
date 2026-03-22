package iquery

import (
	"context"
	"testing"

	"github.com/xuenqlve/kyogre/internal/plugin/iquery"
)

type testSchemaKey struct{ id string }

func (k testSchemaKey) UniqueID() string { return k.id }

func TestMemoryLookupLookupRangeWrap(t *testing.T) {
	lk := &iquery.MemoryLookup{}
	ctx := context.Background()
	if err := lk.Configure("test", map[string]any{"int-digits": 1, "wrap": true}); err != nil {
		t.Fatalf("configure err: %v", err)
	}
	schema := testSchemaKey{id: "t1"}
	params := []iquery.BoundParam{{Column: "id", Type: "tinyint"}}

	req := iquery.RangeRequest{
		Schema:  schema,
		Columns: params,
		Need:    3,
	}

	res, err := lk.LookupRange(ctx, req)
	if err != nil {
		t.Fatalf("lookup err: %v", err)
	}
	t.Logf("lookup result: %+v", res)

	req.Cursor = res.Window.End
	res, err = lk.LookupRange(ctx, req)
	if err != nil {
		t.Fatalf("lookup err: %v", err)
	}
	t.Logf("lookup result: %+v", res)

	req.Cursor = res.Window.End
	res, err = lk.LookupRange(ctx, req)
	if err != nil {
		t.Fatalf("lookup err: %v", err)
	}
	t.Logf("lookup result: %+v", res)

	res, err = lk.LookupRange(ctx, iquery.RangeRequest{
		Schema:  schema,
		Columns: params,
		Need:    4,
		Cursor:  int64(127),
	})
	if err != nil {
		t.Fatalf("lookup err: %v", err)
	}
	t.Logf("lookup result: %+v", res)
}

func TestMemoryLookupScanValuesComposite(t *testing.T) {
	lk := &iquery.MemoryLookup{}
	ctx := context.Background()
	if err := lk.Configure("test", map[string]any{"int-digits": 1}); err != nil {
		t.Fatalf("configure err: %v", err)
	}
	schema := testSchemaKey{id: "t2"}
	//cols := []iquery.BoundParam{{Column: "code", Type: "varchar"}, {Column: "seq", Type: "int"}, {Column: "size", Type: "int"}}
	cols := []iquery.BoundParam{{Column: "seq", Type: "int"}}

	var nextCursor map[string]any
	for i := 0; i < 10; i++ {
		res, err := lk.ScanValues(ctx, iquery.ValueRequest{Schema: schema, Columns: cols, Need: 6, Cursor: nextCursor})
		if err != nil {
			t.Fatalf("scan err: %v", err)
		}
		nextCursor = res.NextCursor
		for _, row := range res.Rows {
			t.Logf("colums: %+v", row)
		}
		t.Logf("index:%d HasMore:%v NextCursor:%v", i, res.HasMore, res.NextCursor)
	}

}

func TestMemoryLookupScanValuesCursorComposite(t *testing.T) {
	lk := &iquery.MemoryLookup{}
	if err := lk.Configure("test", map[string]any{"string-length": 2, "int-digits": 1}); err != nil {
		t.Fatalf("configure err: %v", err)
	}
	schema := testSchemaKey{id: "t3"}
	cols := []iquery.BoundParam{{Column: "code", Type: "varchar"}, {Column: "seq", Type: "int"}}

	cursor := map[string]any{"code": "aa", "seq": int64(5)}
	res, err := lk.ScanValues(nil, iquery.ValueRequest{Schema: schema, Columns: cols, Cursor: cursor, Need: 7})
	if err != nil {
		t.Fatalf("scan err: %v", err)
	}
	//expected := []map[string]any{{"code": "aa", "seq": int64(3)}, {"code": "aa", "seq": int64(4)}}
	for i, row := range res.Rows {
		t.Logf("index:%d row: %+v", i, row)
	}
}
