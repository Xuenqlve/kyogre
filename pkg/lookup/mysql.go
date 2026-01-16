package lookup

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"

	"github.com/mitchellh/mapstructure"
	mysql_schema "github.com/xuenqlve/common/relational_database/mysql"
	"github.com/xuenqlve/common/schema_store"
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

func (q *MySQLLookup) LookupBounds(ctx context.Context, req iquery.LookupRequest) (iquery.LookupResult, error) {
	tableDef, err := q.queryTableDef(req.Schema)
	if err != nil {
		return iquery.LookupResult{}, err
	}
	if len(req.Params) == 0 || req.Params[0].Column == "" {
		return iquery.LookupResult{}, fmt.Errorf("lookup params empty")
	}

	partition := req.Partition
	if partition == "" {
		partition = range_pool.RangePoolLiveName
	}
	need := req.Need
	if need <= 0 {
		need = 1
	}

	minV, maxV, cnt, err := q.queryMinMaxCount(ctx, tableDef, req.Params[0].Column)
	if err != nil {
		return iquery.LookupResult{}, err
	}

	if cnt == 0 || maxV == nil {
		if partition == range_pool.RangePoolFreeName {
			start := q.cfg.FreeMinID
			end := start + need - 1
			if req.WrapAt > 0 && end > req.WrapAt {
				end = req.WrapAt
			}
			if end < start {
				return iquery.LookupResult{}, fmt.Errorf("lookup returned invalid window for %s", req.Schema.UniqueID())
			}
			return iquery.LookupResult{Window: range_pool.IntRange{Start: start, End: end}}, nil
		}
		return iquery.LookupResult{}, fmt.Errorf("empty live range for %s", req.Schema.UniqueID())
	}

	minID, err := toInt64(minV)
	if err != nil {
		return iquery.LookupResult{}, err
	}
	maxID, err := toInt64(maxV)
	if err != nil {
		return iquery.LookupResult{}, err
	}

	if partition == range_pool.RangePoolFreeName {
		if minID <= 0 {
			minID = q.cfg.FreeMinID
		}
		enableLoop := req.WrapAt > 0 && maxID >= req.WrapAt
		if enableLoop {
			start := minID
			end := start + need - 1
			if req.WrapAt > 0 && end > req.WrapAt {
				end = req.WrapAt
			}
			if end < start {
				return iquery.LookupResult{}, fmt.Errorf("lookup returned invalid window for %s", req.Schema.UniqueID())
			}
			return iquery.LookupResult{
				EnableLoop: true,
				Window:     range_pool.IntRange{Start: start, End: end},
			}, nil
		}

		start := maxID + 1
		if req.WrapAt > 0 && start > req.WrapAt {
			return iquery.LookupResult{}, fmt.Errorf("insert range exhausted for %s: start=%d wrapAt=%d", req.Schema.UniqueID(), start, req.WrapAt)
		}
		end := start + need - 1
		if req.WrapAt > 0 && end > req.WrapAt {
			end = req.WrapAt
		}
		if end < start {
			return iquery.LookupResult{}, fmt.Errorf("lookup returned invalid window for %s", req.Schema.UniqueID())
		}
		return iquery.LookupResult{Window: range_pool.IntRange{Start: start, End: end}}, nil
	}

	if minID > maxID {
		return iquery.LookupResult{}, fmt.Errorf("lookup returned invalid window for %s", req.Schema.UniqueID())
	}
	return iquery.LookupResult{Window: range_pool.IntRange{Start: minID, End: maxID}}, nil
}

func (q *MySQLLookup) ScanValues(ctx context.Context, req iquery.ValuesRequest) (iquery.ValuesResult, error) {
	tableDef, err := q.queryTableDef(req.Schema)
	if err != nil {
		return iquery.ValuesResult{}, err
	}
	if len(req.Columns) == 0 {
		return iquery.ValuesResult{}, nil
	}
	limit := req.Limit
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

func toInt64(v any) (int64, error) {
	switch n := v.(type) {
	case nil:
		return 0, nil
	case int:
		return int64(n), nil
	case int8:
		return int64(n), nil
	case int16:
		return int64(n), nil
	case int32:
		return int64(n), nil
	case int64:
		return n, nil
	case uint:
		return int64(n), nil
	case uint8:
		return int64(n), nil
	case uint16:
		return int64(n), nil
	case uint32:
		return int64(n), nil
	case uint64:
		if n > uint64(^uint64(0)>>1) {
			return 0, fmt.Errorf("uint64 overflow: %d", n)
		}
		return int64(n), nil
	case float32:
		return int64(n), nil
	case float64:
		return int64(n), nil
	case []byte:
		return toInt64(string(n))
	case string:
		v, err := strconv.ParseInt(strings.TrimSpace(n), 10, 64)
		if err != nil {
			return 0, fmt.Errorf("string cannot convert to int64: %q", n)
		}
		return v, nil
	case sql.NullInt64:
		if !n.Valid {
			return 0, nil
		}
		return n.Int64, nil
	default:
		return 0, fmt.Errorf("unsupported type %T to int64", v)
	}
}
