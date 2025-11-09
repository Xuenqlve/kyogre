package mysql_config

import (
	"fmt"
	"strings"

	mysql_schema "github.com/xuenqlve/common/relational_database/mysql"
	"github.com/xuenqlve/common/schema_store"
)

type Config struct {
	DataSource string   `mapstructure:"data-source" json:"data-source"`
	Databases  Database `mapstructure:"database" json:"database"`
}

type Database map[string]Tables

type Tables []Table

type Table struct {
	Name    string   `mapstructure:"name" json:"name"`
	Columns []Column `mapstructure:"columns" json:"columns"`
	Indexes []Index  `mapstructure:"indexes" json:"indexes"`
}

type Column struct {
	Name string `mapstructure:"name" json:"name"`
	Type string `mapstructure:"type" json:"type"`
}

type Index struct {
	Name      string   `mapstructure:"name" json:"name"`
	Columns   []string `mapstructure:"columns" json:"columns"`
	IsPrimary bool     `mapstructure:"is_primary" json:"is_primary"`
	IsUnique  bool     `mapstructure:"is_unique" json:"is_unique"`
}

func (c *Config) CreateTableSQLs() []string {
	sqlSet := []string{}
	for database, tables := range c.Databases {
		sqlSet = append(sqlSet, "CREATE DATABASE IF NOT EXISTS "+database)
		for _, table := range tables {
			sqlSet = append(sqlSet, table.CreateTableSQL(database))
		}
	}
	return sqlSet
}

func (c *Config) SchemaKeys() []schema_store.SchemaKey {
	keys := []schema_store.SchemaKey{}
	for database, tables := range c.Databases {
		for _, table := range tables {
			keys = append(keys, &mysql_schema.Index{
				Database: database,
				Table:    table.Name,
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
		t.Name,
		columnDefs,
		t.Name,
	)
}

// TypeTransform 将用户配置的简化类型转换为完整的MySQL列定义
// 支持常见的数据类型别名，便于配置文件编写
func (c Column) TypeTransform() string {
	switch c.Type {
	// 主键类型
	case "primary":
		return "BIGINT UNSIGNED NOT NULL AUTO_INCREMENT"

	// 整数类型
	case "tinyint":
		return "TINYINT NOT NULL DEFAULT 0"
	case "smallint":
		return "SMALLINT NOT NULL DEFAULT 0"
	case "mediumint":
		return "MEDIUMINT NOT NULL DEFAULT 0"
	case "int":
		return "INT NOT NULL DEFAULT 0"
	case "bigint":
		return "BIGINT NOT NULL DEFAULT 0"
	case "bigint_unsigned":
		return "BIGINT UNSIGNED NOT NULL DEFAULT 0"

	// 浮点数类型
	case "float":
		return "FLOAT NOT NULL DEFAULT 0.0"
	case "double":
		return "DOUBLE NOT NULL DEFAULT 0.0"
	case "decimal":
		return "DECIMAL(10,2) NOT NULL DEFAULT 0.00"

	// 字符串类型
	case "char":
		return "CHAR(255) NOT NULL DEFAULT ''"
	case "string", "varchar":
		return "VARCHAR(255) NOT NULL DEFAULT ''"
	case "varchar_large":
		return "VARCHAR(1024) NOT NULL DEFAULT ''"
	case "varchar_xlarge":
		return "VARCHAR(5000) NOT NULL DEFAULT ''"
	case "text":
		return "TEXT"
	case "mediumtext":
		return "MEDIUMTEXT"
	case "longtext":
		return "LONGTEXT"
	case "blob":
		return "BLOB"

	// 日期时间类型
	case "date":
		return "DATE NOT NULL DEFAULT '2000-01-01'"
	case "time":
		return "TIME NOT NULL DEFAULT '00:00:00'"
	case "datetime":
		return "DATETIME NOT NULL DEFAULT '2000-01-01 00:00:00'"
	case "timestamp":
		return "TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP"
	case "timestamp_update":
		return "TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP"
	case "year":
		return "YEAR NOT NULL DEFAULT 2000"

	// 布尔类型（MySQL中使用TINYINT(1)）
	case "boolean":
		return "TINYINT(1) NOT NULL DEFAULT 0"

	// JSON类型
	case "json":
		return "JSON"

	// ENUM类型（默认示例）
	case "enum":
		return "ENUM('active','inactive') NOT NULL DEFAULT 'active'"

	// 如果是未知类型，直接返回原始值（用户可提供完整的MySQL类型定义）
	default:
		return c.Type
	}
}

// ColumnDefinition 生成列定义
func (c Column) ColumnDefinition() string {
	def := fmt.Sprintf("`%s` %s", c.Name, c.TypeTransform())
	def += " COMMENT '" + c.Name + "'"
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
