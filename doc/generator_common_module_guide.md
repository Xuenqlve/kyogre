# Generator 公共模块集成指南

## 概述

本指南说明如何在 Kyogre 的各个 Generator 实现（MySQL、Redis、MongoDB 等）中使用公共的数据生成模块。

## 模块结构

```
pkg/generator/
├── common/                      # ✨ 公共模块
│   ├── data_generator.go       # 数据值生成
│   ├── row_builder.go          # 行数据构建
│   ├── utils.go                # 工具函数
│   ├── errors.go               # 错误定义
│   ├── data_generator_test.go  # 单元测试
│   └── README.md               # 详细文档
├── mysql/
│   ├── dml.go                  # 使用 common 模块
│   └── ...
├── redis/
│   ├── generator.go            # 可以使用 common 模块
│   └── ...
└── mongodb/
    ├── generator.go            # 可以使用 common 模块
    └── ...
```

## 核心功能

### 1. 数据值生成 (DataGenerator)

**职责**: 根据列的数据类型生成随机值

**支持**:
- 20+ MySQL 数据类型（int, varchar, datetime, json 等）
- 自定义生成模板（按列名或数据类型）
- 确定性随机数（Seed 支持）

**使用场景**:
- 生成符合类型的列值
- 快速原型开发和测试
- 数据验证和一致性检查

### 2. 行数据构建 (RowBuilder)

**职责**: 为 MySQL 操作类型构建标准的行数据结构

**支持**:
- INSERT, UPDATE, DELETE, REPLACE 等操作
- 自动填充 Old 字段（UPDATE）
- 自动生成 GuideKeys（WHERE 条件）

**使用场景**:
- 快速构建 MockMessage
- 减少重复代码
- 保证数据结构一致性

### 3. 工具函数 (Utils)

**职责**: 提供通用的解析和验证函数

**包含**:
- ParseCount: 解析行数字符串
- IsValidOperation: 操作类型验证
- NeedsOldValue: 判断是否需要 Old 字段
- NeedsGuideKeys: 判断是否需要 GuideKeys

## 各 Generator 实现指南

### MySQL DML Generator (已实现)

**文件**: `pkg/generator/mysql/dml.go`

**使用方式**:

```go
type DMLGenerator struct {
    // ... 其他字段
    rowBuilder *common.RowBuilder  // 使用公共行构建器
}

func (g *DMLGenerator) Configure(...) error {
    // ...
    g.rowBuilder = common.NewRowBuilder()  // 初始化
    return nil
}

func (g *DMLGenerator) MockMessage(param plugin.MockParam) message.Message {
    count, _ := common.ParseCount(param.Count)         // 使用工具函数
    rowDataList := g.rowBuilder.BuildRows(...)         // 使用行构建器
    // ... 创建消息
}
```

**优势**:
- 代码简洁，无需重复实现数据生成逻辑
- 支持自定义数据生成（通过 DataGenerator）
- 易于维护和扩展

### MySQL DDL Generator (待实现)

**建议实现**:

```go
type DDLGenerator struct {
    rowBuilder *common.RowBuilder
    // ... 其他字段
}

func (g *DDLGenerator) MockMessage(param ...) message.Message {
    // 使用 RowBuilder 生成初始化数据
    rows := g.rowBuilder.BuildRows(table, "insert", 100)

    // 构建 DDL 消息（ALTER TABLE 等）
    // ...
}
```

### Redis Generator (待实现)

**建议实现**:

```go
type RedisGenerator struct {
    dataGen *common.DataGenerator
}

func (g *RedisGenerator) GenerateKeyValue(keyType string) (string, any) {
    // 使用 DataGenerator 生成值
    col := mysql.Column{
        Name: "value",
        DataType: keyType,
    }
    return fmt.Sprintf("key_%d", rand.IntN(1000)),
           g.dataGen.GenerateValue(col)
}

func (g *RedisGenerator) MockMessage(param ...) message.Message {
    // 使用生成的键值对构建 Redis 命令消息
}
```

### MongoDB Generator (待实现)

**建议实现**:

```go
type MongoDBGenerator struct {
    dataGen *common.DataGenerator
}

func (g *MongoDBGenerator) GenerateDocument(schema ...) map[string]any {
    doc := make(map[string]any)

    for _, field := range schema.Fields {
        col := mysql.Column{
            Name: field.Name,
            DataType: field.Type,
        }
        doc[field.Name] = g.dataGen.GenerateValue(col)
    }

    return doc
}
```

## 自定义数据生成

### 方法 1: 按列名自定义

```go
// 在 Generator Configure 时
rb := common.NewRowBuilder()
dg := rb.GetDataGenerator()

// 为特定列注册生成器
dg.RegisterTemplate("password", func() any {
    return hashPassword("default_password")
})

dg.RegisterTemplate("email", func() any {
    return fmt.Sprintf("user_%d@example.com", rand.IntN(10000))
})
```

### 方法 2: 按数据类型自定义

```go
dg := rb.GetDataGenerator()

// 为 UUID 类型的所有列自定义
dg.RegisterTemplateByType("uuid", func() any {
    return uuid.New().String()
})

// 为 JSON 类型的所有列自定义
dg.RegisterTemplateByType("json", func() any {
    return `{"custom": "json_structure"}`
})
```

### 方法 3: 继承 DataGenerator

