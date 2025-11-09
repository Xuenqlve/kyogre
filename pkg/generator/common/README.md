# Common Generator Module

公共的数据生成模块，提供可复用的数据生成、行数据构建等功能。其他 Generator 实现（MySQL、Redis、MongoDB 等）都可以基于此模块进行开发。

## 核心组件

### 1. DataGenerator - 数据值生成器

负责根据 MySQL 列类型生成对应的随机数据。

```go
// 创建生成器
dg := NewDataGenerator()

// 生成单个值
col := mysql.Column{
    Name: "age",
    DataType: "int",
}
value := dg.GenerateValue(col)

// 设置 Seed（便于测试复现）
dg.SetSeed(12345)

// 批量生成
values := dg.GenerateBatch(col, 100)
```

#### 支持的数据类型

| MySQL 类型 | Go 类型 | 示例值 |
|----------|--------|--------|
| `int`, `bigint`, `smallint` | `int64` | `123456` |
| `float`, `double`, `decimal` | `float64` | `12345.67` |
| `varchar`, `char`, `string` | `string` | `val_45678` |
| `text`, `longtext` | `string` | `text_98765_87654_...` |
| `bool`, `boolean` | `bool` | `true` |
| `timestamp`, `datetime` | `string` | `2025-11-09 15:30:00` |
| `date` | `string` | `2025-11-09` |
| `time` | `string` | `15:30:00` |
| `json`, `jsonb` | `string` | `{"id":123,"name":"user_456"}` |
| `uuid` | `string` | `550e8400-e29b-41d4-a716-446655440000` |
| `blob`, `longblob` | `string` | `blob_a1b2c3d4_...` |

#### 自定义数据生成

```go
dg := NewDataGenerator()

// 为特定列名注册生成器
dg.RegisterTemplate("email", func() any {
    return "user@example.com"
})

// 为特定数据类型注册生成器
dg.RegisterTemplateByType("uuid", func() any {
    return uuid.New().String()
})

col := mysql.Column{
    Name: "email",
    DataType: "varchar",
}
value := dg.GenerateValue(col) // 返回 "user@example.com"
```

### 2. RowBuilder - 行数据构建器

便于构建符合 MySQL DML 操作的行数据（RowData 结构体）。

```go
rb := NewRowBuilder()

// 构建 INSERT 行数据
row := rb.BuildInsertRow(tableDef, 0)

// 构建 UPDATE 行数据（包含 Old 字段）
row := rb.BuildUpdateRow(tableDef, 0)

// 构建 DELETE 行数据
row := rb.BuildDeleteRow(tableDef, 0)

// 批量构建行数据
rows := rb.BuildRows(tableDef, "insert", 10)
```

#### 支持的操作类型

- **insert**: 标准 INSERT 操作
- **update**: UPDATE 操作（包含 Old 和 Data）
- **delete**: DELETE 操作（只有 GuideKeys）
- **replace**: REPLACE 操作
- **insert_ignore**: INSERT IGNORE 操作
- **insert_on_duplicate_key**: INSERT ON DUPLICATE KEY UPDATE 操作

### 3. 工具函数

#### ParseCount - 解析行数

```go
// 支持多种格式
count, err := ParseCount("")           // 返回 1
count, err := ParseCount("10")         // 返回 10
count, err := ParseCount("random")     // 返回 1-100 随机值

// 在 MockMessage 中使用
count, _ := ParseCount(param.Count)
```

#### 操作验证函数

```go
// 检查操作类型是否有效
IsValidOperation("insert")        // true
IsValidOperation("invalid")       // false

// 检查是否为写操作
IsWriteOperation("insert")        // true

// 检查是否需要 Old 字段
NeedsOldValue("update")           // true
NeedsOldValue("insert")           // false

// 检查是否需要 GuideKeys（WHERE 条件）
NeedsGuideKeys("delete")          // true
NeedsGuideKeys("insert")          // false

// 获取操作的写入类型
GetOperationWriteType("insert")   // "insert"
GetOperationWriteType("update")   // "update"
```

## 使用示例

### 示例 1: 基础数据生成

```go
package main

import (
    "github.com/xuenqlve/kyogre/pkg/generator/common"
    mysql_schema "github.com/xuenqlve/common/relational_database/mysql"
)

func main() {
    // 创建生成器
    dg := common.NewDataGenerator()

    // 定义列
    columns := []mysql_schema.Column{
        {Name: "id", DataType: "int"},
        {Name: "name", DataType: "varchar"},
        {Name: "email", DataType: "varchar"},
        {Name: "is_active", DataType: "bool"},
    }

    // 生成数据
    for _, col := range columns {
        value := dg.GenerateValue(col)
        println(col.Name, ":", value)
    }
}
```

### 示例 2: 构建行数据

```go
func main() {
    // 创建行构建器
    rb := common.NewRowBuilder()

    // 定义表
    table := &mysql_schema.Table{
        Database: "test",
        Table:    "users",
        Columns: []mysql_schema.Column{
            {Name: "id", DataType: "int", ColumnKey: "PRI"},
            {Name: "name", DataType: "varchar"},
            {Name: "email", DataType: "varchar"},
        },
    }

    // 构建单条行数据
    row := rb.BuildInsertRow(table, 0)
    println("Key:", row.Key)
    println("Data:", row.Data)
    println("GuideKeys:", row.GuideKeys)

    // 批量构建行数据
    rows := rb.BuildRows(table, "insert", 10)
    println("Generated", len(rows), "rows")
}
```

