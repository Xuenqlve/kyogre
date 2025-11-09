# MySQL DML Generator 实现说明

## 概述

`DMLGenerator` 是 Kyogre 压测框架中的核心组件，负责根据表的元数据生成单条 DML 消息（INSERT、UPDATE、DELETE 等）。

## 核心功能

### 1. MockMessage 方法 (pkg/generator/mysql/dml.go:72)

生成一条符合表结构的 MySQL 行操作消息。

**输入参数**:
```go
param plugin.MockParam{
    Operation: "insert|update|delete"  // DML 操作类型
    Count: "1|random|123"              // 生成的行数
    TableSelect: "random|ordered"      // 表选择策略
}
```

**输出**: 返回 `message.MySQLRowMessage` 结构体，包含：
- Database：数据库名
- Table：表名
- Operation：操作类型
- WriteType：写入类型
- Contents：行数据列表（`mysql.RowData`）
- StartTime：消息生成时间

### 2. 表选择策略

Generator 支持 4 种表选择方式：

| 策略 | 常量 | 说明 |
|------|------|------|
| 随机 | `plugin.RandomTableSelect` | 从所有表中随机选择 |
| 顺序 | `plugin.OrderedTableSelect` | 按顺序轮流选择表 |
| 相同 | `plugin.SameWithLastTableSelect` | 继续使用上次的表 |
| 不同 | `plugin.DiffFromLastTableSelect` | 与上次不同的表 |

## 实现细节

### 行数据生成

#### 1. 行 Key 生成 (`generateRowKey`)
```go
format: "database.table.pk_value"
example: "test.users.pk_0_0"
```

用途：唯一标识一行数据，用于日志追踪和去重。

#### 2. 列数据生成 (`generateColumnData`)

为表的每一列生成符合类型的随机数据：

| MySQL 类型 | 生成值 | 示例 |
|----------|--------|------|
| `int, bigint` | 随机整数 | 123456 |
| `varchar, char` | 随机字符串 | `val_45678` |
| `text, longtext` | 随机文本 | `text_98765` |
| `timestamp, datetime` | 当前时间 | `2025-11-09 15:30:00` |
| `bool, boolean` | 随机布尔 | true/false |
| `json, jsonb` | JSON 字符串 | `{"key":"val_12345"}` |
| 其他 | 默认字符串 | `val_xxxxx` |

#### 3. 主键/唯一键生成 (`generateGuideKeys`)

用于 UPDATE/DELETE 操作的 WHERE 条件：

1. 优先使用主键列（`PrimaryIndex`）
2. 其次使用唯一键（`ColumnKey == "UNI"`）
3. 默认使用第一列

```go
// 示例输出
{
    "id": "pk_0_0",
    "email": "user_xxx@example.com"
}
```

### 行数计算 (`parseCount`)

支持三种 Count 格式：

| 格式 | 说明 | 示例 |
|------|------|------|
| 空字符串 | 默认 1 行 | `""` → 1 |
| 数字 | 生成指定行数 | `"10"` → 10 |
| "random" | 生成 1-100 的随机行数 | `"random"` → 45 |

### UPDATE 操作特殊处理

对于 UPDATE 和 UPDATE JOIN 操作，需要生成 `Old` 字段：

```go
rowData := mysql.RowData{
    Old:  {col1: oldValue1, col2: oldValue2, ...},  // 修改前的值
    Data: {col1: newValue1, col2: newValue2, ...},  // 修改后的值
    GuideKeys: {id: pkValue},                       // WHERE 条件
}
```

## 使用示例

### 1. 配置和初始化

```go
// 创建 Generator
gen := &mysql.DMLGenerator{}

// 配置（传入元数据名称）
err := gen.Configure("my-pipeline", map[string]any{
    "metadata": "mysql-config",
})

// 注册元数据
metadata := /* 获取元数据实例 */
gen.RegisterMetadata(metadata)
```

### 2. 生成 INSERT 消息

```go
msg := gen.MockMessage(plugin.MockParam{
    Operation: message.Insert,
    Count: "5",           // 生成 5 行
    TableSelect: "random",
})

// msg 是 *message.MySQLRowMessage，包含 5 行数据
// 可直接传给 Pressure 引擎执行
```

### 3. 生成 UPDATE 消息

```go
msg := gen.MockMessage(plugin.MockParam{
    Operation: message.Update,
    Count: "10",
    TableSelect: "ordered",
})

// msg 包含 10 行更新操作
// 每行都有 Old、Data 和 GuideKeys 三个部分
```

### 4. 生成 DELETE 消息

