# Template 系统 API 参考

## 核心类型

### DataGenerator 结构体

```go
type DataGenerator struct {
    seed      uint64
    randSrc   rand.Source
    rnd       *rand.Rand
    templates map[string]ColumnTemplate
}
```

主要的数据生成器，用于生成各种类型的测试数据。

#### 方法

##### NewDataGenerator() *DataGenerator

创建新的数据生成器实例。

**示例：**
```go
dg := NewDataGenerator()
```

**特性：**
- 自动使用当前时间作为种子
- 自动注册所有内置模板
- 线程不安全（计数器模板）

---

##### SetSeed(seed uint64)

为数据生成器设置随机数种子。

**参数：**
- `seed`: 随机数种子

**示例：**
```go
dg := NewDataGenerator()
dg.SetSeed(12345)

// 再次创建生成器并使用相同种子
dg2 := NewDataGenerator()
dg2.SetSeed(12345)

// dg 和 dg2 生成的序列相同
```

**用途：**
- 测试中重现数据
- 调试生成的数据
- 确保可预测的随机性

---

##### GenerateValue(col mysql_schema.Column) any

根据列定义生成对应类型的值。

**参数：**
- `col`: 列定义，包含列名和数据类型

**返回值：**
- 生成的值（类型取决于列定义或模板）

**示例：**
```go
dg := NewDataGenerator()

// 生成邮箱
col := mysql_schema.Column{Name: "email", DataType: "varchar"}
email := dg.GenerateValue(col)
fmt.Println(email)  // 输出: user_567890@gmail.com

// 生成整数
col2 := mysql_schema.Column{Name: "age", DataType: "int"}
age := dg.GenerateValue(col2)
fmt.Println(age)  // 输出: 42

// 生成 UUID
col3 := mysql_schema.Column{Name: "id", DataType: "varchar"}
uuid := dg.GenerateValue(col3)
fmt.Println(uuid)  // 输出: 550e8400-e29b-41d4-a716-446655440000
```

**优先级：**
1. 首先检查是否有注册的自定义模板
2. 其次根据列名查找内置模板
3. 最后根据数据类型生成通用值

---

##### RegisterTemplate(columnName string, generator func() any)

为特定的列名注册自定义生成模板。

**参数：**
- `columnName`: 列名
- `generator`: 生成函数，返回生成的值

**示例：**
```go
dg := NewDataGenerator()

// 注册自定义模板：订单ID
counter := int64(0)
dg.RegisterTemplate("order_id", func() any {
    counter++
    return fmt.Sprintf("ORD-%010d", counter)
})

// 使用自定义模板
col := mysql_schema.Column{Name: "order_id"}
fmt.Println(dg.GenerateValue(col))  // 输出: ORD-0000000001
fmt.Println(dg.GenerateValue(col))  // 输出: ORD-0000000002
```

**注意：**
- 自定义模板优先级最高
- 可以覆盖内置模板
- 生成函数应该是幂等的或有意地改变状态

---

##### GenerateBatch(col mysql_schema.Column, count int) []any

批量生成指定列的多个值。

**参数：**
- `col`: 列定义
- `count`: 要生成的值的个数

**返回值：**
- 包含生成值的切片

**示例：**
```go
dg := NewDataGenerator()

col := mysql_schema.Column{Name: "username", DataType: "varchar"}
usernames := dg.GenerateBatch(col, 5)

for i, username := range usernames {
    fmt.Printf("%d: %v\n", i+1, username)
}
// 输出:
// 1: user_567890
// 2: player_123456
// 3: member_789012
// 4: user_234567
// 5: admin_345678
```

---

##### GenerateStringWithLength(length int) string

生成指定长度的随机字符串。

**参数：**
- `length`: 字符串长度

**返回值：**
- 指定长度的随机字符串

**示例：**
```go
dg := NewDataGenerator()

str := dg.GenerateStringWithLength(20)
fmt.Println(len(str))  // 输出: 20
fmt.Println(str)       // 输出: aB3cD!@xY9ZwkL9mN@pQ
```

**字符集：**
- 大写字母: A-Z
- 小写字母: a-z
- 数字: 0-9

---

##### GenerateRandomInt(min, max int64) int64

生成指定范围内的随机整数。

**参数：**
- `min`: 最小值（包含）
- `max`: 最大值（包含）

**返回值：**
- 范围内的随机整数

