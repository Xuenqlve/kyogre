package generator_context

import (
	"fmt"

	mysql_schema "github.com/xuenqlve/common/relational_database/mysql"
)

// MySQL 生成上下文 kind。
const (
	MySQLRow         = "mysql-row"
	MySQLTransaction = "mysql-transaction"
	MySQLDDL         = "mysql-ddl"
)

// MySQLRowSpec 描述单表 DML 的最小生成信息。
type MySQLRowSpec struct {
	Operation string
	Hint      string
	WriteType string
	// Columns 为本次写入/更新涉及的列集合，空表示使用全量列；delete 会忽略。
	Columns   []string
	Schema    *mysql_schema.Table
}

func (s *MySQLRowSpec) Validate() error {
	if s == nil {
		return fmt.Errorf("row spec is nil")
	}
	if s.Operation == "" {
		return fmt.Errorf("row spec operation is empty")
	}
	if s.Schema == nil {
		return fmt.Errorf("row spec schema is nil")
	}
	return nil
}

// MySQLRowContext 生成 mysql-row 的上下文。
type MySQLRowContext struct {
	Spec MySQLRowSpec
	*baseContext
}

func NewMySQLRowContext(spec MySQLRowSpec, opts ...ContextOption) *MySQLRowContext {
	return &MySQLRowContext{
		Spec:        spec,
		baseContext: NewBaseContext(MySQLRow, opts...),
	}
}

func (c *MySQLRowContext) Validate() error {
	if c == nil {
		return fmt.Errorf("context is nil")
	}
	return c.Spec.Validate()
}

// MySQLTransactionContext 生成 mysql-transaction 的上下文（由多个 mysql-row 组成）。
type MySQLTransactionContext struct {
	Rows []MySQLRowSpec
	*baseContext
}

func NewMySQLTransactionContext(rows []MySQLRowSpec, opts ...ContextOption) *MySQLTransactionContext {
	return &MySQLTransactionContext{
		Rows:        rows,
		baseContext: NewBaseContext(MySQLTransaction, opts...),
	}
}

func (c *MySQLTransactionContext) Validate() error {
	if c == nil {
		return fmt.Errorf("context is nil")
	}
	if len(c.Rows) == 0 {
		return fmt.Errorf("transaction rows are empty")
	}
	for i := range c.Rows {
		if err := c.Rows[i].Validate(); err != nil {
			return fmt.Errorf("transaction row[%d] invalid: %w", i, err)
		}
	}
	return nil
}

// MySQLDDLColumn 描述 DDL 变更的列信息。
type MySQLDDLColumn struct {
	Name   string
	Type   string
	Change string
}

// MySQLDDLSpec 描述 DDL 操作的最小生成信息。
type MySQLDDLSpec struct {
	DDLType  string
	Columns  []MySQLDDLColumn
	Schema   *mysql_schema.Table
}

func (s *MySQLDDLSpec) Validate() error {
	if s == nil {
		return fmt.Errorf("ddl spec is nil")
	}
	if s.DDLType == "" {
		return fmt.Errorf("ddl spec type is empty")
	}
	if s.Schema == nil {
		return fmt.Errorf("ddl spec schema is nil")
	}
	return nil
}

// MySQLDDLContext 生成 mysql-ddl 的上下文。
type MySQLDDLContext struct {
	Spec MySQLDDLSpec
	*baseContext
}

func NewMySQLDDLContext(spec MySQLDDLSpec, opts ...ContextOption) *MySQLDDLContext {
	return &MySQLDDLContext{
		Spec:        spec,
		baseContext: NewBaseContext(MySQLDDL, opts...),
	}
}

func (c *MySQLDDLContext) Validate() error {
	if c == nil {
		return fmt.Errorf("context is nil")
	}
	return c.Spec.Validate()
}