### 示例 3: 自定义数据生成

```go
func main() {
    dg := common.NewDataGenerator()

    // 注册自定义生成器
    dg.RegisterTemplate("password", func() any {
        return "hashedPassword_" + uuid.New().String()
    })

    col := mysql_schema.Column{
        Name: "password",
        DataType: "varchar",
    }
    value := dg.GenerateValue(col)
    println("Password:", value)
}
```

## 在其他 Generator 中使用

### 在 DDL Generator 中使用

```go
// 为 DDL 操作生成数据
type DDLGenerator struct {
    dataGen *common.DataGenerator
    rowBuilder *common.RowBuilder
}

func (g *DDLGenerator) GenerateTestData(table *mysql.Table) {
    rows := g.rowBuilder.BuildRows(table, "insert", 100)
    // 使用 rows 执行 INSERT 操作
}
```

### 在 Redis Generator 中使用

```go
// Redis 可以使用 DataGenerator 生成键值对的值
type RedisGenerator struct {
    dataGen *common.DataGenerator
}

func (g *RedisGenerator) GenerateValue(key string, dataType string) any {
    col := mysql.Column{
        Name: key,
        DataType: dataType,
    }
    return g.dataGen.GenerateValue(col)
}
```

## 性能考虑

### 1. 随机数生成

- 使用 `math/rand/v2` 的线程安全随机数生成器
- 可通过 `SetSeed` 固定随机数序列（便于测试）

### 2. 内存管理

- `RowBuilder` 创建的 `RowData` 为值类型（Stack 分配）
- 大量生成时考虑对象池优化

```go
// 对象池示例
var rowPool = sync.Pool{
    New: func() any { return &mysql.RowData{} },
}

func getRow() *mysql.RowData {
    return rowPool.Get().(*mysql.RowData)
}
```

### 3. 数据生成优化

- 预先生成常用列值以减少重复计算
- 使用缓存的 Template 避免每次 lookup

## 错误处理

所有错误都继承自 `GeneratorError`：

```go
if err := someOperation(); err != nil {
    if common.IsGeneratorError(err) {
        code := common.GetErrorCode(err)
        // 根据错误码处理
    }
}
```

## 扩展建议

### 1. 添加新的数据类型

在 `data_generator.go` 的 `generateByType` 中添加 case：

```go
case "custom_type":
    return dg.generateCustom(col)

func (dg *DataGenerator) generateCustom(col mysql_schema.Column) any {
    // 自定义逻辑
}
```

### 2. 添加数据约束

```go
type ColumnConstraint struct {
    MinValue any
    MaxValue any
    Pattern  string // 正则表达式
    Values   []any  // 枚举值
}

func (dg *DataGenerator) GenerateWithConstraint(col mysql_schema.Column, constraint ColumnConstraint) any {
    // 考虑约束条件生成值
}
```

### 3. 统计分析

```go
type GenerationStats struct {
    ColumnName string
    ValueCount int
    UniqueCount int
    MinValue any
    MaxValue any
}
```

## 文件结构

```
pkg/generator/common/
├── data_generator.go        # 核心数据生成器
├── row_builder.go           # 行数据构建器
├── utils.go                 # 工具函数和常量
├── errors.go                # 错误定义
├── data_generator_test.go   # 单元测试
└── README.md                # 本文件
```

## API 参考

### DataGenerator

```go
func NewDataGenerator() *DataGenerator
func (dg *DataGenerator) SetSeed(seed uint64)
func (dg *DataGenerator) GenerateValue(col Column) any
func (dg *DataGenerator) RegisterTemplate(columnName string, generator func() any)
func (dg *DataGenerator) RegisterTemplateByType(dataType string, generator func() any)
func (dg *DataGenerator) GenerateStringWithLength(length int) string
func (dg *DataGenerator) GenerateRandomInt(min, max int64) int64
func (dg *DataGenerator) GenerateRandomFloat(min, max float64) float64
func (dg *DataGenerator) GenerateBatch(col Column, count int) []any
```

### RowBuilder

```go
func NewRowBuilder() *RowBuilder
func (rb *RowBuilder) BuildInsertRow(table *Table, index int) *RowData
func (rb *RowBuilder) BuildUpdateRow(table *Table, index int) *RowData
func (rb *RowBuilder) BuildDeleteRow(table *Table, index int) *RowData
func (rb *RowBuilder) BuildReplaceRow(table *Table, index int) *RowData
func (rb *RowBuilder) BuildRows(table *Table, operation string, count int) []RowData
func (rb *RowBuilder) SetSeed(seed uint64)
func (rb *RowBuilder) GetDataGenerator() *DataGenerator
```

### Utils

```go
func ParseCount(countStr string) (int, error)
func IsValidOperation(op string) bool
func IsWriteOperation(op string) bool
func NeedsOldValue(operation string) bool
func NeedsGuideKeys(operation string) bool
func GetOperationWriteType(operation string) string
func ValidateConfig(config *BatchOperationConfig) error
```
