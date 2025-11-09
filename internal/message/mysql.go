package message

import (
	"time"

	"github.com/xuenqlve/common/relational_database/mysql"
)

const (
	MySQLRow         = "mysql-row"
	MySQLTransaction = "mysql-transaction"
	MySQLDDL         = "mysql-ddl"
)

type MySQLDDLMessage struct {
	Metadata
	// DDL 语句
	mysql.DDLStatement
}

// Type 返回消息类型
func (m *MySQLDDLMessage) Type() string {
	return MySQLDDL
}

// StartTime 返回消息的起始时间
func (m *MySQLDDLMessage) StartTime() time.Time {
	return m.Metadata.StartTime
}

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
	Contents []mysql.RowData
}

type Row struct {
}

// write type
var (
	Replace              = "replace"
	Insert               = "insert"
	InsertOnDuplicateKey = "insert_on_duplicate_key"
	InsertIgnore         = "insert_ignore"
	Delete               = "delete"
	Update               = "update"
	UpdateJoin           = "update_join"
)

type Metadata struct {
	Operation string    `json:"operation" c:"operation"`
	Database  string    `json:"database" c:"database"`
	Table     string    `json:"table" c:"table"`
	Hint      string    `json:"hint" c:"hint"`
	WriteType string    `json:"write_type" c:"write_type"`
	StartTime time.Time `json:"start_time" c:"start_time"`
}
