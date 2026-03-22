package mysql

import (
	"fmt"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/xuenqlve/common/errors"
	"github.com/xuenqlve/kyogre/internal/message"
	"github.com/xuenqlve/kyogre/internal/plugin/generator"
	genctx "github.com/xuenqlve/kyogre/pkg/generator_context"
	message2 "github.com/xuenqlve/kyogre/pkg/message"
)

const MySQL generator.Type = "mysql"

type Config struct{}

type Generator struct {
	pipeline string
	cfg      Config
}

func init() {
	generator.RegisterGenerator(MySQL, &Generator{}, false)
}

func (g *Generator) Configure(pipeline string, data map[string]any) error {
	g.pipeline = pipeline
	if err := mapstructure.Decode(data, &g.cfg); err != nil {
		return errors.Trace(err)
	}
	return nil
}

func (g *Generator) Kinds() []string {
	return []string{genctx.MySQLRow, genctx.MySQLTransaction, genctx.MySQLDDL}
}

func (g *Generator) Generate(ctx generator.GenerationContext) (message.Message, error) {
	if ctx == nil {
		return nil, fmt.Errorf("context is nil")
	}
	if err := ctx.Validate(); err != nil {
		return nil, err
	}
	switch typed := ctx.(type) {
	case *genctx.MySQLRowContext:
		return g.buildRowMessage(typed)
	case *genctx.MySQLTransactionContext:
		return g.buildTransactionMessage(typed)
	case *genctx.MySQLDDLContext:
		return g.buildDDLMessage(typed)
	default:
		return nil, fmt.Errorf("unsupported context type: %T", ctx)
	}
}

func (g *Generator) Close() {}

func (g *Generator) buildRowMessage(ctx *genctx.MySQLRowContext) (message.Message, error) {
	if ctx == nil {
		return nil, fmt.Errorf("mysql generator: row context is nil")
	}
	spec := ctx.Spec
	if spec.Schema == nil {
		return nil, fmt.Errorf("mysql generator: row schema is nil")
	}
	count := 1
	if snap := ctx.Strategy(); snap != nil {
		count = snap.ResolveCount()
	}
	if count <= 0 {
		return nil, nil
	}
	rowBuilder := NewRowBuilder(ctx.Strategy(), ctx.Providers())
	rows, err := rowBuilder.BuildRows(spec.Schema, spec.Operation, count, spec.Columns)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}
	rowMsg := &message2.MySQLRowMessage{
		SQLRows: message2.SQLRows{
			Metadata: message2.Metadata{
				Database:  spec.Schema.Database,
				Table:     spec.Schema.Table,
				Operation: spec.Operation,
				WriteType: spec.WriteType,
				Hint:      spec.Hint,
			},
			Contents: rows,
		},
	}
	return rowMsg, nil
}

func (g *Generator) buildTransactionMessage(ctx *genctx.MySQLTransactionContext) (message.Message, error) {
	if ctx == nil {
		return nil, fmt.Errorf("mysql generator: transaction context is nil")
	}
	if len(ctx.Rows) == 0 {
		return nil, nil
	}
	rowBuilder := NewRowBuilder(ctx.Strategy(), ctx.Providers())
	sqlRows := make([]message2.SQLRows, 0, len(ctx.Rows))
	for i := range ctx.Rows {
		spec := ctx.Rows[i]
		if spec.Schema == nil {
			return nil, fmt.Errorf("mysql generator: transaction row[%d] schema is nil", i)
		}
		count := 1
		if snap := ctx.Strategy(); snap != nil {
			count = snap.ResolveCount()
		}
		if count <= 0 {
			continue
		}
		rows, err := rowBuilder.BuildRows(spec.Schema, spec.Operation, count, spec.Columns)
		if err != nil {
			return nil, err
		}
		if len(rows) == 0 {
			continue
		}
		sqlRows = append(sqlRows, message2.SQLRows{
			Metadata: message2.Metadata{
				Database:  spec.Schema.Database,
				Table:     spec.Schema.Table,
				Operation: spec.Operation,
				WriteType: spec.WriteType,
				Hint:      spec.Hint,
			},
			Contents: rows,
		})
	}
	if len(sqlRows) == 0 {
		return nil, nil
	}
	return &message2.MySQLTransactionMessage{
		GTID:    fmt.Sprintf("transaction_%d", time.Now().UnixNano()),
		SQLRows: sqlRows,
	}, nil
}

func (g *Generator) buildDDLMessage(ctx *genctx.MySQLDDLContext) (message.Message, error) {
	if ctx == nil {
		return nil, fmt.Errorf("mysql generator: ddl context is nil")
	}
	spec := ctx.Spec
	if spec.Schema == nil {
		return nil, fmt.Errorf("mysql generator: ddl schema is nil")
	}
	statement, err := buildDDLStatement(spec)
	if err != nil {
		return nil, err
	}
	ddlMsg := &message2.MySQLDDLMessage{
		Metadata: message2.Metadata{
			Database:  spec.Schema.Database,
			Table:     spec.Schema.Table,
			Operation: spec.DDLType,
			WriteType: spec.DDLType,
		},
		DDLStatement: statement,
	}
	return ddlMsg, nil
}

var (
	_ message.Message = (*message2.MySQLRowMessage)(nil)
	_ message.Message = (*message2.MySQLTransactionMessage)(nil)
	_ message.Message = (*message2.MySQLDDLMessage)(nil)
)
