package lookup

import (
	"testing"

	"github.com/xuenqlve/kyogre/internal/plugin/iquery"
	"github.com/xuenqlve/kyogre/pkg/tool/range_pool"
)

type testSchemaKey struct{ id string }

func (k testSchemaKey) UniqueID() string { return k.id }

func TestMemoryLookupLookupBoundsWrap(t *testing.T) {
	lookup := &MemoryLookup{}
	if err := lookup.Configure("test", map[string]any{"int-digits": 1}); err != nil {
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

	res, err := lookup.LookupBounds(nil, req)
	if err != nil {
		t.Fatalf("lookup err: %v", err)
	}
	if res.EnableLoop || res.Window.Start != 1 || res.Window.End != 3 {
		t.Fatalf("unexpected window: %+v", res)
	}

	res, err = lookup.LookupBounds(nil, req)
	if err != nil {
		t.Fatalf("lookup err: %v", err)
	}
	if res.EnableLoop || res.Window.Start != 4 || res.Window.End != 6 {
		t.Fatalf("unexpected window: %+v", res)
	}

	res, err = lookup.LookupBounds(nil, req)
	if err != nil {
		t.Fatalf("lookup err: %v", err)
	}
	if res.EnableLoop || res.Window.Start != 7 || res.Window.End != 9 {
		t.Fatalf("unexpected window: %+v", res)
	}

	res, err = lookup.LookupBounds(nil, iquery.LookupRequest{
		Schema:    schema,
		Params:    params,
		Partition: range_pool.RangePoolFreeName,
		Need:      2,
		WrapAt:    9,
	})
	if err != nil {
		t.Fatalf("lookup err: %v", err)
	}
	if !res.EnableLoop || res.Window.Start != 1 || res.Window.End != 2 {
		t.Fatalf("unexpected loop window: %+v", res)
	}
}

func TestMemoryLookupScanValuesComposite(t *testing.T) {
	lookup := &MemoryLookup{}
	if err := lookup.Configure("test", map[string]any{"string-length": 2, "int-digits": 1}); err != nil {
		t.Fatalf("configure err: %v", err)
	}
	schema := testSchemaKey{id: "t2"}
	cols := []iquery.BoundParam{{Column: "code", Type: "varchar"}, {Column: "seq", Type: "int"}}

	res, err := lookup.ScanValues(nil, iquery.ValuesRequest{Schema: schema, Columns: cols, Limit: 3})
	if err != nil {
		t.Fatalf("scan err: %v", err)
	}
	if len(res.Rows) != 3 {
		t.Fatalf("expected 3 rows, got %d", len(res.Rows))
	}
	expected := [][]any{{"aa", int64(0)}, {"aa", int64(1)}, {"aa", int64(2)}}
	for i := range expected {
		if res.Rows[i][0] != expected[i][0] || res.Rows[i][1] != expected[i][1] {
			t.Fatalf("row %d expected %v got %v", i, expected[i], res.Rows[i])
		}
	}
}
