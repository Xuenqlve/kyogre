# Template 系统实现总结

## 📋 项目概述

为 Kyogre Generator 实现了一个完整的模板系统，支持通过配置文件指定常见字段类型的数据生成方式。系统包含 **30+ 预定义模板**，覆盖用户身份、个人信息、地址、业务数据、时间戳等常见场景。

## ✨ 核心成果

### 1. 创建 Template 系统 (`pkg/generator/common/template.go`)

**文件大小**: ~425 行代码

**包含内容**:
- 30+ 模板常量定义
- 7 大类模板注册系统
- 完整的模板生成函数实现

### 2. 模板分类（30+ 模板）

#### 身份标识类（3 个）
- **uuid**: UUID v4 格式 (550e8400-e29b-41d4-a716-446655440000)
- **object_id**: MongoDB ObjectID 风格 (32 位十六进制)
- **snowflake**: Snowflake ID (分布式唯一 ID)

#### 用户信息类（6 个）
- **username**: 用户名 (user_123456)
- **email**: 邮箱地址 (user_123456@gmail.com)
- **phone**: 中国手机号 (13012345678)
- **password**: 随机密码 (aB3cD!@xY9)
- **real_name**: 中文真实姓名 (张伟)
- **nickname**: 英文昵称 (HappyTiger_123)

#### 地址信息类（7 个）
- **country**: 国家名 (China, United States)
- **province**: 中国省份 (北京, 上海)
- **city**: 城市名 (北京, 深圳)
- **address**: 详细地址 (中关村1号写字楼A123室)
- **zip_code**: 邮编 (100001)
- **ipv4**: IPv4 地址 (192.168.1.1)
- **ipv6**: IPv6 地址 (550e:8400:e29b:41d4:a716:4466:5544:0000)

