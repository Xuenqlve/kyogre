package lookup

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"strings"
	"sync"

	"github.com/mitchellh/mapstructure"
	mysql_schema "github.com/xuenqlve/common/relational_database/mysql"
	"github.com/xuenqlve/common/schema_store"
	sql_tool "github.com/xuenqlve/common/sql"
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
	mu       sync.Mutex
	// maxBySchemaColumn caches observed max values to avoid repeated max queries.
	maxBySchemaColumn map[string]map[string]int64
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
	q.maxBySchemaColumn = make(map[string]map[string]int64)
	return nil
}

func (q *MySQLLookup) LookupRange(ctx context.Context, req iquery.RangeRequest) (iquery.RangeResult, error) {
	tableDef, err := q.queryTableDef(req.Schema)
	if err != nil {
		return iquery.RangeResult{}, err
	}
	if len(req.Columns) == 0 || req.Columns[0].Column == "" {
		return iquery.RangeResult{}, fmt.Errorf("lookup params empty")
	}

	need := req.Need
	if need <= 0 {
		need = 1
	}
	field := req.Columns[0].Column
	wrapAt := q.columnMaxID(tableDef, req.Columns[0])
	maxCached := q.getCachedMax(req.Schema.UniqueID(), field)
	if maxCached > 0 && req.Cursor+need-1 <= maxCached {
		start := req.Cursor + 1
		end := start + need - 1
		if end > maxCached {
			end = maxCached
		}
		if end < start {
			return iquery.RangeResult{}, fmt.Errorf("lookup returned invalid window for %s", req.Schema.UniqueID())
		}
		return iquery.RangeResult{Window: range_pool.IntRange{Start: start, End: end}}, nil
	}

	if wrapAt > 0 && req.Cursor >= wrapAt {
		minV, maxV, cnt, err := q.queryMinMaxCount(ctx, tableDef, field)
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
		if maxID < wrapAt {
			return iquery.RangeResult{}, fmt.Errorf("lookup range exhausted for %s: start=%d wrapAt=%d", req.Schema.UniqueID(), req.Cursor+1, wrapAt)
		}
		q.setCachedMax(req.Schema.UniqueID(), field, maxID)
		start := minID
		end := start + need - 1
		if end > maxID {
			end = maxID
		}
		if end < start {
			return iquery.RangeResult{}, fmt.Errorf("lookup returned invalid window for %s", req.Schema.UniqueID())
		}
		return iquery.RangeResult{
			EnableLoop: true,
			Window:     range_pool.IntRange{Start: start, End: end},
		}, nil
	}

	minV, maxV, cnt, err := q.queryMinMaxCountAfter(ctx, tableDef, field, req.Cursor)
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
	q.setCachedMax(req.Schema.UniqueID(), field, maxID)
	if minID > maxID {
		return iquery.RangeResult{}, fmt.Errorf("lookup returned invalid window for %s", req.Schema.UniqueID())
	}
	start := minID
	end := start + need - 1
	if end > maxID {
		end = maxID
	}
	if end < start {
		return iquery.RangeResult{}, fmt.Errorf("lookup returned invalid window for %s", req.Schema.UniqueID())
	}
	return iquery.RangeResult{Window: range_pool.IntRange{Start: start, End: end}}, nil
}

