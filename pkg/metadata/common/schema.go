package common

import (
	"fmt"
	"strings"

	mysql_schema "github.com/xuenqlve/common/relational_database/mysql"
	"github.com/xuenqlve/common/schema_store"
)

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
	Timestamp       = "timestamp"
	TimestampUpdate = "timestamp_update"
	Year            = "year"

	Char         = "char"
	Varchar      = "varchar"
	String       = "string"
	VarcharLarge = "varchar_large"
	Text         = "text"
	Mediumtext   = "mediumtext"
	Longtext     = "longtext"
	Blob         = "blob"

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
	case Char, String, Varchar, VarcharLarge:
		return ""
	case Text, Mediumtext, Longtext, Blob:
		return ""

	// 日期时间类型
	case Date:
		return "2000-01-01"
	case Time:
		return "00:00:00"
	case Datetime:
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
		return "bigint"
	case "string", "varchar_large", "varchar_xlarge":
		return "varchar"
	case "bigint_unsigned":
		return "bigint"
	case "timestamp_update":
		return "timestamp"
	case "boolean":
		return "tinyint"
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
