package common

import (
	"fmt"
	"math/rand/v2"
	"strings"
	"time"

	mysql_schema "github.com/xuenqlve/common/relational_database/mysql"
)

// DataGenerator 根据 MySQL 列类型生成对应的随机数据
type DataGenerator struct {
	seed      uint64
	randSrc   rand.Source
	rnd       *rand.Rand
	templates map[string]ColumnTemplate
}

// ColumnTemplate 列值生成模板，便于自定义
type ColumnTemplate struct {
	Name      string
	Generator func() any
}

// NewDataGenerator 创建新的数据生成器
func NewDataGenerator() *DataGenerator {
	dg := &DataGenerator{
		templates: make(map[string]ColumnTemplate),
	}
	// 初始化默认的随机数生成器
	dg.SetSeed(uint64(time.Now().UnixNano()))
	// 注册所有内置模板
	RegisterAllBuiltinTemplates(dg)
	return dg
}

// SetSeed 设置随机数种子（便于测试复现）
func (dg *DataGenerator) SetSeed(seed uint64) {
	dg.seed = seed
	dg.randSrc = rand.NewPCG(seed, seed)
	dg.rnd = rand.New(dg.randSrc)
}

// getRand 获取随机数生成器，如果未初始化则初始化
func (dg *DataGenerator) getRand() *rand.Rand {
	if dg.rnd == nil {
		dg.SetSeed(uint64(time.Now().UnixNano()))
	}
	return dg.rnd
}

// GenerateValue 根据列定义生成对应类型的值
func (dg *DataGenerator) GenerateValue(col mysql_schema.Column) any {
	// 首先检查是否有自定义模板
	if template, exists := dg.templates[col.Name]; exists {
		return template.Generator()
	}

	// 根据数据类型生成值
	return dg.generateByType(col)
}

// RegisterTemplate 为特定列名注册自定义生成模板
// 例：注册"email"列的生成器
func (dg *DataGenerator) RegisterTemplate(columnName string, generator func() any) {
	dg.templates[columnName] = ColumnTemplate{
		Name:      columnName,
		Generator: generator,
	}
}

// RegisterTemplateByType 为特定数据类型注册生成模板
func (dg *DataGenerator) RegisterTemplateByType(dataType string, generator func() any) {
	dg.templates["__type__:"+dataType] = ColumnTemplate{
		Name:      dataType,
		Generator: generator,
	}
}

// generateByType 根据 MySQL 数据类型生成对应的值
func (dg *DataGenerator) generateByType(col mysql_schema.Column) any {
	dataType := strings.ToLower(col.DataType)

	switch dataType {
	// 整数类型
	case "tinyint":
		return dg.generateTinyInt(col)
	case "smallint":
		return dg.generateSmallInt(col)
	case "int", "integer":
		return dg.generateInt(col)
	case "bigint", "mediumint", "serial":
		return dg.generateInteger(col)

	// 浮点数类型
	case "decimal", "numeric", "fixed", "float", "double", "real":
		return dg.generateFloat(col)

	// 布尔类型
	case "bool", "boolean":
		return dg.getRand().IntN(2) == 0

	// 时间类型
	case "timestamp", "datetime", "date", "time", "year":
		return dg.generateTime(dataType)

	// 文本类型（长文本）
	case "text", "longtext", "mediumtext", "tinytext":
		return dg.generateLongText()

	// 二进制类型
	case "blob", "longblob", "mediumblob", "tinyblob":
		return dg.generateBlob()

	// 字符串类型
	case "varchar", "char", "string", "character":
		return dg.generateString()

	// JSON 类型
	case "json", "jsonb":
		return dg.generateJSON()

	// UUID
	case "uuid":
		return dg.generateUUID()

	// 枚举和集合类型
	case "enum", "set":
		return dg.generateEnum(col)

	default:
		// 默认作为字符串处理
		return dg.generateString()
	}
}

// generateTinyInt 生成 TINYINT 值 (-128 to 127 or 0 to 255)
func (dg *DataGenerator) generateTinyInt(col mysql_schema.Column) int64 {
	if col.IsUnsigned {
		return int64(dg.getRand().IntN(256)) // 0-255
	}
	return int64(dg.getRand().IntN(256) - 128) // -128 to 127
}

// generateSmallInt 生成 SMALLINT 值 (-32768 to 32767 or 0 to 65535)
func (dg *DataGenerator) generateSmallInt(col mysql_schema.Column) int64 {
	if col.IsUnsigned {
		return int64(dg.getRand().IntN(65536)) // 0-65535
	}
	return int64(dg.getRand().IntN(65536) - 32768) // -32768 to 32767
}

