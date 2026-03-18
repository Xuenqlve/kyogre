package scenario

import (
	"context"
	"testing"

	"github.com/xuenqlve/kyogre/internal/plugin/generator"
	pluginMetadata "github.com/xuenqlve/kyogre/internal/plugin/metadata"
	genctx "github.com/xuenqlve/kyogre/pkg/generator_context"
	"github.com/xuenqlve/kyogre/pkg/message"
	mysqlmetadata "github.com/xuenqlve/kyogre/pkg/metadata/mysql"
	mysqlscenario "github.com/xuenqlve/kyogre/pkg/scenario/mysql"
)

func TestMySQLScenarioRowContext(t *testing.T) {
	md := buildMySQLMetadata(t)

	sc := &mysqlscenario.Scenario{}
	err := sc.Configure(pipelineName, map[string]any{
		"mode":             "row",
		"message-count":    2,
		"operation":        message.Insert,
		"rows-per-message": 3,
		"schemas":          []string{"kyogre.users"},
	})
	if err != nil {
		t.Fatalf("configure mysql scenario: %v", err)
	}

	ctxChan := make(chan generator.GenerationContext, 2)
	sc.Start(context.Background(), md, nil, ctxChan)

	for i := 0; i < 2; i++ {
		genCtx := <-ctxChan
		rowCtx, ok := genCtx.(*genctx.MySQLRowContext)
		if !ok {
			t.Fatalf("unexpected context type: %T", genCtx)
		}
		if rowCtx.Spec.Schema == nil {
			t.Fatalf("row context schema is nil")
		}
		if rowCtx.Spec.Schema.Database != "kyogre" || rowCtx.Spec.Schema.Table != "users" {
			t.Fatalf("unexpected schema: %s.%s", rowCtx.Spec.Schema.Database, rowCtx.Spec.Schema.Table)
		}
		if rowCtx.Spec.Operation != message.Insert {
			t.Fatalf("unexpected operation: %s", rowCtx.Spec.Operation)
		}
		if got := rowCtx.Strategy().ResolveCount(); got != 3 {
			t.Fatalf("unexpected row count: %d", got)
		}
	}
}

func TestMySQLScenarioTransactionContext(t *testing.T) {
	md := buildMySQLMetadata(t)

	sc := &mysqlscenario.Scenario{}
	err := sc.Configure(pipelineName, map[string]any{
		"mode":             "transaction",
		"message-count":    1,
		"operation":        message.Insert,
		"rows-per-message": 2,
		"transaction-size": 3,
		"schemas":          []string{"kyogre.orders"},
	})
	if err != nil {
		t.Fatalf("configure mysql scenario: %v", err)
	}

	ctxChan := make(chan generator.GenerationContext, 1)
	sc.Start(context.Background(), md, nil, ctxChan)

	genCtx := <-ctxChan
	txCtx, ok := genCtx.(*genctx.MySQLTransactionContext)
	if !ok {
		t.Fatalf("unexpected context type: %T", genCtx)
	}
	if len(txCtx.Rows) != 3 {
		t.Fatalf("unexpected transaction size: %d", len(txCtx.Rows))
	}
	for i := range txCtx.Rows {
		row := txCtx.Rows[i]
		if row.Schema == nil {
			t.Fatalf("transaction row[%d] schema is nil", i)
		}
		if row.Schema.Database != "kyogre" || row.Schema.Table != "orders" {
			t.Fatalf("unexpected schema in row[%d]: %s.%s", i, row.Schema.Database, row.Schema.Table)
		}
		if row.Operation != message.Insert {
			t.Fatalf("unexpected operation in row[%d]: %s", i, row.Operation)
		}
	}
	if got := txCtx.Strategy().ResolveCount(); got != 2 {
		t.Fatalf("unexpected transaction row count: %d", got)
	}
}

func buildMySQLMetadata(t *testing.T) pluginMetadata.Metadata {
	t.Helper()

	md, err := pluginMetadata.GetMetadata(mysqlmetadata.MySQL)
	if err != nil {
		t.Fatalf("get mysql metadata: %v", err)
	}
	err = md.Configure(pipelineName, map[string]any{
		"data-source": mysqlmetadata.MockDataSource,
		"template":    mysqlmetadata.CustomizeTemplate,
		"databases": mysqlmetadata.Database{
			"kyogre": mysqlmetadata.Tables{
				{
					Table: "users",
					Columns: []mysqlmetadata.Column{
						{Column: "id", Type: mysqlmetadata.Primary},
						{Column: "name", Type: mysqlmetadata.String},
					},
					Indexes: []mysqlmetadata.Index{
						{Name: "PRIMARY", Columns: []string{"id"}, IsPrimary: true},
					},
				},
				{
					Table: "orders",
					Columns: []mysqlmetadata.Column{
						{Column: "id", Type: mysqlmetadata.Primary},
						{Column: "amount", Type: mysqlmetadata.Decimal},
					},
					Indexes: []mysqlmetadata.Index{
						{Name: "PRIMARY", Columns: []string{"id"}, IsPrimary: true},
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("configure mysql metadata: %v", err)
	}
	if err = md.Initialize(context.Background()); err != nil {
		t.Fatalf("initialize mysql metadata: %v", err)
	}
	t.Cleanup(func() {
		_ = md.Close()
	})
	return md
}