func (q *MySQLLookup) ScanValues(ctx context.Context, req iquery.ValueRequest) (iquery.ValuesResult, error) {
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
	rawCols := make([]string, 0, len(req.Columns))
	for _, c := range req.Columns {
		if c.Column == "" {
			continue
		}
		rawCols = append(rawCols, c.Column)
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
		cursorMap := req.Cursor
		if len(rawCols) == 1 {
			base += fmt.Sprintf(" WHERE %s > ?", cols[0])
			val, ok := cursorMap[rawCols[0]]
			if !ok {
				return iquery.ValuesResult{}, fmt.Errorf("cursor missing column %q", rawCols[0])
			}
			args = append(args, val)
		} else {
			cursorVals := make([]any, 0, len(rawCols))
			for _, col := range rawCols {
				val, ok := cursorMap[col]
				if !ok {
					return iquery.ValuesResult{}, fmt.Errorf("cursor missing column %q", col)
				}
				cursorVals = append(cursorVals, val)
			}
			parts := make([]string, 0, len(rawCols))
			for i := range rawCols {
				sub := make([]string, 0, i+1)
				for j := 0; j < i; j++ {
					sub = append(sub, fmt.Sprintf("%s = ?", cols[j]))
					args = append(args, cursorVals[j])
				}
				sub = append(sub, fmt.Sprintf("%s > ?", cols[i]))
				args = append(args, cursorVals[i])
				parts = append(parts, fmt.Sprintf("(%s)", strings.Join(sub, " AND ")))
			}
			base += fmt.Sprintf(" WHERE %s", strings.Join(parts, " OR "))
		}
	}
	base += fmt.Sprintf(" ORDER BY %s LIMIT ?", strings.Join(cols, ","))
	args = append(args, limit)
	result := iquery.ValuesResult{Rows: make([]map[string]any, 0, limit)}
	data, err := sql_tool.Query(ctx, q.db, base, args...)
	if err != nil {
		return iquery.ValuesResult{}, err
	}
	result.Rows = data
	if len(data) > 0 {
		result.NextCursor = data[len(data)-1]
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

func (q *MySQLLookup) queryMinMaxCountAfter(ctx context.Context, table *mysql_schema.Table, field string, cursor int64) (any, any, int64, error) {
	query := fmt.Sprintf("SELECT MIN(%s), MAX(%s), COUNT(%s) FROM %s.%s WHERE %s > ?",
		quoteIdentifier(field),
		quoteIdentifier(field),
		quoteIdentifier(field),
		quoteIdentifier(table.Database),
		quoteIdentifier(table.Table),
		quoteIdentifier(field),
	)
	var minV any
	var maxV any
	var count int64
	if err := q.db.QueryRowContext(ctx, query, cursor).Scan(&minV, &maxV, &count); err != nil {
		return nil, nil, 0, err
	}
	return minV, maxV, count, nil
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

func (q *MySQLLookup) getCachedMax(schemaID, field string) int64 {
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.maxBySchemaColumn == nil {
		return 0
	}
	cols := q.maxBySchemaColumn[schemaID]
	if cols == nil {
		return 0
	}
	return cols[field]
}

func (q *MySQLLookup) setCachedMax(schemaID, field string, maxID int64) {
	if maxID <= 0 {
		return
	}
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.maxBySchemaColumn == nil {
		q.maxBySchemaColumn = make(map[string]map[string]int64)
	}
	cols := q.maxBySchemaColumn[schemaID]
	if cols == nil {
		cols = make(map[string]int64)
		q.maxBySchemaColumn[schemaID] = cols
	}
	if maxID > cols[field] {
		cols[field] = maxID
	}
}

func (q *MySQLLookup) columnMaxID(table *mysql_schema.Table, param iquery.ColumnParam) int64 {
	if table == nil || param.Column == "" {
		return hardMaxInt64ByType(param.Type)
	}
	col, ok := table.Column(param.Column)
	if !ok {
		return hardMaxInt64ByType(param.Type)
	}
	typ := strings.TrimSpace(col.RawType)
	if typ == "" {
		typ = strings.TrimSpace(col.DataType)
	}
	if col.IsUnsigned && typ != "" {
		lower := strings.ToLower(typ)
		if !strings.Contains(lower, "unsigned") {
			typ = typ + " unsigned"
		}
	}
	return hardMaxInt64ByType(typ)
}

func hardMaxInt64ByType(typ string) int64 {
	if typ == "" {
		return math.MaxInt64
	}
	t := strings.ToLower(strings.TrimSpace(typ))
	switch {
	case strings.Contains(t, "bigint unsigned"):
		return math.MaxInt64 // cannot represent full uint64 in int64, cap for safety
	case strings.Contains(t, "bigint"):
		return math.MaxInt64
	case strings.Contains(t, "int unsigned"):
		return int64(^uint32(0))
	case strings.Contains(t, "int"):
		return math.MaxInt32
	case strings.Contains(t, "smallint unsigned"):
		return int64(^uint16(0))
	case strings.Contains(t, "smallint"):
		return math.MaxInt16
	case strings.Contains(t, "tinyint unsigned"):
		return int64(^uint8(0))
	case strings.Contains(t, "tinyint"):
		return math.MaxInt8
	default:
		return math.MaxInt64
	}
}