**示例：**
```go
dg := NewDataGenerator()

// 生成 1 到 100 之间的随机数
val := dg.GenerateRandomInt(1, 100)
fmt.Println(val)  // 输出: 42

// 生成 0 到 1000000 之间的随机数
id := dg.GenerateRandomInt(0, 1000000)
fmt.Println(id)   // 输出: 567890
```

---

##### GenerateRandomFloat(min, max float64) float64

生成指定范围内的随机浮点数。

**参数：**
- `min`: 最小值（包含）
- `max`: 最大值（不包含）

**返回值：**
- 范围内的随机浮点数

**示例：**
```go
dg := NewDataGenerator()

// 生成 0.0 到 100.0 之间的随机数
price := dg.GenerateRandomFloat(0.0, 100.0)
fmt.Println(price)  // 输出: 45.67

// 生成 -10.0 到 10.0 之间的随机数
temp := dg.GenerateRandomFloat(-10.0, 10.0)
fmt.Println(temp)   // 输出: 3.45
```

---

### ColumnTemplate 结构体

```go
type ColumnTemplate struct {
    Name      string
    Generator func() any
}
```

表示一个列值生成模板。

---

## 模板注册函数

### RegisterAllBuiltinTemplates(dg *DataGenerator)

向数据生成器注册所有内置模板。

**调用时机：**
- 在 `NewDataGenerator()` 中自动调用
- 无需手动调用

**包含的模板：**
- 身份标识: uuid, object_id, snowflake
- 用户信息: username, email, phone, password, real_name, nickname
- 地址信息: country, province, city, address, zip_code, ipv4, ipv6
- 业务数据: url, domain, company, product, money, price
- 状态信息: status, gender, boolean
- 时间信息: birthday, created_at, updated_at
- 计数器: counter, sequence

---

### GetAvailableTemplates() []string

获取所有可用的模板名称列表。

**返回值：**
- 包含所有模板名称的字符串切片

**示例：**
```go
templates := GetAvailableTemplates()
fmt.Printf("Available templates: %v\n", templates)
fmt.Printf("Total count: %d\n", len(templates))

// 遍历所有模板
for i, tmpl := range templates {
    fmt.Printf("%2d) %s\n", i+1, tmpl)
}
```

**输出：**
```
Available templates: [uuid object_id snowflake username email ...]
Total count: 30

 1) uuid
 2) object_id
 3) snowflake
 4) username
 5) email
...
```

---

## 工具函数

### ParseCount(countStr string) (int, error)

解析计数字符串，支持数字、"random" 或空字符串。

**参数：**
- `countStr`: 计数字符串

**返回值：**
- `count`: 解析结果
- `error`: 解析错误

**示例：**
```go
// 明确的数字
count, err := ParseCount("10")
// count = 10, err = nil

// 随机值
count, err := ParseCount("random")
// count = 1-100 之间的随机数, err = nil

// 空字符串
count, err := ParseCount("")
// count = 1, err = nil

// 无效输入
count, err := ParseCount("-5")
// count = 0, err = error
```

---

### IsValidOperation(op string) bool

检查操作类型是否有效。

**参数：**
- `op`: 操作类型字符串

**返回值：**
- 操作是否有效

**有效操作：**
- insert
- update
- update_join
- delete
- replace

**示例：**
```go
IsValidOperation("insert")      // true
IsValidOperation("update")      // true
IsValidOperation("delete")      // true
IsValidOperation("select")      // false
IsValidOperation("")            // false
```

---

### NeedsOldValue(operation string) bool

判断操作是否需要旧值。

**参数：**
- `operation`: 操作类型

**返回值：**
- 是否需要旧值

**需要旧值的操作：**
- update
- update_join

**示例：**
```go
NeedsOldValue("update")         // true
NeedsOldValue("update_join")    // true
NeedsOldValue("insert")         // false
NeedsOldValue("delete")         // false
```

---

### NeedsGuideKeys(operation string) bool

判断操作是否需要指南键（WHERE 条件）。

**参数：**
- `operation`: 操作类型

**返回值：**
- 是否需要指南键

**需要指南键的操作：**
- update
- update_join
- delete

**示例：**
```go
NeedsGuideKeys("update")        // true
NeedsGuideKeys("delete")        // true
NeedsGuideKeys("insert")        // false
NeedsGuideKeys("replace")       // false
```

---

## 类型定义

### Column 结构体扩展