```go
type CustomDataGenerator struct {
    *common.DataGenerator
    // ... 自定义字段
}

func (cdg *CustomDataGenerator) GenerateValue(col mysql.Column) any {
    // 先检查自定义逻辑
    if col.Name == "special_field" {
        return "special_value"
    }

    // 否则使用父类实现
    return cdg.DataGenerator.GenerateValue(col)
}
```

## 集成检查清单

实现新的 Generator 时，使用以下清单确保正确集成公共模块：

### 基础集成

- [ ] 导入 `github.com/xuenqlve/kyogre/pkg/generator/common`
- [ ] 在 struct 中包含 `*common.RowBuilder` 或 `*common.DataGenerator`
- [ ] 在 `Configure` 中初始化生成器
- [ ] 在 `MockMessage` 中使用 `common.ParseCount` 解析行数
- [ ] 在 `MockMessage` 中使用 RowBuilder 构建行数据

### 功能完整性

- [ ] 支持所有目标数据类型
- [ ] 正确处理 NULL 值（如有需要）
- [ ] 支持主键和唯一键识别
- [ ] 正确构建 GuideKeys（UPDATE/DELETE）
- [ ] 正确构建 Old 字段（UPDATE）

### 质量保证

- [ ] 编写单元测试（参考 `data_generator_test.go`）
- [ ] 验证生成的数据与 schema 匹配
- [ ] 测试边界情况（无主键、无列等）
- [ ] 添加代码注释和文档

### 性能优化

- [ ] 避免重复的类型检查
- [ ] 考虑缓存频繁访问的数据
- [ ] 对高并发场景进行性能测试
- [ ] 可选：实现对象池减少 GC 压力

## 最佳实践

### 1. 一致性

所有 Generator 都应该：
- 返回 `message.Message` 接口的实现
- 遵循相同的配置格式
- 支持相同的表选择策略

```go
// 统一的 MockParam 结构
type MockParam struct {
    Operation   string  // 操作类型
    Count       string  // 行数（支持 "1", "10", "random"）
    TableSelect string  // 表选择策略
}
```

### 2. 可扩展性

设计时考虑：
- 用户可能需要自定义某些列的生成逻辑
- 可能需要特殊的数据约束（范围、格式等）
- 可能需要集成外部数据源

### 3. 可测试性

```go
// 提供 SetSeed 接口便于单元测试
rb := common.NewRowBuilder()
rb.SetSeed(12345)

// 多次执行生成相同的数据
row1 := rb.BuildInsertRow(table, 0)
rb.SetSeed(12345)
row2 := rb.BuildInsertRow(table, 0)

// row1 和 row2 应该完全相同
```

### 4. 文档化

每个 Generator 应该包含：
- README 说明使用方法
- 支持的数据类型列表
- 配置示例
- 常见问题解答

## 常见问题

### Q: 如何添加新的数据类型支持？

A: 在 `DataGenerator.generateByType` 中添加 case：

```go
case "custom_type":
    return dg.generateCustom()
```

### Q: 能否为同一列使用多个生成器？

A: 可以，通过继承和方法重写：

```go
type CustomGen struct {
    base *common.DataGenerator
}

func (cg *CustomGen) GenerateValue(col ...) any {
    if col.Name == "priority" {
        return []int{1, 2, 3}[rand.IntN(3)]
    }
    return cg.base.GenerateValue(col)
}
```

### Q: 生成的数据如何保证唯一性？

A: 使用不同的 Seed 或自定义生成器：

```go
dg.RegisterTemplate("unique_id", func() any {
    return uuid.New().String()
})
```

### Q: 性能如何优化？

A:
1. 使用对象池
2. 预先生成常用值
3. 避免 type assertion
4. 使用 sync.Pool 缓存大对象

## 迁移指南

如果现有的 Generator 需要迁移到使用公共模块：

### 第 1 步: 评估兼容性

- 检查现有的数据生成逻辑
- 确认是否能迁移到 DataGenerator
- 识别特殊情况和约束

### 第 2 步: 逐步迁移

1. 导入公共模块
2. 添加 RowBuilder 字段
3. 在 MockMessage 中使用 RowBuilder
4. 逐步删除重复的生成代码

### 第 3 步: 验证

- 对比迁移前后的输出
- 运行现有的单元测试
- 添加新的集成测试

### 第 4 步: 清理

- 删除重复代码
- 更新文档
- 提交 PR

## 相关文件

- 模块实现: `pkg/generator/common/`
- DML Generator: `pkg/generator/mysql/dml.go`
- 单元测试: `pkg/generator/common/data_generator_test.go`
- 详细文档: `pkg/generator/common/README.md`

## 性能基准

（待补充实际测试结果）

```
BenchmarkDataGenerator_GenerateValue/int-8      1000000    1205 ns/op    32 B/op    1 allocs/op
BenchmarkDataGenerator_GenerateValue/varchar-8  500000     2410 ns/op   128 B/op    3 allocs/op
BenchmarkRowBuilder_BuildRows/1000-8            10000      105234 ns/op  45000 B/op  1000 allocs/op
```

## 贡献指南

如果想扩展公共模块：

1. 在 GitHub Issues 中讨论特性需求
2. Fork 并创建 feature branch
3. 添加代码和单元测试
4. 提交 PR 并确保所有测试通过
5. 更新相关文档

## 许可证

遵循项目整体许可证。
