package mysql

import (
	"strings"

	mysql_schema "github.com/xuenqlve/common/relational_database/mysql"
	"github.com/xuenqlve/kyogre/pkg/tool/mock"
)

// DataGenerator 根据 MySQL 列类型生成对应的随机数据
type DataGenerator struct {
	*mock.DataGenerator
}

// ColumnTemplate 列值生成模板，便于自定义

// NewDataGenerator 创建新的数据生成器
func NewDataGenerator() *DataGenerator {
	return &DataGenerator{mock.NewDataGenerator()}
}

// GenerateValue 根据列定义生成对应类型的值
func (dg *DataGenerator) GenerateValue(col mysql_schema.Column) any {
	// 首先检查是否有自定义模板
	if exist, value := dg.TemplateValue(col.Name); exist {
		return value
	}
	// 根据数据类型生成值
	return dg.generateByType(col)
}

// generateByType 根据 MySQL 数据类型生成对应的值
func (dg *DataGenerator) generateByType(col mysql_schema.Column) any {
	dataType := strings.ToLower(col.DataType)
	switch dataType {
	// 整数类型
	case "tinyint":
		return dg.GenerateTinyInt(col.IsUnsigned)
	case "smallint":
		return dg.GenerateSmallInt(col.IsUnsigned)
	case "int", "integer":
		return dg.GenerateInt(col.IsGenerated)
	case "bigint", "mediumint", "serial":
		return dg.GenerateInteger(col.IsUnsigned)
	// 浮点数类型
	case "decimal", "numeric", "fixed", "float", "double", "real":
		return dg.GenerateFloat(col.IsUnsigned)
	// 布尔类型
	case "bool", "boolean":
		return dg.GenerateBool()
	// 时间类型
	case "timestamp", "datetime", "date", "time", "year":
		return dg.GenerateTime(dataType)
	// 文本类型（长文本）
	case "text", "longtext", "mediumtext", "tinytext":
		return dg.GenerateRandomLongText()
	// 二进制类型
	case "blob", "longblob", "mediumblob", "tinyblob":
		return dg.GenerateRandomBlob()
	// 字符串类型
	case "varchar", "char", "string", "character":
		return dg.GenerateRandomString()
	// JSON 类型
	case "json", "jsonb":
		return dg.GenerateJSON()
	// UUID
	case "uuid":
		return dg.GenerateUUID()
	// 枚举和集合类型
	case "enum", "set":
		return dg.GenerateEnum()
	default:
		// 默认作为字符串处理
		return dg.GenerateRandomString()
	}
}

// GenerateBatch 批量生成指定类型的值
func (dg *DataGenerator) GenerateBatch(col mysql_schema.Column, count int) []any {
	result := make([]any, count)
	for i := 0; i < count; i++ {
		result[i] = dg.GenerateValue(col)
	}
	return result
}
