package scenario

import (
	"context"
	"testing"

	mysql_schema "github.com/xuenqlve/common/relational_database/mysql"
	"github.com/xuenqlve/common/schema_store"
	"github.com/xuenqlve/kyogre/internal/plugin/iquery"
	"github.com/xuenqlve/kyogre/internal/plugin/metadata"
	genctx "github.com/xuenqlve/kyogre/pkg/generator_context"
	"github.com/xuenqlve/kyogre/pkg/message"
	base "github.com/xuenqlve/kyogre/pkg/scenario/base"
	mysqlScenario "github.com/xuenqlve/kyogre/pkg/scenario/mysql"
)

type stubSchemaStore struct {
	schemas map[string]any
}

func (s *stubSchemaStore) GetSchema(key schema_store.SchemaKey) (any, error) {
	return s.schemas[key.UniqueID()], nil
}
func (s *stubSchemaStore) InvalidateSchemaCache(schema_store.SchemaKey) {}
func (s *stubSchemaStore) InvalidateCache()                             {}
func (s *stubSchemaStore) IsInCache(schema_store.SchemaKey) bool        { return true }
func (s *stubSchemaStore) Close() error                                 { return nil }

type stubMetadata struct {
	keys     []schema_store.SchemaKey
	store    schema_store.SchemaStore
	fields   map[string][]iquery.BoundParam
	lookupOn bool
}

func (s *stubMetadata) Configure(string, map[string]any) error { return nil }
func (s *stubMetadata) Initialize(context.Context) error       { return nil }
func (s *stubMetadata) SchemaKeys() []schema_store.SchemaKey   { return s.keys }
func (s *stubMetadata) SchemaPrimaryField(key schema_store.SchemaKey) ([]iquery.BoundParam, error) {
	return s.fields[key.UniqueID()], nil
}
func (s *stubMetadata) IQueryEnabled() bool                   { return s.lookupOn }
func (s *stubMetadata) SchemaStore() schema_store.SchemaStore { return s.store }
func (s *stubMetadata) Close() error                          { return nil }

type stubProvider struct{}

func (s *stubProvider) Columns() []string      { return []string{"id"} }
func (s *stubProvider) Rows() []map[string]any { return []map[string]any{{"id": 1}} }

func TestMySQLBuilderLoadTargets(t *testing.T) {
	builder := &mysqlScenario.Builder{}
	if err := builder.Configure("test", nil); err != nil {
		t.Fatalf("configure builder: %v", err)
	}

	key := &mysql_schema.Index{Database: "test_db", Table: "users"}
	table := buildMySQLBuilderTable("test_db", "users")
	md := &stubMetadata{
		keys: []schema_store.SchemaKey{key},
		store: &stubSchemaStore{schemas: map[string]any{
			key.UniqueID(): table,
		}},
		fields: map[string][]iquery.BoundParam{
			key.UniqueID(): []iquery.BoundParam{{Column: "id", Type: "bigint"}},
		},
		lookupOn: true,
	}

	targets, err := builder.LoadTargets(md, nil)
	if err != nil {
		t.Fatalf("load targets: %v", err)
	}
	if len(targets) != 1 {
		t.Fatalf("expected 1 target, got %d", len(targets))
	}
	if targets[0].Key != "test_db.users" {
		t.Fatalf("unexpected target key: %s", targets[0].Key)
	}
	if targets[0].SequenceSpec == nil || targets[0].SequenceSpec.Schema.UniqueID() != key.UniqueID() {
		t.Fatalf("expected sequence spec bound to target")
	}
}

func TestMySQLBuilderBuildRowAndTransaction(t *testing.T) {
	builder := &mysqlScenario.Builder{}
	table := buildMySQLBuilderTable("test_db", "users")
	provider := &stubProvider{}
	plan := base.Plan{
		Builder:        string(mysqlScenario.BuilderType),
		Mode:           base.ModeRow,
		Operation:      message.Insert,
		Target:         base.Target{Key: "test_db.users", Schema: table},
		RowsPerMessage: 3,
		Columns:        []string{"id", "name"},
		Hint:           "/*+ test */",
		WriteType:      "insert",
		Providers: map[string]iquery.Provider{
			"test_db.users": provider,
		},
		Extras: map[string]any{"source": "unit"},
	}

	ctx, err := builder.Build(plan)
	if err != nil {
		t.Fatalf("build row context: %v", err)
	}
	rowCtx, ok := ctx.(*genctx.MySQLRowContext)
	if !ok {
		t.Fatalf("unexpected row context type: %T", ctx)
	}
	if rowCtx.Spec.Schema != table || rowCtx.Spec.Operation != message.Insert {
		t.Fatalf("unexpected row spec: %#v", rowCtx.Spec)
	}
	if rowCtx.Strategy().ResolveCount() != 3 {
		t.Fatalf("unexpected row count: %d", rowCtx.Strategy().ResolveCount())
	}
	if rowCtx.Provider("test_db.users") != provider {
		t.Fatalf("expected provider propagated")
	}
	if rowCtx.Extras()["source"] != "unit" {
		t.Fatalf("expected extras propagated")
	}

	plan.Mode = base.ModeTransaction
	plan.TransactionSize = 2
	ctx, err = builder.Build(plan)
	if err != nil {
		t.Fatalf("build transaction context: %v", err)
	}
	txCtx, ok := ctx.(*genctx.MySQLTransactionContext)
	if !ok {
		t.Fatalf("unexpected transaction context type: %T", ctx)
	}
	if len(txCtx.Rows) != 2 {
		t.Fatalf("expected 2 transaction rows, got %d", len(txCtx.Rows))
	}
	if txCtx.Strategy().ResolveCount() != 3 {
		t.Fatalf("unexpected transaction row count: %d", txCtx.Strategy().ResolveCount())
	}
}

func TestMySQLBuilderRejectsInvalidSchemaType(t *testing.T) {
	builder := &mysqlScenario.Builder{}
	_, err := builder.Build(base.Plan{
		Builder:        string(mysqlScenario.BuilderType),
		Mode:           base.ModeRow,
		Operation:      message.Insert,
		Target:         base.Target{Key: "broken", Schema: struct{}{}},
		RowsPerMessage: 1,
	})
	if err == nil {
		t.Fatalf("expected schema type mismatch error")
	}
}

func TestMySQLBuilderRegistered(t *testing.T) {
	builder, err := base.GetBuilder(mysqlScenario.BuilderType)
	if err != nil {
		t.Fatalf("get mysql builder: %v", err)
	}
	if _, ok := builder.(*mysqlScenario.Builder); !ok {
		t.Fatalf("unexpected builder type: %T", builder)
	}
}

func buildMySQLBuilderTable(database, tableName string) *mysql_schema.Table {
	table := &mysql_schema.Table{
		Database: database,
		Table:    tableName,
		Columns: []mysql_schema.Column{
			{Name: "id", DataType: "bigint"},
			{Name: "name", DataType: "varchar", RawType: "varchar(64)"},
		},
	}
	table.SetScanColumns([]string{"id"})
	return table
}

var _ metadata.Metadata = (*stubMetadata)(nil)