```go
type Column struct {
    Column string // 列名
    Type   string // 列类型
    Mock   string // Mock 模板名称（可选）
}
```

新增的 `Mock` 字段用于指定列使用的模板。

---

## 使用场景

### 场景 1：生成用户注册数据

```go
dg := NewDataGenerator()

type User struct {
    ID       any
    Username any
    Email    any
    Phone    any
    Password any
    City     any
    CreatedAt any
}

user := User{
    ID:       dg.GenerateValue(mysql_schema.Column{Name: "id", DataType: "varchar"}),
    Username: dg.GenerateValue(mysql_schema.Column{Name: "username"}),
    Email:    dg.GenerateValue(mysql_schema.Column{Name: "email"}),
    Phone:    dg.GenerateValue(mysql_schema.Column{Name: "phone"}),
    Password: dg.GenerateValue(mysql_schema.Column{Name: "password"}),
    City:     dg.GenerateValue(mysql_schema.Column{Name: "city"}),
    CreatedAt: dg.GenerateValue(mysql_schema.Column{Name: "created_at"}),
}
```

### 场景 2：批量生成订单数据

```go
dg := NewDataGenerator()

// 生成 1000 个订单号
orderIds := dg.GenerateBatch(
    mysql_schema.Column{Name: "order_id", DataType: "varchar"},
    1000,
)

for i, id := range orderIds {
    fmt.Printf("Order %d: %v\n", i+1, id)
}
```

### 场景 3：生成可复现的测试数据

```go
dg1 := NewDataGenerator()
dg1.SetSeed(12345)

dg2 := NewDataGenerator()
dg2.SetSeed(12345)

col := mysql_schema.Column{Name: "email"}

// 两个生成器会生成相同的序列
email1 := dg1.GenerateValue(col)
email2 := dg2.GenerateValue(col)

if email1 == email2 {
    fmt.Println("Seed works correctly!")
}
```

### 场景 4：使用自定义模板

```go
dg := NewDataGenerator()

// 为产品库存使用自定义生成逻辑
stock := 0
dg.RegisterTemplate("stock", func() any {
    stock += 10
    return stock
})

for i := 0; i < 5; i++ {
    col := mysql_schema.Column{Name: "stock"}
    fmt.Println(dg.GenerateValue(col))
}
// 输出: 10, 20, 30, 40, 50
```

---

## 性能指标

| 操作 | 耗时 | 备注 |
|------|------|------|
| 生成 UUID | < 1ms | 格式化耗时 |
| 生成邮箱 | < 1ms | 随机选择和格式化 |
| 生成手机号 | < 1ms | 前缀选择和数字生成 |
| 生成密码 | < 1ms | 字符随机选择 |
| 批量生成 1000 个值 | < 100ms | 平均 0.1ms/个 |
| 内存占用 | < 1MB | 不涉及大数据结构 |

---

## 错误处理

### 处理无效的列名

```go
dg := NewDataGenerator()

col := mysql_schema.Column{Name: "unknown_column", DataType: "varchar"}
value := dg.GenerateValue(col)

// 如果列名不匹配任何模板，会根据数据类型生成通用值
fmt.Println(value)  // 输出: val_12345
```

### 处理无效的数据类型

```go
dg := NewDataGenerator()

col := mysql_schema.Column{Name: "field", DataType: "unknown_type"}
value := dg.GenerateValue(col)

// 未知类型默认作为字符串处理
fmt.Println(value)  // 输出: val_67890
```

---

## 最佳实践

1. **单例使用**：在应用中使用单个 DataGenerator 实例
   ```go
   var dg = NewDataGenerator()
   ```

2. **模板复用**：对相同含义的列使用相同的模板
   ```go
   dg.GenerateValue(Column{Name: "creator_email", ...})    // email
   dg.GenerateValue(Column{Name: "updater_email", ...})    // email
   ```

3. **种子设置**：在测试中设置种子以确保可复现性
   ```go
   dg.SetSeed(12345)
   ```

4. **自定义模板**：对业务特定的字段使用自定义模板
   ```go
   dg.RegisterTemplate("my_field", myGenerator)
   ```

5. **错误处理**：虽然 GenerateValue 不返回错误，但应验证生成的数据
   ```go
   value := dg.GenerateValue(col)
   if value == nil || value == "" {
       log.Fatal("Failed to generate value")
   }
   ```

---