#### 业务数据类（6 个）
- **url**: 网址 (https://example.com/product/12345)
- **domain**: 域名 (example.com)
- **company**: 公司名 (北京科技有限公司)
- **product**: 产品名 (Premium Phone V5)
- **money**: 金额/元 (5234.56)
- **price**: 价格/带符号 (¥234.56)

#### 状态类（3 个）
- **status**: 状态值 (active, inactive, pending)
- **gender**: 性别 (male, female, other)
- **boolean**: 布尔值 (true, false)

#### 时间类（3 个）
- **birthday**: 生日 (1985-06-15) - 18-78 岁
- **created_at**: 创建时间 - 最近 365 天
- **updated_at**: 更新时间 - 最近 30 天

#### 计数器类（2 个）
- **counter**: 自增计数 (1, 2, 3, ...)
- **sequence**: 序列号 (1000000001, 1000000002, ...)

## 🔧 技术实现

### 模板注册机制

```go
// 在 DataGenerator 初始化时自动注册所有模板
func NewDataGenerator() *DataGenerator {
    dg := &DataGenerator{
        templates: make(map[string]ColumnTemplate),
    }
    dg.SetSeed(uint64(time.Now().UnixNano()))
    RegisterAllBuiltinTemplates(dg)  // ✨ 自动注册
    return dg
}
```

### 模板分类注册

系统采用模块化设计，分 7 个注册函数：

```go
func RegisterAllBuiltinTemplates(dg *DataGenerator) {
    registerIdentityTemplates(dg)      // 身份标识
    registerUserTemplates(dg)          // 用户信息
    registerAddressTemplates(dg)       // 地址信息
    registerBusinessTemplates(dg)      // 业务数据
    registerStatusTemplates(dg)        // 状态类
    registerTimeTemplates(dg)          // 时间类
    registerCounterTemplates(dg)       // 计数器
}
```

### 数据生成示例

```go
// 生成邮箱
func generateEmail() any {
    domains := []string{"gmail.com", "qq.com", "163.com", ...}
    domain := domains[rand.IntN(len(domains))]
    return fmt.Sprintf("user_%d@%s", rand.IntN(1000000), domain)
}
// 输出: user_567890@qq.com

// 生成手机号
func generatePhone() any {
    prefixes := []string{"130", "131", "132", ...}
    prefix := prefixes[rand.IntN(len(prefixes))]
    return fmt.Sprintf("%s%08d", prefix, rand.IntN(100000000))
}
// 输出: 13012345678

// 生成 UUID
func generateUUID() any {
    return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x", ...)
}
// 输出: 550e8400-e29b-41d4-a716-446655440000
```

## 📊 数据生成示例

### 用户注册场景

```go
dg := common.NewDataGenerator()

// 邮箱字段 - 使用 email 模板
email := dg.GenerateValue(mysql.Column{Name: "email"})
// 结果: user_567890@qq.com ✅

// 手机字段 - 使用 phone 模板
phone := dg.GenerateValue(mysql.Column{Name: "phone"})
// 结果: 13012345678 ✅

// 用户名 - 使用 username 模板
username := dg.GenerateValue(mysql.Column{Name: "username"})
// 结果: user_123456 ✅

// 创建时间 - 使用 created_at 模板
createdAt := dg.GenerateValue(mysql.Column{Name: "created_at"})
// 结果: 2025-10-20 14:30:45 ✅
```

### 商品信息场景

```go
// 产品名 - 使用 product 模板
product := dg.GenerateValue(mysql.Column{Name: "product"})
// 结果: Premium Phone V5 ✅

// 价格 - 使用 price 模板
price := dg.GenerateValue(mysql.Column{Name: "price"})
// 结果: ¥234.56 ✅

// 公司名 - 使用 company 模板
company := dg.GenerateValue(mysql.Column{Name: "company"})
// 结果: 北京科技有限公司 ✅

// 网站 - 使用 domain 模板
domain := dg.GenerateValue(mysql.Column{Name: "domain"})
// 结果: demo123.com ✅
```

## 🎯 使用方式

### 方式 1: 直接使用（代码中）

```go
dg := common.NewDataGenerator()

// 使用预定义模板
dg.RegisterTemplate("email", func() any {
    return "custom@example.com"
})

col := mysql.Column{Name: "email", DataType: "varchar"}
value := dg.GenerateValue(col)  // custom@example.com
```

### 方式 2: 通过配置文件（后续实现）

```yaml
metadata:
  type: mysql
  config:
    databases:
      test:
        - table: users
          columns:
            - column: email
              type: varchar
              mock: email          # ✨ 使用 email 模板
            - column: phone
              type: varchar
              mock: phone          # ✨ 使用 phone 模板
            - column: created_at
              type: timestamp
              mock: created_at     # ✨ 使用 created_at 模板
```

### 方式 3: 自定义扩展

```go
dg := common.NewDataGenerator()

// 注册自定义模板
dg.RegisterTemplate("custom_id", func() any {
    return fmt.Sprintf("CUST_%d", time.Now().UnixMilli())
})

// 现在可以使用自定义模板
col := mysql.Column{Name: "custom_id"}
value := dg.GenerateValue(col)  // CUST_1699517445123
```

## 📈 特性对比

| 特性 | 之前 | 现在 |
|------|------|------|
| **预定义模板** | 0 | 30+ |
| **支持的字段类型** | 基础 (int, varchar) | 真实业务数据 |
| **自定义能力** | 有限 | 灵活 (可注册自定义) |
| **数据真实性** | 随机值 | 符合实际业务 |
| **开箱即用** | 否 | 是 ✅ |
| **代码行数** | - | ~425 行 |

## 🏗️ 架构设计

```
DataGenerator
    ↓
NewDataGenerator()
    ↓
RegisterAllBuiltinTemplates()
    ├── registerIdentityTemplates()     (3 个)
    ├── registerUserTemplates()         (6 个)
    ├── registerAddressTemplates()      (7 个)
    ├── registerBusinessTemplates()     (6 个)
    ├── registerStatusTemplates()       (3 个)
    ├── registerTimeTemplates()         (3 个)
    └── registerCounterTemplates()      (2 个)
    ↓
GenerateValue(col)
    ↓
检查 templates[col.Name]
    ↓
使用对应模板生成值
```

## 📚 文档清单

### 创建的文件

| 文件 | 行数 | 说明 |
|------|------|------|
| `pkg/generator/common/template.go` | 425 | 模板系统核心 |
| `doc/template_system_implementation.md` | - | 本文档 |

### 修改的文件

| 文件 | 修改内容 |
|------|---------|
| `pkg/generator/common/data_generator.go` | 新增 RegisterAllBuiltinTemplates 调用 |

## 🔮 后续扩展计划

### 短期（配置集成）
- [ ] 扩展 metadata config 支持 `mock` 字段
- [ ] Column 结构体添加 `Mock` 字段
- [ ] Metadata.Configure() 中集成模板注册

### 中期（更多模板）
- [ ] 银行卡号、身份证号等金融数据
- [ ] 社交媒体用户名、微博、QQ 等
- [ ] 各国城市、地区、邮编库
- [ ] 行业、职位、部门等组织数据

### 长期（高级功能）
- [ ] 模板参数化（如生成指定范围的金额）
- [ ] 模板组合（如生成"姓名 + 身份证"的对应关系）
- [ ] 数据一致性（如 IP 地址和城市的关联）
- [ ] 性能优化（预生成缓存、批量生成）

## ✅ 质量保证

- ✅ 代码编译通过
- ✅ 无 lint 警告
- ✅ 30+ 模板覆盖常见场景
- ✅ 开箱即用（无需配置）
- ✅ 完全向后兼容
- ✅ 易于扩展（支持自定义模板）

## 💡 使用建议

### 1. 快速开始

```go
// 最简单的使用方式
dg := common.NewDataGenerator()
value := dg.GenerateValue(col)  // 自动使用对应模板
```

### 2. 生产环境

对于生产环境，建议：
- 使用预定义的真实数据源（而不仅是随机）
- 验证生成的数据符合业务规则
- 对敏感字段（如密码）进行加密处理

### 3. 测试环境

对于测试环境，可以：
- 直接使用预定义模板
- 通过 SetSeed() 固定数据（便于重现 bug）
- 使用自定义模板模拟特殊场景

## 📖 API 参考

### GetAvailableTemplates()

获取所有可用的模板类型列表

```go
templates := common.GetAvailableTemplates()
// 返回: []string{"uuid", "email", "phone", "city", ...}
```

### RegisterTemplate(name string, generator func() any)

注册自定义模板

```go
dg := common.NewDataGenerator()
dg.RegisterTemplate("my_template", func() any {
    return "custom_value"
})
```

### GenerateValue(col Column) any

生成单个值

```go
col := mysql.Column{Name: "email", DataType: "varchar"}
value := dg.GenerateValue(col)  // 返回邮箱
```

## 总结

通过 Template 系统，Kyogre Generator 现在能够生成真实、符合业务场景的测试数据。用户可以：

1. ✅ **开箱即用** - 30+ 预定义模板无需配置
2. ✅ **灵活扩展** - 轻松添加自定义模板
3. ✅ **真实数据** - 生成符合实际业务的数据
4. ✅ **向后兼容** - 完全兼容现有代码
5. ✅ **高效便捷** - 一行代码生成完整的虚拟用户、订单等

---

**创建时间**: 2025-11-09
**模块版本**: v1.0
**维护者**: Kyogre Team
