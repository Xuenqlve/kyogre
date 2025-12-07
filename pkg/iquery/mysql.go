package iquery

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/go-faster/errors"
	"github.com/mitchellh/mapstructure"
	mysql_schema "github.com/xuenqlve/common/relational_database/mysql"
	"github.com/xuenqlve/common/schema_store"
	"github.com/xuenqlve/kyogre/internal/plugin/iquery"
	ds "github.com/xuenqlve/kyogre/pkg/data_source/mysql"
)

const (
	MySQL iquery.LookupType = "mysql"
)

func init() {
	iquery.RegisterIQuery(MySQL, &MySQLLookup{}, false)
}

type MySQLIQueryConfig struct {
	DataSource string `mapstructure:"data-source" json:"data-source"`
}

type MySQLLookup struct {
	pipeline string
	db       *sql.DB
	cfg      *MySQLIQueryConfig
	schema   schema_store.SchemaStore
}

func (q *MySQLLookup) Configure(pipeline string, data map[string]any) (err error) {
	q.pipeline = pipeline
	q.cfg = &MySQLIQueryConfig{}
	if err = mapstructure.Decode(data, q.cfg); err != nil {
		return
	}
	if q.db, err = ds.Connection(q.cfg.DataSource); err != nil {
		return
	}
	q.schema = schema_store.NewBaseSchemaStore(mysql_schema.NewSchema(q.db))
	return
}

func (q *MySQLLookup) Lookup(ctx context.Context, req iquery.LookupRequest) ([]iquery.LookupResult, error) {
	results := make([]iquery.LookupResult, 0, len(req.Items))
	for _, item := range req.Items {
		if item.Schema == nil || item.Field == "" {
			continue
		}
		tableDef, err := q.queryTableDef(item.Schema)
		if err != nil {
			return nil, err
		}
		maxValue, err := q.queryMax(ctx, tableDef, item.Field)
		if err != nil {
			return nil, err
		}
		rowCount, err := q.queryRowCount(ctx, tableDef)
		if err != nil {
			return nil, err
		}
		results = append(results, iquery.LookupResult{
			Schema: item.Schema,
			Field:  item.Field,
			Max:    maxValue,
			Rows:   rowCount,
			Extras: map[string]any{},
		})
	}
	return results, nil
}

func (q *MySQLLookup) Close() error {
	if q.db == nil {
		return nil
	}
	return q.db.Close()
}

func (q *MySQLLookup) queryTableDef(key schema_store.SchemaKey) (*mysql_schema.Table, error) {
	schema, err := q.schema.GetSchema(key)
	if err != nil {
		return nil, err
	}
	tableDef, ok := schema.(*mysql_schema.Table)
	if !ok {
		return nil, errors.Errorf("invalid table definition type, expect *mysql_schema.Table, got %T", schema)
	}
	return tableDef, nil
}

func (q *MySQLLookup) queryMax(ctx context.Context, table *mysql_schema.Table, field string) (int64, error) {
	query := fmt.Sprintf("SELECT MAX(%s) FROM %s.%s",
		quoteIdentifier(field),
		quoteIdentifier(table.Database),
		quoteIdentifier(table.Table),
	)
	var maxValue sql.NullInt64
	if err := q.db.QueryRowContext(ctx, query).Scan(&maxValue); err != nil {
		return 0, err
	}
	if !maxValue.Valid {
		return 0, nil
	}
	return maxValue.Int64, nil
}

func (q *MySQLLookup) queryRowCount(ctx context.Context, table *mysql_schema.Table) (int64, error) {
	query := fmt.Sprintf("SELECT COUNT(*) FROM %s.%s",
		quoteIdentifier(table.Database),
		quoteIdentifier(table.Table),
	)
	var count int64
	if err := q.db.QueryRowContext(ctx, query).Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

func quoteIdentifier(name string) string {
	return fmt.Sprintf("`%s`", strings.ReplaceAll(name, "`", "``"))
}
