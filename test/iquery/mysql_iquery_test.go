package iquery

import (
	"context"
	"testing"

	mysql_schema "github.com/xuenqlve/common/relational_database/mysql"
	"github.com/xuenqlve/common/transform"
	"github.com/xuenqlve/kyogre/internal/plugin/iquery"
	"github.com/xuenqlve/kyogre/internal/plugin/metadata"
	"github.com/xuenqlve/kyogre/pkg/lookup"
	"github.com/xuenqlve/kyogre/pkg/metadata/mysql"
	//"github.com/xuenqlve/kyogre/pkg/metadata_template"
)

func templateMySQLMetadata(template, datasource string) (md metadata.Metadata, err error) {
	if md, err = metadata.GetMetadata(mysql.MySQL); err != nil {
		return
	}
	cfg := map[string]interface{}{
		"data-source": datasource,
		"template":    template,
	}
	if err = md.Configure(pipeline, cfg); err != nil {
		return
	}
	if err = md.Initialize(context.Background()); err != nil {
		return
	}
	return md, nil
}

func TestMySQLIQuery(t *testing.T) {
	ctx := context.Background()
	lk, err := iquery.GetIQueryModule(lookup.MySQL)
	if err != nil {
		t.Error(err)
		return
	}
	defer lk.Close()

	if err = lk.Configure(pipeline, map[string]any{"data-source": mysqlDataSource}); err != nil {
		t.Fatalf("configure err: %v", err)
	}
	t.Run("ScanValues", func(t *testing.T) {
		database := "test"
		table := "composite_test"
		req := iquery.ValueRequest{
			Schema:  &mysql_schema.Index{Database: database, Table: table},
			Columns: []iquery.ColumnParam{{Column: "id", Type: "int"}, {Column: "name", Type: "varchar"}},
			Need:    3,
		}
		res, err := lk.ScanValues(ctx, req)
		if err != nil {
			t.Fatalf("scan values err: %v", err)
		}
		t.Logf("HasMore:%v NextCursor:%v", res.HasMore, res.NextCursor)
		for _, row := range res.Rows {
			t.Logf("rows:%v", row)
		}
	})

	t.Run("LookupRange", func(t *testing.T) {
		database := "test"
		table := "user"
		req := iquery.RangeRequest{
			Schema:  &mysql_schema.Index{Database: database, Table: table},
			Columns: []iquery.ColumnParam{{Column: "id", Type: "int64"}},
			Need:    3,
			Cursor:  100,
		}
		res := iquery.RangeResult{}
		if res, err = lk.LookupRange(ctx, req); err != nil {
			t.Logf("LookupRange err: %v", err)
			return
		}
		t.Logf("EnableLoop:%v %d~%d", res.EnableLoop, res.Window.Start, res.Window.End)
	})
}

func assertRowInts(t *testing.T, row map[string]any, wantA, wantB int64) {
	t.Helper()
	a, err := transform.ToInt(row["a"])
	if err != nil {
		t.Fatalf("row a convert err: %v", err)
	}
	b, err := transform.ToInt(row["b"])
	if err != nil {
		t.Fatalf("row b convert err: %v", err)
	}
	if a != wantA || b != wantB {
		t.Fatalf("row mismatch: got (%d,%d) want (%d,%d)", a, b, wantA, wantB)
	}
}
