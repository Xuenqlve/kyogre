package lookup

import (
	"context"
	"fmt"
	"sync"

	"github.com/mitchellh/mapstructure"

	"github.com/xuenqlve/kyogre/internal/plugin/iquery"
)

const (
	Memory iquery.LookupType = "memory"
)

func init() {
	iquery.RegisterIQuery(Memory, &MemoryLookup{}, false)
}

type MemoryConfig struct {
	Tables map[string]MemoryTableConfig `mapstructure:"tables" json:"tables"`
}

type MemoryTableConfig struct {
	RowCount int64                        `mapstructure:"row_count" json:"row_count"`
	Fields   map[string]MemoryFieldConfig `mapstructure:"fields" json:"fields"`
}

type MemoryFieldConfig struct {
	Type     string `mapstructure:"type" json:"type"`
	MinValue any    `mapstructure:"min_value" json:"min_value"`
	MaxValue any    `mapstructure:"max_value" json:"max_value"`
}

type MemoryLookup struct {
	pipeline string
	cfg      *MemoryConfig

	mu   sync.Mutex
	data map[string]*tableData
}

func (q *MemoryLookup) Configure(pipeline string, data map[string]any) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	q.pipeline = pipeline
	cfg := &MemoryConfig{}
	if len(data) > 0 {
		if err := mapstructure.Decode(data, cfg); err != nil {
			return err
		}
	}
	if cfg.Tables == nil {
		cfg.Tables = make(map[string]MemoryTableConfig)
	}

	q.cfg = cfg
	q.data = make(map[string]*tableData, len(cfg.Tables))
	for key, table := range cfg.Tables {
		td := &tableData{
			RowCount: table.RowCount,
			Fields:   make(map[string]*fieldData, len(table.Fields)),
		}
		for column, field := range table.Fields {
			td.Fields[column] = &fieldData{
				Type:     field.Type,
				MinValue: field.MinValue,
				MaxValue: field.MaxValue,
			}
		}
		q.data[key] = td
	}
	return nil
}

func (q *MemoryLookup) Lookup(ctx context.Context, req iquery.LookupRequest) (iquery.LookupResult, error) {
	q.mu.Lock()
	defer q.mu.Unlock()

	if req.Schema == nil {
		return iquery.LookupResult{}, fmt.Errorf("lookup schema is nil")
	}

	td := q.getOrCreateTableData(req.Schema.UniqueID())
	bounds := make([]iquery.Bound, 0, len(req.Params))
	for _, param := range req.Params {
		if param.Column == "" {
			continue
		}
		field := td.getOrCreateField(param.Column, param.Type)
		bounds = append(bounds, iquery.Bound{
			BoundParam: iquery.BoundParam{
				Column: param.Column,
				Type:   firstNonEmpty(param.Type, field.Type),
			},
			MinValue: field.MinValue,
			MaxValue: field.MaxValue,
			Count:    int(td.RowCount),
		})
	}

	return iquery.LookupResult{Bounds: bounds}, nil
}

func (q *MemoryLookup) Close() error {
	q.mu.Lock()
	defer q.mu.Unlock()

	q.data = nil
	q.cfg = nil
	return nil
}

func (q *MemoryLookup) getOrCreateTableData(key string) *tableData {
	if q.data == nil {
		q.data = make(map[string]*tableData)
	}
	if td, ok := q.data[key]; ok {
		return td
	}
	td := &tableData{
		Fields: make(map[string]*fieldData),
	}
	q.data[key] = td
	return td
}

type tableData struct {
	RowCount int64
	Fields   map[string]*fieldData
}

type fieldData struct {
	Type     string
	MinValue any
	MaxValue any
}

func (td *tableData) getOrCreateField(column, fieldType string) *fieldData {
	if td.Fields == nil {
		td.Fields = make(map[string]*fieldData)
	}
	if field, ok := td.Fields[column]; ok {
		if field.Type == "" {
			field.Type = fieldType
		}
		return field
	}
	field := &fieldData{Type: fieldType}
	td.Fields[column] = field
	return field
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}
