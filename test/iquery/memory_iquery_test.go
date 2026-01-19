package iquery

import (
	"context"
	"testing"

	"github.com/xuenqlve/kyogre/internal/plugin/iquery"
	"github.com/xuenqlve/kyogre/pkg/lookup"
	"github.com/xuenqlve/kyogre/pkg/tool/range_pool"
)

type testSchemaKey struct{ id string }

func (k testSchemaKey) UniqueID() string { return k.id }

func TestMemoryLookupLookupBoundsWrap(t *testing.T) {
	lk := &lookup.MemoryLookup{}
	ctx := context.Background()
	if err := lk.Configure("test", map[string]any{"int-digits": 1}); err != nil {
		t.Fatalf("configure err: %v", err)
	}
	schema := testSchemaKey{id: "t1"}
	params := []iquery.BoundParam{{Column: "id", Type: "int"}}

	req := iquery.LookupRequest{
		Schema:    schema,
		Params:    params,
		Partition: range_pool.RangePoolFreeName,
		Need:      3,
		WrapAt:    9,
	}

	res, err := lk.LookupBounds(ctx, req)
	if err != nil {
		t.Fatalf("lookup err: %v", err)
	}
	t.Logf("lookup result: %+v", res)

	res, err = lk.LookupBounds(ctx, req)
	if err != nil {
		t.Fatalf("lookup err: %v", err)
	}
	t.Logf("lookup result: %+v", res)

	res, err = lk.LookupBounds(ctx, req)
	if err != nil {
		t.Fatalf("lookup err: %v", err)
	}
	t.Logf("lookup result: %+v", res)

	res, err = lk.LookupBounds(ctx, iquery.LookupRequest{
		Schema:    schema,
		Params:    params,
		Partition: range_pool.RangePoolFreeName,
		Need:      4,
		WrapAt:    9,
	})
	if err != nil {
		t.Fatalf("lookup err: %v", err)
	}
	t.Logf("lookup result: %+v", res)
}

func TestMemoryLookupScanValuesComposite(t *testing.T) {
	lk := &lookup.MemoryLookup{}
	ctx := context.Background()
	if err := lk.Configure("test", map[string]any{"string-length": 1, "int-digits": 1}); err != nil {
		t.Fatalf("configure err: %v", err)
	}
	schema := testSchemaKey{id: "t2"}
	cols := []iquery.BoundParam{{Column: "code", Type: "varchar"}, {Column: "seq", Type: "int"}, {Column: "size", Type: "int"}}

	var nextCursor any
	for i := 0; i < 100; i++ {
		res, err := lk.ScanValues(ctx, iquery.ValuesRequest{Schema: schema, Columns: cols, Limit: 6, Cursor: nextCursor})
		if err != nil {
			t.Fatalf("scan err: %v", err)
		}
		nextCursor = res.NextCursor
		t.Logf("HasMore:%v NextCursor:%v", res.HasMore, res.NextCursor)
		for _, row := range res.Rows {
			t.Logf("colums: %+v", row)
		}

	}

}
