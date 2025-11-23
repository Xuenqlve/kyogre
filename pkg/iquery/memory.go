package iquery

import (
	"context"
	"sync"

	"github.com/mitchellh/mapstructure"
	"github.com/xuenqlve/common/schema_store"
	"github.com/xuenqlve/kyogre/internal/plugin"
)

type MemoryConfig struct {
	DefaultMaxValue int64 `mapstructure:"default-max-value" json:"default-max-value"`
	DefaultRowCount int64 `mapstructure:"default-row-count" json:"default-row-count"`
	Increment       int64 `mapstructure:"increment" json:"increment"`
}

// tableData stores the mock data for a table
type tableData struct {
	RowCount int64            // current row count
	Fields   map[string]int64 // max value per field
}

type MemoryIQuery struct {
	pipeline string
	cfg      *MemoryConfig
	mu       sync.RWMutex
	data     map[string]*tableData // key: schema key string representation
}

func (q *MemoryIQuery) Configure(pipeline string, data map[string]any) (err error) {
	q.pipeline = pipeline
	q.cfg = &MemoryConfig{
		DefaultMaxValue: 0,
		DefaultRowCount: 0,
		Increment:       1,
	}
	if err = mapstructure.Decode(data, q.cfg); err != nil {
		return
	}
	q.data = make(map[string]*tableData)
	return
}

// getOrCreateTableData returns existing table data or creates mock data if not exists
func (q *MemoryIQuery) getOrCreateTableData(key schema_store.SchemaKey) *tableData {
	keyStr := key.UniqueID()

	q.mu.Lock()
	defer q.mu.Unlock()

	if td, ok := q.data[keyStr]; ok {
		return td
	}

	// Create mock data
	td := &tableData{
		RowCount: q.cfg.DefaultRowCount,
		Fields:   make(map[string]int64),
	}
	q.data[keyStr] = td
	return td
}

// getFieldMaxValue returns the max value for a field, creating default if not exists
func (q *MemoryIQuery) getFieldMaxValue(td *tableData, field string) int64 {
	q.mu.Lock()
	defer q.mu.Unlock()

	if maxVal, ok := td.Fields[field]; ok {
		return maxVal
	}
	td.Fields[field] = q.cfg.DefaultMaxValue
	return q.cfg.DefaultMaxValue
}

func (q *MemoryIQuery) QueryMaxValue(ctx context.Context, key schema_store.SchemaKey, field string) (int64, error) {
	td := q.getOrCreateTableData(key)
	maxVal := q.getFieldMaxValue(td, field)

	// Increment the max value after query
	q.mu.Lock()
	td.Fields[field] += q.cfg.Increment
	q.mu.Unlock()

	return maxVal, nil
}

func (q *MemoryIQuery) QueryRowCount(ctx context.Context, key schema_store.SchemaKey) (int64, error) {
	td := q.getOrCreateTableData(key)

	q.mu.Lock()
	rowCount := td.RowCount
	td.RowCount += q.cfg.Increment
	q.mu.Unlock()
	return rowCount, nil
}

func (q *MemoryIQuery) BatchQuery(ctx context.Context, keys []schema_store.SchemaKey) ([]*plugin.QueryResult, error) {
	results := make([]*plugin.QueryResult, 0, len(keys))
	for _, key := range keys {
		// Use default field name for batch query
		result, err := q.GetQueryResult(ctx, key, "id")
		if err != nil {
			return nil, err
		}
		results = append(results, result)
	}
	return results, nil
}

func (q *MemoryIQuery) GetQueryResult(ctx context.Context, key schema_store.SchemaKey, field string) (*plugin.QueryResult, error) {
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

func (q *MemoryIQuery) Close() error {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.data = nil
	return nil
}
