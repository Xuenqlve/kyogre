package mysql

import (
	"fmt"
	"strings"

	"github.com/xuenqlve/common/errors"
	mysql_schema "github.com/xuenqlve/common/relational_database/mysql"
	"github.com/xuenqlve/common/schema_store"
)

// MockLoadSchemaTool 是一个不依赖数据库连接的schema加载工具
// 从内存中的mock数据返回Table对象
type MockLoadSchemaTool struct {
	mockTables map[string]*mysql_schema.Table
}

// LoadSchema 根据SchemaKey从mock数据中返回Table
func (m *MockLoadSchemaTool) LoadSchema(key schema_store.SchemaKey) (any, error) {
	// key 应该是 mysql_schema.Index 类型
	idx, ok := key.(*mysql_schema.Index)
	if !ok {
		return nil, errors.Errorf("invalid schema key type, expected *mysql_schema.Index, got %T", key)
	}

	tableKey := idx.Database + "." + idx.Table
	table, exists := m.mockTables[tableKey]
	if !exists {
		return nil, errors.Errorf("mock table not found: %s", tableKey)
	}

	return table, nil
}

// Close 关闭资源（mock版本不需要实际操作）
func (m *MockLoadSchemaTool) Close() error {
	return nil
}

// Database 数据库配置，key为数据库名称，value为表列表
type Database map[string]Tables

// Tables 表配置列表
type Tables []Table

// Table 表的配置结构
type Table struct {
	Table   string   `mapstructure:"table" json:"table"`
	Columns []Column `mapstructure:"columns" json:"columns"`
	Indexes []Index  `mapstructure:"indexes" json:"indexes"`
}

// Column 列的配置结构
type Column struct {
	Column string `mapstructure:"column" json:"column"`
	Type   string `mapstructure:"type" json:"type"`
}

// Index 索引的配置结构
type Index struct {
	Name      string   `mapstructure:"name" json:"name"`
	Columns   []string `mapstructure:"columns" json:"columns"`
	IsPrimary bool     `mapstructure:"is_primary" json:"is_primary"`
	IsUnique  bool     `mapstructure:"is_unique" json:"is_unique"`
}

// SchemaKeys 从数据库配置中提取所有表的schema keys
func (d Database) SchemaKeys() []schema_store.SchemaKey {
	keys := []schema_store.SchemaKey{}
	for database, tables := range d {
		for _, table := range tables {
			keys = append(keys, &mysql_schema.Index{
				Database: database,
				Table:    table.Table,
			})
		}
	}
	return keys
}

func (d Database) Validate() error {
	if len(d) == 0 {
		return errors.New("database is empty")
	}
	for database, tables := range d {
		if len(tables) == 0 {
			return errors.Errorf("database:%s table is empty", database)
		}
		for _, table := range tables {
			if len(table.Columns) == 0 {
				return errors.Errorf("database:%s table %s has no columns", database, table.Table)
			}
		}
	}
	return nil
}

// CreateTableSQL 生成CREATE TABLE语句
func (t Table) CreateTableSQL(database string) string {
	var parts []string

	// 1. 生成列定义
	for _, col := range t.Columns {
		parts = append(parts, col.ColumnDefinition())
	}

	// 2. 添加约束 (主键、唯一索引、普通索引)
	for _, idx := range t.Indexes {
		parts = append(parts, idx.IndexDefinition())
	}

	// 3. 组合为完整的CREATE TABLE语句
	columnDefs := strings.Join(parts, ",\n  ")
	return fmt.Sprintf(
		"CREATE TABLE IF NOT EXISTS `%s`.`%s` (\n  %s\n) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='%s-测试表';",
		database,
		t.Table,
		columnDefs,
		t.Table,
	)
}

const (
	Primary         = "primary"
	Tinyint         = "tinyint"
	Smallint        = "smallint"
	Mediumint       = "mediumint"
	Int             = "int"
	Bigint          = "bigint"
	BigintUnsigned  = "bigint_unsigned"
	Float           = "float"
	Double          = "double"
	Decimal         = "decimal"
	Date            = "date"
	Time            = "time"
	Datetime        = "datetime"
	DatetimeUpdate  = "datetime_update"
	Timestamp       = "timestamp"
	TimestampUpdate = "timestamp_update"
	Year            = "year"

	Char          = "char"
	Varchar       = "varchar"
	String        = "string"
	VarcharLarge  = "varchar_large"
	VarcharXLarge = "varchar_xlarge"
	Text          = "text"
	Mediumtext    = "mediumtext"
	Longtext      = "longtext"
	Blob          = "blob"

	Boolean = "boolean"
	Json    = "json"

	Enum = "enum"
	Set  = "set"
)

