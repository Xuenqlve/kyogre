package lookup

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/mitchellh/mapstructure"
	mysql_schema "github.com/xuenqlve/common/relational_database/mysql"
	"github.com/xuenqlve/common/schema_store"
	"github.com/xuenqlve/common/transform"
	"github.com/xuenqlve/kyogre/internal/plugin/iquery"
	ds "github.com/xuenqlve/kyogre/pkg/data_source/mysql"
	"github.com/xuenqlve/kyogre/pkg/tool/range_pool"
)

const (
	MySQL iquery.LookupType = "mysql"
)

func init() {
	iquery.RegisterIQuery(MySQL, &MySQLLookup{}, false)
}

type MySQLIQueryConfig struct {
	DataSource string `mapstructure:"data-source" json:"data-source"`
	FreeMinID  int64  `mapstructure:"free-min-id" json:"free-min-id"`
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
	if q.cfg.FreeMinID <= 0 {
		q.cfg.FreeMinID = 1
	}
	if q.db, err = ds.Connection(q.cfg.DataSource); err != nil {
		return
	}
	q.schema = schema_store.NewBaseSchemaStore(mysql_schema.NewSchema(q.db))
	return nil
}

func (q *MySQLLookup) LookupRange(ctx context.Context, req iquery.Request) (iquery.RangeResult, error) {
	tableDef, err := q.queryTableDef(req.Schema)
	if err != nil {
		return iquery.RangeResult{}, err
	}
	if len(req.Columns) == 0 || req.Columns[0].Column == "" {
		return iquery.RangeResult{}, fmt.Errorf("lookup params empty")
	}

	minV, maxV, cnt, err := q.queryMinMaxCount(ctx, tableDef, req.Columns[0].Column)
	if err != nil {
		return iquery.RangeResult{}, err
	}

	if cnt == 0 || maxV == nil {
		return iquery.RangeResult{}, fmt.Errorf("empty live range for %s", req.Schema.UniqueID())
	}

	minID, err := transform.ToInt(minV)
	if err != nil {
		return iquery.RangeResult{}, err
	}
	maxID, err := transform.ToInt(maxV)
	if err != nil {
		return iquery.RangeResult{}, err
	}

	if minID > maxID {
		return iquery.RangeResult{}, fmt.Errorf("lookup returned invalid window for %s", req.Schema.UniqueID())
	}
	return iquery.RangeResult{Window: range_pool.IntRange{Start: minID, End: maxID}}, nil
}

func (q *MySQLLookup) ScanValues(ctx context.Context, req iquery.Request) (iquery.ValuesResult, error) {
	tableDef, err := q.queryTableDef(req.Schema)
	if err != nil {
		return iquery.ValuesResult{}, err
	}
	if len(req.Columns) == 0 {
		return iquery.ValuesResult{}, nil
	}
	limit := int(req.Need)
	if limit <= 0 {
		limit = 1000
	}

	cols := make([]string, 0, len(req.Columns))
	for _, c := range req.Columns {
		if c.Column == "" {
			continue
		}
		cols = append(cols, quoteIdentifier(c.Column))
	}
	if len(cols) == 0 {
		return iquery.ValuesResult{}, nil
	}

	base := fmt.Sprintf("SELECT %s FROM %s.%s",
		strings.Join(cols, ","),
		quoteIdentifier(tableDef.Database),
		quoteIdentifier(tableDef.Table),
	)
	args := make([]any, 0, 2)
	if req.Cursor != nil {
		base += fmt.Sprintf(" WHERE %s > ?", cols[0])
		args = append(args, req.Cursor)
	}
	base += fmt.Sprintf(" ORDER BY %s LIMIT ?", cols[0])
	args = append(args, limit)

	rows, err := q.db.QueryContext(ctx, base, args...)
	if err != nil {
		return iquery.ValuesResult{}, err
	}
	defer rows.Close()

	result := iquery.ValuesResult{Rows: make([][]any, 0, limit)}
	for rows.Next() {
		values := make([]any, len(cols))
		dests := make([]any, len(cols))
		for i := range dests {
			dests[i] = &values[i]
		}
		if err = rows.Scan(dests...); err != nil {
			return iquery.ValuesResult{}, err
		}
		result.Rows = append(result.Rows, values)
	}
	if err = rows.Err(); err != nil {
		return iquery.ValuesResult{}, err
	}

	if len(result.Rows) > 0 {
		result.NextCursor = result.Rows[len(result.Rows)-1][0]
	}
	result.HasMore = len(result.Rows) == limit
	return result, nil
}

func (q *MySQLLookup) Close() error {
	if q.db == nil {
		return nil
	}
	return q.db.Close()
}

func (q *MySQLLookup) queryTableDef(key schema_store.SchemaKey) (*mysql_schema.Table, error) {
	s, err := q.schema.GetSchema(key)
	if err != nil {
		return nil, err
	}
	tableDef, ok := s.(*mysql_schema.Table)
	if !ok {
		return nil, fmt.Errorf("invalid table definition type %T", s)
	}
	return tableDef, nil
}

func (q *MySQLLookup) queryMinMaxCount(ctx context.Context, table *mysql_schema.Table, field string) (any, any, int64, error) {
	query := fmt.Sprintf("SELECT MIN(%s), MAX(%s), COUNT(%s) FROM %s.%s",
		quoteIdentifier(field),
		quoteIdentifier(field),
		quoteIdentifier(field),
		quoteIdentifier(table.Database),
		quoteIdentifier(table.Table),
	)
	var minV any
	var maxV any
	var count int64
	if err := q.db.QueryRowContext(ctx, query).Scan(&minV, &maxV, &count); err != nil {
		return nil, nil, 0, err
	}
	return minV, maxV, count, nil
}

func quoteIdentifier(name string) string {
	return fmt.Sprintf("`%s`", strings.ReplaceAll(name, "`", "``"))
}
