package generator

import (
	"testing"

	mysql_schema "github.com/xuenqlve/common/relational_database/mysql"
	mysql_generator "github.com/xuenqlve/kyogre/pkg/generator/mysql"
	genctx "github.com/xuenqlve/kyogre/pkg/generator_context"
	"github.com/xuenqlve/kyogre/pkg/message"
)

func TestMySQLGenerator(t *testing.T) {
	gen := &mysql_generator.Generator{}
	if err := gen.Configure("test", map[string]any{}); err != nil {
		t.Fatalf("configure generator: %v", err)
	}

	table := buildMySQLTable()

	rowCtx := genctx.NewMySQLRowContext(genctx.MySQLRowSpec{
		Operation: message.Insert,
		Schema:    table,
	})
	rowMsg, err := gen.Generate(rowCtx)
	if err != nil {
		t.Fatalf("generate row message: %v", err)
	}
	t.Logf("row message: %#v", rowMsg)

	txCtx := genctx.NewMySQLTransactionContext([]genctx.MySQLRowSpec{
		{
			Operation: message.Insert,
			Schema:    table,
		},
		{
			Operation: message.Update,
			Schema:    table,
			Columns:   []string{"name"},
		},
	})
	txMsg, err := gen.Generate(txCtx)
	if err != nil {
		t.Fatalf("generate transaction message: %v", err)
	}
	t.Logf("transaction message: %#v", txMsg)

	ddlCtx := genctx.NewMySQLDDLContext(genctx.MySQLDDLSpec{
		DDLType: "create table",
		Schema:  table,
	})
	ddlMsg, err := gen.Generate(ddlCtx)
	if err != nil {
		t.Fatalf("generate ddl message: %v", err)
	}
	t.Logf("ddl message: %#v", ddlMsg)
}

func buildMySQLTable() *mysql_schema.Table {
	table := &mysql_schema.Table{
		Database: "test_db",
		Table:    "users",
		Columns: []mysql_schema.Column{
			{
				Name:     "id",
				DataType: "int",
			},
			{
				Name:     "name",
				DataType: "varchar",
				RawType:  "varchar(64)",
			},
			{
				Name:     "email",
				DataType: "varchar",
				RawType:  "varchar(128)",
			},
		},
	}
	table.SetScanColumns([]string{"id"})
	return table
}
