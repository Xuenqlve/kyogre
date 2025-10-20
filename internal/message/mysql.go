package message

import (
	"time"

	"github.com/xuenqlve/timburr/pkg/tool/schema_store"
)

const (
	MySQLRow         = "mysql-row"
	MySQLTransaction = "mysql-transaction"
)

type MySQLTransactionMessage struct {
	GTID    string
	SQLRows []SQLRows
}

func (m *MySQLTransactionMessage) Type() string {
	return MySQLTransaction
}

func (m *MySQLTransactionMessage) StartTime() time.Time {
	return m.SQLRows[0].StartTime
}

type MySQLRowMessage struct {
	SQLRows
}

func (m *MySQLRowMessage) Type() string {
	return MySQLRow
}

func (m *MySQLRowMessage) StartTime() time.Time {
	return m.Metadata.StartTime
}

type SQLRows struct {
	Metadata
	Contents []Row
}

type Row struct {
	Key       string         `json:"key" c:"key"`
	Data      map[string]any `json:"data" c:"data"`
	Old       map[string]any `json:"old" c:"old,omitempty"`
	GuideKeys map[string]any `json:"guide_keys" c:"guide_keys"`
}

type defaultStruct struct{}

func (r *Row) IsColumnSetDefault(columnName string) bool {
	data, ok := r.Data[columnName]
	if !ok {
		return false
	}
	_, ok = data.(defaultStruct)
	return ok
}

// write type
const (
	Replace              = "replace"
	Insert               = "insert"
	InsertOnDuplicateKey = "insert_on_duplicate_key"
	InsertIgnore         = "insert_ignore"
)

type Metadata struct {
	Operation schema_store.DML `json:"operation" c:"operation"`
	Database  string           `json:"database" c:"database"`
	Table     string           `json:"table" c:"table"`
	Hint      string           `json:"hint" c:"hint"`
	WriteType string           `json:"write_type" c:"write_type"`
	StartTime time.Time        `json:"start_time" c:"start_time"`
}
