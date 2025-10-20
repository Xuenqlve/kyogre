package gh_ost

import "time"

const (
	// DDLMessageType DDL 消息类型标识
	DDLMessageType = "ddl"
)

// DDLMessage 表示 DDL 操作的消息
type DDLMessage struct {
	// 数据库名
	Database string
	// 表名
	Table string
	// DDL 语句
	SQL string
	// 操作类型（ALTER TABLE, CREATE INDEX, DROP INDEX 等）
	Operation string
	// 消息发起时间
	StartTimeValue time.Time
}

// Type 返回消息类型
func (m *DDLMessage) Type() string {
	return DDLMessageType
}

// StartTime 返回消息的起始时间
func (m *DDLMessage) StartTime() time.Time {
	return m.StartTimeValue
}
