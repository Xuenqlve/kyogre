package iquery

import (
	"context"
	"database/sql"

	"github.com/go-faster/errors"
	"github.com/mitchellh/mapstructure"
	mysql_schema "github.com/xuenqlve/common/relational_database/mysql"
	"github.com/xuenqlve/common/schema_store"
	"github.com/xuenqlve/kyogre/internal/plugin"
	ds "github.com/xuenqlve/kyogre/pkg/data_source/mysql"
)

const (
	MySQL plugin.IQueryType = "mysql"
)

func init() {
	plugin.RegisterIQuery(MySQL, &MySQLIQuery{}, false)
}

type MySQLIQueryConfig struct {
	DataSource string `mapstructure:"data-source" json:"data-source"`
}

type MySQLIQuery struct {
	pipeline string
	db       *sql.DB
	cfg      *MySQLIQueryConfig
	schema   schema_store.SchemaStore
}

func (q *MySQLIQuery) Configure(pipeline string, data map[string]any) (err error) {
	q.pipeline = pipeline
	if err = mapstructure.Decode(data, q.cfg); err != nil {
		return
	}
	if q.db, err = ds.Connection(q.cfg.DataSource); err != nil {
		return
	}
	q.schema = schema_store.NewBaseSchemaStore(mysql_schema.NewSchema(q.db))
	return
}

func (q *MySQLIQuery) queryTableDef(key schema_store.SchemaKey) (*mysql_schema.Table, error) {
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

func (q *MySQLIQuery) QueryMaxValue(ctx context.Context, key schema_store.SchemaKey, field string) (int64, error) {
	tableDef, err := q.queryTableDef(key)
	if err != nil {
		return 0, err
	}

	query := "SELECT MAX(" + field + ") FROM " + tableDef.Database + "." + tableDef.Table
	var maxValue sql.NullInt64
	err = q.db.QueryRowContext(ctx, query).Scan(&maxValue)
	if err != nil {
		return 0, err
	}

	if !maxValue.Valid {
		return 0, nil
	}
	return maxValue.Int64, nil
}

func (q *MySQLIQuery) QueryRowCount(ctx context.Context, key schema_store.SchemaKey) (int64, error) {
	tableDef, err := q.queryTableDef(key)
	if err != nil {
		return 0, err
	}

	query := "SELECT COUNT(*) FROM " + tableDef.Database + "." + tableDef.Table
	var count int64
	err = q.db.QueryRowContext(ctx, query).Scan(&count)
	if err != nil {
		return 0, err
	}
	return count, nil
}

func (q *MySQLIQuery) BatchQuery(ctx context.Context, keys []schema_store.SchemaKey) ([]*plugin.QueryResult, error) {
	results := make([]*plugin.QueryResult, 0, len(keys))
	for _, key := range keys {
		// For batch query, get the max value of the first primary key
		tableDef, err := q.queryTableDef(key)
		if err != nil {
			return nil, err
		}

		var field string
		if len(tableDef.PrimaryIndex) > 0 {
			field = tableDef.PrimaryIndex[0]
		} else if len(tableDef.Columns) > 0 {
			field = tableDef.Columns[0].Name
		} else {
			// If no suitable field, continue to next key
			continue
		}

		result, err := q.GetQueryResult(ctx, key, field)
		if err != nil {
			return nil, err
		}
		results = append(results, result)
	}
	return results, nil
}

func (q *MySQLIQuery) GetQueryResult(ctx context.Context, key schema_store.SchemaKey, field string) (*plugin.QueryResult, error) {
	maxValue, err := q.QueryMaxValue(ctx, key, field)
	if err != nil {
		return nil, err
	}

	rowCount, err := q.QueryRowCount(ctx, key)
	if err != nil {
		return nil, err
	}

	return &plugin.QueryResult{
		TableKey:        key,
		Field:           field,
		MaxValue:        maxValue,
		CurrentRowCount: rowCount,
		Metadata:        make(map[string]any),
	}, nil
}

func (q *MySQLIQuery) Close() error {
	return q.db.Close()
}
