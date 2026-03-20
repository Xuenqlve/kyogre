package mysql

import (
	"fmt"

	mysql_schema "github.com/xuenqlve/common/relational_database/mysql"
	"github.com/xuenqlve/common/schema_store"
	"github.com/xuenqlve/kyogre/internal/plugin/generator"
	"github.com/xuenqlve/kyogre/internal/plugin/iquery"
	"github.com/xuenqlve/kyogre/internal/plugin/metadata"
	genctx "github.com/xuenqlve/kyogre/pkg/generator_context"
	base "github.com/xuenqlve/kyogre/pkg/scenario/base"
)

const BuilderType base.BuilderType = "mysql"

type BuilderConfig struct{}

type Builder struct {
	pipeline string
	cfg      BuilderConfig
}

func (b *Builder) Configure(pipeline string, data map[string]any) error {
	b.pipeline = pipeline
	b.cfg = BuilderConfig{}
	return nil
}

func (b *Builder) LoadTargets(md metadata.Metadata, names []string) ([]base.Target, error) {
	if md == nil {
		return nil, fmt.Errorf("mysql builder metadata is nil")
	}
	store := md.SchemaStore()
	if store == nil {
		return nil, fmt.Errorf("mysql builder schema store is nil")
	}

	allowed := make(map[string]struct{}, len(names))
	for _, name := range names {
		if name != "" {
			allowed[name] = struct{}{}
		}
	}

	targets := make([]base.Target, 0, len(md.SchemaKeys()))
	for _, key := range md.SchemaKeys() {
		schema, err := store.GetSchema(key)
		if err != nil {
			return nil, fmt.Errorf("mysql builder load schema %s: %w", key.UniqueID(), err)
		}
		table, ok := schema.(*mysql_schema.Table)
		if !ok {
			return nil, fmt.Errorf("mysql builder schema %s type mismatch: %T", key.UniqueID(), schema)
		}
		targetKey := fmt.Sprintf("%s.%s", table.Database, table.Table)
		if len(allowed) > 0 {
			if _, ok := allowed[targetKey]; !ok {
				continue
			}
		}
		target := base.Target{
			Key:    targetKey,
			Schema: table,
		}
		if md.IQueryEnabled() {
			fields, err := md.SchemaPrimaryField(key)
			if err != nil {
				return nil, fmt.Errorf("mysql builder load sequence fields %s: %w", key.UniqueID(), err)
			}
			target.SequenceSpec = buildSequenceSpec(key, fields)
		}
		targets = append(targets, target)
	}
	return targets, nil
}

func (b *Builder) Build(plan base.Plan) (generator.GenerationContext, error) {
	if err := plan.Validate(); err != nil {
		return nil, err
	}
	table, ok := plan.Target.Schema.(*mysql_schema.Table)
	if !ok {
		return nil, fmt.Errorf("mysql builder target schema type mismatch: %T", plan.Target.Schema)
	}

	opts := make([]genctx.ContextOption, 0, len(plan.Providers)+2)
	opts = append(opts, genctx.WithStrategy(generator.NewSnapshot(generator.WithCountFixed(plan.RowsPerMessage))))
	if len(plan.Extras) > 0 {
		opts = append(opts, genctx.WithExtras(cloneMap(plan.Extras)))
	}
	for key, provider := range plan.Providers {
		if provider == nil {
			continue
		}
		opts = append(opts, genctx.WithIQueryProvider(key, provider))
	}

	switch plan.Mode {
	case base.ModeRow:
		return genctx.NewMySQLRowContext(buildRowSpec(plan, table), opts...), nil
	case base.ModeTransaction:
		rows := make([]genctx.MySQLRowSpec, 0, plan.TransactionSize)
		for i := 0; i < plan.TransactionSize; i++ {
			rows = append(rows, buildRowSpec(plan, table))
		}
		return genctx.NewMySQLTransactionContext(rows, opts...), nil
	default:
		return nil, fmt.Errorf("mysql builder unsupported plan mode: %s", plan.Mode)
	}
}

func buildRowSpec(plan base.Plan, table *mysql_schema.Table) genctx.MySQLRowSpec {
	return genctx.MySQLRowSpec{
		Operation: plan.Operation,
		Hint:      plan.Hint,
		WriteType: plan.WriteType,
		Columns:   append([]string(nil), plan.Columns...),
		Schema:    table,
	}
}

func buildSequenceSpec(key schema_store.SchemaKey, fields []iquery.BoundParam) *iquery.SequenceSpec {
	spec := iquery.MakeSequenceSpec(key, fields)
	return &spec
}

func cloneMap(src map[string]any) map[string]any {
	if len(src) == 0 {
		return nil
	}
	dst := make(map[string]any, len(src))
	for key, value := range src {
		dst[key] = value
	}
	return dst
}