// TypeTransform 将用户配置的简化类型转换为完整的MySQL列定义
// 支持常见的数据类型别名，便于配置文件编写
func (c Column) TypeTransform() string {
	switch c.Type {
	// 主键类型
	case Primary:
		return "BIGINT UNSIGNED NOT NULL AUTO_INCREMENT"

	// 整数类型
	case Tinyint:
		return "TINYINT NOT NULL DEFAULT 0"
	case Smallint:
		return "SMALLINT NOT NULL DEFAULT 0"
	case Mediumint:
		return "MEDIUMINT NOT NULL DEFAULT 0"
	case Int:
		return "INT NOT NULL DEFAULT 0"
	case Bigint:
		return "BIGINT NOT NULL DEFAULT 0"
	case BigintUnsigned:
		return "BIGINT UNSIGNED NOT NULL DEFAULT 0"

	// 浮点数类型
	case Float:
		return "FLOAT NOT NULL DEFAULT 0.0"
	case Double:
		return "DOUBLE NOT NULL DEFAULT 0.0"
	case Decimal:
		return "DECIMAL(10,2) NOT NULL DEFAULT 0.00"

	// 字符串类型
	case Char:
		return "CHAR(255) NOT NULL DEFAULT ''"
	case String, Varchar:
		return "VARCHAR(255) NOT NULL DEFAULT ''"
	case VarcharLarge:
		return "VARCHAR(1024) NOT NULL DEFAULT ''"
	case VarcharXLarge:
		return "VARCHAR(5000) NOT NULL DEFAULT ''"
	case Text:
		return "TEXT"
	case Mediumtext:
		return "MEDIUMTEXT"
	case Longtext:
		return "LONGTEXT"
	case Blob:
		return "BLOB"

	// 日期时间类型
	case Date:
		return "DATE NOT NULL DEFAULT '2000-01-01'"
	case Time:
		return "TIME NOT NULL DEFAULT '00:00:00'"
	case Datetime:
		return "DATETIME NOT NULL DEFAULT '2000-01-01 00:00:00'"
	case DatetimeUpdate:
		return "datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP"
	case Timestamp:
		return "TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP"
	case TimestampUpdate:
		return "TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP"
	case Year:
		return "YEAR NOT NULL DEFAULT 2000"

	// 布尔类型（MySQL中使用TINYINT(1)）
	case Boolean:
		return "TINYINT(1) NOT NULL DEFAULT 0"

	// JSON类型
	case Json:
		return "JSON"

	// ENUM类型（默认示例）
	case Enum:
		return "ENUM('active','inactive') NOT NULL DEFAULT 'active'"

	// 如果是未知类型，直接返回原始值（用户可提供完整的MySQL类型定义）
	default:
		return c.Type
	}
}

func (c Column) DefaultVal() string {
	switch c.Type {
	// 主键类型 - 自增，无需默认值
	case Primary:
		return ""

	// 整数类型
	case Tinyint, Smallint, Mediumint, Int, Bigint, BigintUnsigned, Boolean:
		return "0"
	case Year:
		return "2000"

	// 浮点数类型
	case Float, Double:
		return "0.0"
	case Decimal:
		return "0.00"

	// 字符串类型
	case Char, String, Varchar, VarcharLarge, VarcharXLarge:
		return ""
	case Text, Mediumtext, Longtext, Blob:
		return ""

	// 日期时间类型
	case Date:
		return "2000-01-01"
	case Time:
		return "00:00:00"
	case Datetime, DatetimeUpdate:
		return "2000-01-01 00:00:00"
	case Timestamp, TimestampUpdate:
		return "" // CURRENT_TIMESTAMP 自动设置
	// JSON 类型
	case Json:
		return "{}"

	// ENUM 类型
	case Enum:
		return "active"

	// SET 类型
	case Set:
		return ""

	// 默认返回空值
	default:
		return ""
	}
}

func (c Column) DataType() string {
	switch c.Type {
	// 主键类型
	case Primary:
		return Bigint
	case String, VarcharLarge:
		return Varchar
	case BigintUnsigned:
		return Bigint
	case DatetimeUpdate:
		return Datetime
	case TimestampUpdate:
		return Timestamp
	case Boolean:
		return Tinyint
	default:
		return c.Type
	}
}

// ColumnDefinition 生成列定义
func (c Column) ColumnDefinition() string {
	def := fmt.Sprintf("`%s` %s", c.Column, c.TypeTransform())
	comment := c.Column
	def += " COMMENT '" + comment + "'"
	return def
}

// IndexDefinition 生成索引定义
func (i Index) IndexDefinition() string {
	columnList := fmt.Sprintf("`%s`", strings.Join(i.Columns, "`,`"))

	if i.IsPrimary {
		return fmt.Sprintf("PRIMARY KEY (%s)", columnList)
	}

	if i.IsUnique {
		return fmt.Sprintf("UNIQUE KEY `%s` (%s)", i.Name, columnList)
	}

	return fmt.Sprintf("KEY `%s` (%s)", i.Name, columnList)
}
