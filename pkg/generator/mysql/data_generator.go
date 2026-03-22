package mysql

import (
	"strings"

	mysql_schema "github.com/xuenqlve/common/relational_database/mysql"
	"github.com/xuenqlve/kyogre/pkg/tool/mock"
)

// DataGenerator 根据 MySQL 列类型生成对应的随机数据。
type DataGenerator struct {
	*mock.DataGenerator
}

// NewDataGenerator 创建新的数据生成器。
func NewDataGenerator() *DataGenerator {
	return &DataGenerator{mock.NewDataGenerator()}
}

// GenerateValue 根据列定义生成对应类型的值。
func (dg *DataGenerator) GenerateValue(col mysql_schema.Column) any {
	if dg == nil {
		return nil
	}
	// 首先检查是否有自定义模板
	if exist, value := dg.TemplateValue(col.Name); exist {
		return value
	}
	// 根据数据类型生成值
	return dg.generateByType(col)
}

// generateByType 根据 MySQL 数据类型生成对应的值。
func (dg *DataGenerator) generateByType(col mysql_schema.Column) any {
	dataType := strings.ToLower(col.DataType)
	switch dataType {
	case "tinyint":
		return dg.GenerateTinyInt(false)
	case "smallint":
		return dg.GenerateSmallInt(false)
	case "int", "integer":
		return dg.GenerateInt(col.IsGenerated)
	case "bigint", "mediumint", "serial":
		return dg.GenerateInteger(false)
	case "decimal", "numeric", "fixed", "float", "double", "real":
		return dg.GenerateFloat(false)
	case "bool", "boolean":
		return dg.GenerateBool()
	case "timestamp", "datetime", "date", "time", "year":
		return dg.GenerateTime(dataType)
	case "text", "longtext", "mediumtext", "tinytext":
		return dg.GenerateRandomLongText()
	case "blob", "longblob", "mediumblob", "tinyblob":
		return dg.GenerateRandomBlob()
	case "varchar", "char", "string", "character":
		return dg.GenerateRandomString()
	case "json", "jsonb":
		return dg.GenerateJSON()
	case "uuid":
		return dg.GenerateUUID()
	case "enum", "set":
		return dg.GenerateEnum()
	default:
		return dg.GenerateRandomString()
	}
}

// GenerateBatch 批量生成指定类型的值。
func (dg *DataGenerator) GenerateBatch(col mysql_schema.Column, count int) []any {
	result := make([]any, count)
	for i := 0; i < count; i++ {
		result[i] = dg.GenerateValue(col)
	}
	return result
}