// generateInt 生成 INT 值 (-2147483648 to 2147483647 or 0 to 4294967295)
func (dg *DataGenerator) generateInt(col mysql_schema.Column) int64 {
	if col.IsUnsigned {
		return int64(dg.getRand().IntN(100000000)) // 0到一亿
	}
	return int64(dg.getRand().IntN(100000000) - 50000000) // -5千万 到 5千万
}

// generateInteger 生成 BIGINT 值 (-9223372036854775808 to 9223372036854775807)
func (dg *DataGenerator) generateInteger(col mysql_schema.Column) int64 {
	if col.IsUnsigned {
		return dg.getRand().Int64() & 0x7FFFFFFFFFFFFFFF // 正数
	}
	return dg.getRand().Int64() // 任意 64 位整数
}

// generateFloat 生成浮点数值
func (dg *DataGenerator) generateFloat(col mysql_schema.Column) float64 {
	val := dg.getRand().Float64() * 1000000
	if !col.IsUnsigned && dg.getRand().IntN(2) == 0 {
		val = -val
	}
	return val
}

// generateString 生成随机字符串
func (dg *DataGenerator) generateString() string {
	return fmt.Sprintf("val_%d", dg.getRand().IntN(100000))
}

// generateLongText 生成长文本
func (dg *DataGenerator) generateLongText() string {
	return fmt.Sprintf("text_%d_%d_%d", dg.getRand().IntN(10000), dg.getRand().IntN(10000), dg.getRand().IntN(10000))
}

// generateBlob 生成 BLOB 数据（返回十六进制字符串表示）
func (dg *DataGenerator) generateBlob() string {
	return fmt.Sprintf("blob_%x_%x", dg.getRand().Int64(), dg.getRand().Int64())
}

// generateJSON 生成 JSON 对象
func (dg *DataGenerator) generateJSON() string {
	return fmt.Sprintf(`{"id":%d,"name":"user_%d","timestamp":"%s"}`,
		dg.getRand().IntN(1000000),
		dg.getRand().IntN(100000),
		time.Now().Format(time.RFC3339))
}

// generateUUID 生成 UUID 格式字符串
func (dg *DataGenerator) generateUUID() string {
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		dg.getRand().IntN(0xffffffff),
		dg.getRand().IntN(0xffff),
		dg.getRand().IntN(0xffff),
		dg.getRand().IntN(0xffff),
		dg.getRand().Int64()&0xffffffffffff)
}

// generateTime 生成时间值
func (dg *DataGenerator) generateTime(timeType string) any {
	now := time.Now()

	switch timeType {
	case "timestamp", "datetime":
		// 返回当前时间的字符串格式
		return now.Format("2006-01-02 15:04:05")

	case "date":
		return now.Format("2006-01-02")

	case "time":
		return now.Format("15:04:05")

	case "year":
		return now.Year()

	default:
		return now
	}
}

// generateEnum 生成枚举值
func (dg *DataGenerator) generateEnum(col mysql_schema.Column) string {
	// 提取枚举值列表
	// ENUM 类型的 RawType 格式通常是: "enum('val1','val2','val3')"
	// 这里简化处理，返回一个占位符
	return fmt.Sprintf("enum_val_%d", dg.getRand().IntN(5))
}

// initDefaultTemplates 初始化默认模板
func (dg *DataGenerator) initDefaultTemplates() {
	// 这里可以添加一些常见列名的默认模板
	// 例如：email、password、username 等
}

// GenerateStringWithLength 生成指定长度的字符串
func (dg *DataGenerator) GenerateStringWithLength(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[dg.getRand().IntN(len(charset))]
	}
	return string(b)
}

// GenerateRandomInt 生成指定范围的随机整数
func (dg *DataGenerator) GenerateRandomInt(min, max int64) int64 {
	return min + int64(dg.getRand().IntN(int(max-min+1)))
}

// GenerateRandomFloat 生成指定范围的随机浮点数
func (dg *DataGenerator) GenerateRandomFloat(min, max float64) float64 {
	return min + dg.getRand().Float64()*(max-min)
}

// GenerateBatch 批量生成指定类型的值
func (dg *DataGenerator) GenerateBatch(col mysql_schema.Column, count int) []any {
	result := make([]any, count)
	for i := 0; i < count; i++ {
		result[i] = dg.GenerateValue(col)
	}
	return result
}
