package iquery

import (
	"github.com/xuenqlve/common/schema_store"
)

// SequenceConfig 描述分段生成数据所需的序列参数
type SequenceConfig struct {
    Enabled      bool                   `json:"enabled"`
    Key          string                 `json:"key,omitempty"`
    Field        string                 `json:"field,omitempty"`
    StartValue   int64                  `json:"start_value,omitempty"`
    EndValue     int64                  `json:"end_value,omitempty"`
    CurrentValue int64                  `json:"current_value,omitempty"`
    Step         int64                  `json:"step,omitempty"`
    Width        int64                  `json:"width,omitempty"`
    Schema       schema_store.SchemaKey `json:"-"`
}
