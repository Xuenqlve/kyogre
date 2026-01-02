package iquery

import (
	"context"
	"sync"

	"github.com/xuenqlve/kyogre/internal/plugin/iquery"
)

const (
	Memory iquery.LookupType = "memory"
)

func init() {
	iquery.RegisterIQuery(Memory, &MemoryLookup{}, false)
}

//type MemoryConfig struct {
//	//DefaultMaxValue int64 `mapstructure:"default-max-value" json:"default-max-value"`
//	DefaultRowCount int64 `mapstructure:"default-row-count" json:"default-row-count"`
//	Increment       int64 `mapstructure:"increment" json:"increment"`
//}

// tableData stores the mock data for a table
type tableData struct {
	RowCount int64
	Fields   map[string]any
}

type MemoryLookup struct {
	pipeline string
	//cfg      *MemoryConfig
	mu   sync.Mutex
	data map[string][]iquery.Bound // key: schema key string representation
}

func (q *MemoryLookup) Configure(pipeline string, data map[string]any) error {
	q.pipeline = pipeline
	//q.cfg = &MemoryConfig{
	//	DefaultMaxValue: 0,
	//	DefaultRowCount: 0,
	//	Increment:       1,
	//}
	//if err := mapstructure.Decode(data, q.cfg); err != nil {
	//	return err
	//}
	//if q.cfg.Increment <= 0 {
	//	q.cfg.Increment = 1
	//}
	//q.data = make(map[string]*tableData)
	return nil
}

func (q *MemoryLookup) Lookup(ctx context.Context, req iquery.LookupRequest) (iquery.LookupResult, error) {
	results := iquery.LookupResult{}

	q.mu.Lock()
	defer q.mu.Unlock()

	td := q.getOrCreateTableData(req.Schema.UniqueID())
	for _, f := range req.Params {
		if f.Column == "" {
			continue
		}
		maxVal := q.getFieldMaxValue(td, f.Column)
		// Compose result before mutating, so caller拿到的是当前值
		result := iquery.LookupResult{}

		//results = append(results, iquery.LookupResult{
		//	Schema: item.Schema,
		//	Fields: models.FieldValue{
		//		Column:   f.Column,
		//		Type:     f.Type,
		//		MinValue: nil,
		//		MaxValue: maxVal,
		//	},
		//	Extras: map[string]any{"rows": td.RowCount},
		//})

		// 模拟真实环境中数据增长
		//td.Fields[f.Column] = maxVal + q.cfg.Increment
		//td.RowCount += q.cfg.Increment
	}
	return results, nil
}

func (q *MemoryLookup) Close() error {
	q.mu.Lock()
	defer q.mu.Unlock()
	//q.data = nil
	return nil
}

func (q *MemoryLookup) getOrCreateTableData(key string) *tableData {
	if td, ok := q.data[key]; ok {
		return td
	}
	td := &tableData{
		RowCount: q.cfg.DefaultRowCount,
		Fields:   make(map[string]int64),
	}
	q.data[key] = td
	return td
}

func (q *MemoryLookup) getFieldMaxValue(td *tableData, field string) int64 {
	if val, ok := td.Fields[field]; ok {
		return val
	}
	td.Fields[field] = q.cfg.DefaultMaxValue
	return q.cfg.DefaultMaxValue
}