```go
msg := gen.MockMessage(plugin.MockParam{
    Operation: message.Delete,
    Count: "random",  // 随机行数（1-100）
    TableSelect: "same_with_last",
})

// msg 包含随机数量的删除操作
```

## 数据结构

### mysql.RowData

```go
type RowData struct {
    Key       string         // 行的唯一标识
    Data      map[string]any // 列名 -> 列值（用于 INSERT/UPDATE）
    Old       map[string]any // 修改前的值（仅 UPDATE 需要）
    GuideKeys map[string]any // WHERE 条件的列 -> 列值
}
```

### message.MySQLRowMessage

```go
type MySQLRowMessage struct {
    SQLRows message.SQLRows // 包含元数据和行数据
}

type SQLRows struct {
    Metadata message.Metadata
    Contents []mysql.RowData
}

type Metadata struct {
    Operation string    // "insert", "update", "delete"
    Database  string    // 数据库名
    Table     string    // 表名
    Hint      string    // SQL hint（可选）
    WriteType string    // 写入类型
    StartTime time.Time // 消息生成时间
}
```

## 性能优化建议

### 1. 随机数生成

使用 `math/rand/v2` 的线程安全随机数生成：
```go
import "math/rand/v2"

rand.IntN(1000000)  // 生成 0-999999 的随机整数
```

### 2. 缓冲策略

对于高吞吐量场景：
```go
// Scenario 可以预先生成多条消息放入缓冲区
messages := make([]message.Message, 0, 1000)
for i := 0; i < 1000; i++ {
    msg := gen.MockMessage(param)
    messages = append(messages, msg)
}
// 批量发送给 Pressure
```

### 3. 对象池

避免频繁的内存分配：
```go
// 可选：为高频 Generator 创建对象池
var rowDataPool = sync.Pool{
    New: func() any { return &mysql.RowData{} },
}

func (g *DMLGenerator) getRowData() *mysql.RowData {
    return rowDataPool.Get().(*mysql.RowData)
}
```

## 常见问题

### Q: 生成的主键值会重复吗？

A: 不会。主键值格式为 `pk_hitIndex_index`，其中 `hitIndex` 为表的选择序号，`index` 为行的序号，确保全局唯一。

### Q: 支持复合主键吗？

A: 支持。如果表有复合主键，`generateGuideKeys` 会为每个主键列都生成对应的 GuideKey。

### Q: 生成的数据可以自定义吗？

A: 当前版本使用默认的随机生成策略。可扩展的方式：
1. 在 Config 中添加自定义规则字段
2. 添加 Seed 数据源
3. 扩展 `generateColumnValue` 方法支持模板

### Q: 时间戳生成精度如何？

A: 使用 `time.Now()` 生成，精度为纳秒级。每条消息的 `StartTime` 都是调用时刻的时间。

## 扩展点

### 自定义列值生成

```go
// 继承 DMLGenerator
type CustomGenerator struct {
    DMLGenerator
}

// 重写方法
func (g *CustomGenerator) generateColumnValue(col mysql.Column) any {
    // 自定义逻辑
    if col.Name == "email" {
        return fmt.Sprintf("user_%d@example.com", rand.IntN(100000))
    }
    return g.DMLGenerator.generateColumnValue(col)
}
```

### 支持更多数据类型

修改 `generateColumnValue` 的 switch 语句：

```go
case "uuid":
    return uuid.New().String()
case "inet":
    return fmt.Sprintf("192.168.%d.%d", rand.IntN(256), rand.IntN(256))
// ... 更多类型
```

## 集成示例

与 Scenario 的集成流程：

```
Generator
   ↓ (MockMessage)
Message (MySQLRowMessage)
   ↓ (Scenario)
Message Channel
   ↓ (Pressure)
Database Execution
```

示例代码：
```go
// Scenario 中循环生成消息
ticker := time.NewTicker(time.Millisecond * 10)
for range ticker.C {
    // 随机选择一个 Generator
    selectedGen := scenario.selectGenerator()

    // 生成消息
    msg := selectedGen.MockMessage(plugin.MockParam{
        Operation: "insert",
        Count: "1",
        TableSelect: "random",
    })

    // 推送给 Pressure
    select {
    case messageChannel <- msg:
    case <-ctx.Done():
        return
    }
}
```

## 测试验证

参考测试文件：`test/pressure/mysql_row_test.go`

```go
// 验证 INSERT 消息结构
msg := gen.MockMessage(plugin.MockParam{
    Operation: message.Insert,
    Count: "2",
    TableSelect: "random",
})

assert.NotNil(t, msg)
assert.Equal(t, message.MySQLRow, msg.Type())
assert.Equal(t, 2, len(msg.(*message.MySQLRowMessage).Contents))
```
