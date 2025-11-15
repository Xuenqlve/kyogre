package mock

import (
	"fmt"
	"math/rand/v2"
	"time"
)

type DataGenerator struct {
	seed      uint64
	randSrc   rand.Source
	rnd       *rand.Rand
	templates map[string]ColumnTemplate
}

type ColumnTemplate struct {
	Name      string
	Generator func() any
}

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

func (dg *DataGenerator) RegisterTemplate(columnName string, generator func() any) {
	dg.templates[columnName] = ColumnTemplate{
		Name:      columnName,
		Generator: generator,
	}
}

func (dg *DataGenerator) RegisterTemplateByType(dataType string, generator func() any) {
	dg.templates["__type__:"+dataType] = ColumnTemplate{
		Name:      dataType,
		Generator: generator,
	}
}

func (dg *DataGenerator) TemplateValue(mock string) (bool, any) {
	if template, exists := dg.templates[mock]; exists {
		return true, template.Generator()
	}
	return false, nil
}

// GenerateTinyInt 生成 TINYINT 值 (-128 to 127 or 0 to 255)
func (dg *DataGenerator) GenerateTinyInt(isUnsigned bool) int64 {
	if isUnsigned {
		return int64(dg.getRand().IntN(256)) // 0-255
	}
	return int64(dg.getRand().IntN(256) - 128) // -128 to 127
}

// GenerateSmallInt 生成 SMALLINT 值 (-32768 to 32767 or 0 to 65535)
func (dg *DataGenerator) GenerateSmallInt(isUnsigned bool) int64 {
	if isUnsigned {
		return int64(dg.getRand().IntN(65536)) // 0-65535
	}
	return int64(dg.getRand().IntN(65536) - 32768) // -32768 to 32767
}

// GenerateInt 生成 INT 值 (-2147483648 to 2147483647 or 0 to 4294967295)
func (dg *DataGenerator) GenerateInt(isUnsigned bool) int64 {
	if isUnsigned {
		return int64(dg.getRand().IntN(100000000)) // 0到一亿
	}
	return int64(dg.getRand().IntN(100000000) - 50000000) // -5千万 到 5千万
}

// GenerateInteger 生成 BIGINT 值 (-9223372036854775808 to 9223372036854775807)
func (dg *DataGenerator) GenerateInteger(isUnsigned bool) int64 {
	if isUnsigned {
		return dg.getRand().Int64() & 0x7FFFFFFFFFFFFFFF // 正数
	}
	return dg.getRand().Int64() // 任意 64 位整数
}

func (dg *DataGenerator) GenerateRandomIntRange(min, max int64) int64 {
	return min + int64(dg.getRand().IntN(int(max-min+1)))
}

func (dg *DataGenerator) GenerateRandomInt(num int) int {
	return dg.getRand().IntN(num)
}

// GenerateFloat 生成浮点数值
func (dg *DataGenerator) GenerateFloat(isUnsigned bool) float64 {
	val := dg.getRand().Float64() * 1000000
	if !isUnsigned && dg.getRand().IntN(2) == 0 {
		val = -val
	}
	return val
}

func (dg *DataGenerator) GenerateRandomFloat(min, max float64) float64 {
	return min + dg.getRand().Float64()*(max-min)
}

func (dg *DataGenerator) GenerateBool() bool {
	return dg.getRand().IntN(2) == 0
}

const (
	charsetIgnoreCase = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0"
	charsetLow        = "abcdefghijklmnopqrstuvwxyz"
	charset           = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
)

// GenerateString 生成随机字符串
func (dg *DataGenerator) GenerateString(length int, ignoreCase bool) string {
	c := charsetLow
	if ignoreCase {
		c = charsetIgnoreCase
	}
	str := make([]byte, length)
	for i := range str {
		str[i] = c[rand.IntN(len(c))]
	}
	return fmt.Sprintf("val_%s", string(str))
}

func (dg *DataGenerator) GenerateRandomString() string {
	str := make([]byte, dg.getRand().IntN(32))
	for i := range str {
		str[i] = charset[dg.getRand().IntN(len(charset))]
	}
	return fmt.Sprintf("val_%s", string(str))
}

func (dg *DataGenerator) GenerateStringWithLength(length int) string {
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[dg.getRand().IntN(len(charset))]
	}
	return string(b)
}

func (dg *DataGenerator) GenerateRandomLongText() string {
	str := make([]byte, dg.getRand().IntN(32))
	for i := range str {
		str[i] = charset[rand.IntN(len(charset))]
	}
	return fmt.Sprintf("text_%s", string(str))
}

func (dg *DataGenerator) GenerateLongTextWithLength(length int, ignoreCase bool) string {
	c := charsetLow
	if ignoreCase {
		c = charsetIgnoreCase
	}
	str := make([]byte, length)
	for i := range str {
		str[i] = c[rand.IntN(len(c))]
	}
	return fmt.Sprintf("text_%s", string(str))
}

func (dg *DataGenerator) GenerateRandomBlob() string {
	str := make([]byte, dg.getRand().IntN(32))
	for i := range str {
		str[i] = charset[rand.IntN(len(charset))]
	}
	return fmt.Sprintf("blob_%s", string(str))
}

func (dg *DataGenerator) GenerateBlobWithLength(length int, ignoreCase bool) string {
	c := charsetLow
	if ignoreCase {
		c = charsetIgnoreCase
	}
	str := make([]byte, length)
	for i := range str {
		str[i] = c[rand.IntN(len(c))]
	}
	return fmt.Sprintf("blob_%s", string(str))
}

func (dg *DataGenerator) GenerateEnum() string {
	// 提取枚举值列表
	// ENUM 类型的 RawType 格式通常是: "enum('val1','val2','val3')"
	// 这里简化处理，返回一个占位符
	return fmt.Sprintf("enum_val_%d", dg.getRand().IntN(5))
}

func (dg *DataGenerator) GenerateJSON() string {
	return fmt.Sprintf(`{"id":%d,"name":"user_%d","timestamp":"%s"}`,
		dg.getRand().IntN(1000000),
		dg.getRand().IntN(100000),
		time.Now().Format(time.RFC3339))
}

func (dg *DataGenerator) GenerateUUID() string {
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		dg.getRand().IntN(0xffffffff),
		dg.getRand().IntN(0xffff),
		dg.getRand().IntN(0xffff),
		dg.getRand().IntN(0xffff),
		dg.getRand().Int64()&0xffffffffffff)
}

func (dg *DataGenerator) GenerateTime(timeType string) any {
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
