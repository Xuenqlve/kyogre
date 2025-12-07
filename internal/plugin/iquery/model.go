package iquery

import (
	"github.com/xuenqlve/common/schema_store"
)

// SequenceSegment 分段信息
type SequenceSegment struct {
	Ranges []*SegmentRange
}

type SegmentRange struct {
	WorkerID   int
	StartValue int64
	EndValue   int64
}

type SequenceConfig struct {
	Enabled      bool   `json:"enabled"`
	Field        string `json:"field"` // 递增字段
	StartValue   int64  `json:"start_value"`
	EndValue     int64  `json:"end_value"`
	CurrentValue int64  `json:"current_value"` // 由Worker维护
	Step         int64  `json:"step"`          // 递增步长，默认1
}

type QueryResult struct {
	TableKey        schema_store.SchemaKey // 表的唯一标识符 "db.table"
	Field           string                 // 反查的字段
	MaxValue        int64                  // 字段的最大值
	CurrentRowCount int64                  // 表的当前行数
	Metadata        map[string]any         // 其他元数据
}
