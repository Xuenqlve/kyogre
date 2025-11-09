# Kyogre Template 系统完整指南

## 目录

1. [概述](#概述)
2. [快速开始](#快速开始)
3. [核心概念](#核心概念)
4. [模板详解](#模板详解)
5. [配置集成](#配置集成)
6. [API 参考](#api-参考)
7. [最佳实践](#最佳实践)
8. [常见问题](#常见问题)

---

## 概述

**Template 系统**是 Kyogre 的数据生成模块，提供了 30+ 预定义的、符合业务规范的测试数据生成模板。

### 主要特性

| 特性 | 说明 |
|------|------|
| **30+ 预定义模板** | 覆盖身份标识、用户信息、地址、业务数据等 |
| **真实数据生成** | 生成符合业务规范的数据（如真实邮箱域名、中国手机号等） |
| **配置文件集成** | 通过 YAML/TOML 配置文件指定模板 |
| **自定义扩展** | 支持注册自定义模板 |
| **高性能** | 每个模板生成耗时 < 1ms |
| **可复现性** | 支持种子设置确保数据可复现 |

### 设计目标

- ✅ 提供真实、有意义的测试数据
- ✅ 减少手工编写测试数据的工作
- ✅ 支持大规模数据生成
- ✅ 易于扩展和定制
- ✅ 性能高效

---

## 快速开始

### 1. 最简单的使用方式

```go
package main

import (
    "fmt"
    "github.com/xuenqlve/kyogre/pkg/generator/common"
    mysql_schema "github.com/xuenqlve/common/relational_database/mysql"
)

func main() {
    // 创建生成器
    dg := common.NewDataGenerator()

    // 生成邮箱
    email := dg.GenerateValue(mysql_schema.Column{Name: "email"})
    fmt.Println(email)  // 输出: user_567890@qq.com

    // 生成手机号
    phone := dg.GenerateValue(mysql_schema.Column{Name: "phone"})
    fmt.Println(phone)  // 输出: 13012345678
}
```

### 2. 在配置文件中使用

```yaml
metadata:
  mysql:
    databases:
      test_db:
        - table: users
          columns:
            - column: id
              type: primary
            - column: email
              type: varchar
              mock: email           # 指定使用 email 模板
            - column: phone
              type: varchar
              mock: phone           # 指定使用 phone 模板
            - column: created_at
              type: timestamp
              mock: created_at      # 指定使用 created_at 模板
```

### 3. 查看所有可用模板

```go
templates := common.GetAvailableTemplates()
for i, t := range templates {
    fmt.Printf("%2d) %s\n", i+1, t)
}
```

---

## 核心概念

### 数据生成流程

```
列定义 (Column)
    ↓
GenerateValue()
    ↓
检查自定义模板 → 如果存在，使用自定义模板
    ↓ (不存在)
检查内置模板 (按列名) → 如果存在，使用内置模板
    ↓ (不存在)
根据数据类型生成通用值
    ↓
返回生成的值
```

### 三层优先级

1. **自定义模板** (最高优先级)
   ```go
   dg.RegisterTemplate("email", myCustomEmailGenerator)
   ```

2. **内置模板** (中等优先级)
   ```go
   // 根据列名 "email" 自动匹配 email 模板
   dg.GenerateValue(Column{Name: "email"})
   ```

3. **类型默认生成** (最低优先级)
   ```go
   // 根据 DataType "varchar" 生成通用字符串
   dg.GenerateValue(Column{DataType: "varchar"})
   ```

### 模板与列的绑定方式

```
配置文件: Mock 字段
    ↓
读取 Mock 字段值 (如 "email")
    ↓
查找对应的模板
    ↓
使用模板的生成函数
    ↓
生成数据
```

---

## 模板详解

### 1. 身份标识类 (3 个)

用途：生成唯一标识符

| 模板 | 格式 | 示例 | 用途 |
|------|------|------|------|
| uuid | 标准 UUID | `550e8400-e29b-41d4-a716-446655440000` | 通用唯一标识 |
| object_id | MongoDB ObjectID | `0000000167a8f9c35e2b41f9b123456789012345` | 分布式ID |
| snowflake | 雪花算法 | `1830921458956009472` | 高性能ID |

**选择建议：**
- 通用场景 → uuid
- MongoDB 集成 → object_id
- 高并发 → snowflake

### 2. 用户信息类 (6 个)

用途：生成用户相关数据

| 模板 | 格式 | 示例 | 特点 |
|------|------|------|------|
| username | 前缀+数字 | `user_567890`, `admin_123456` | 多个前缀变体 |
| email | 用户名@域名 | `user_567890@qq.com` | 真实邮箱域名 |
| phone | 国家代码+11位数字 | `13012345678` | 中国手机号格式 |
| password | 大小写字母+数字+特殊字符 | `aB3cD!@xY9Zw` | 符合安全规范 |
| real_name | 中文姓名 | `张伟`, `王秀英` | 真实中文姓名 |
| nickname | 英文昵称 | `HappyTiger_123`, `LuckyDragon_456` | 创意性名称 |

**应用场景：**
```yaml
columns:
  - column: username
    mock: username     # 用户登录名
  - column: email
    mock: email        # 联系邮箱
  - column: phone
    mock: phone        # 联系电话
  - column: password
    mock: password     # 密码（已加密存储）
  - column: display_name
    mock: real_name    # 显示名称
```

### 3. 地址信息类 (7 个)

用途：生成地址相关数据

| 模板 | 示例 | 数据源 | 特点 |
|------|------|--------|------|
| country | China, USA, Japan | 国家列表 | 多语言 |
| province | 北京, 上海, 广东 | 中国省份 | 真实行政区划 |
| city | 深圳, 杭州, 成都 | 中国主要城市 | 多样性 |
| address | 中关村1号写字楼A123室 | 格式化生成 | 真实感强 |
| zip_code | 100001, 200010 | 真实邮编 | 区域对应 |
| ipv4 | 192.168.1.1 | 标准格式 | 有效IP范围 |
| ipv6 | 550e:8400:e29b:41d4:... | 标准格式 | 完整地址 |

**组合使用：**
```go
// 生成完整地址
province := dg.GenerateValue(Column{Name: "province"})    // 北京
city := dg.GenerateValue(Column{Name: "city"})            // 深圳
address := dg.GenerateValue(Column{Name: "address"})      // 中关村1号...
```

### 4. 业务数据类 (6 个)

用途：生成业务相关的数据

| 模板 | 示例 | 格式 | 业务含义 |
|------|------|------|---------|
| url | https://example.com/product/12345 | 完整 URL | 资源链接 |
| domain | example.com, demo123.io | 域名 | 网站域名 |
| company | 北京科技有限公司 | 公司名 | 公司信息 |
| product | Premium Phone V5, Smart Tablet V3 | 产品名 | 产品名称 |
| money | 5234.56, 8912.34 | 数字金额 | 交易金额 |
| price | ¥234.56, ¥789.12 | 带符号价格 | 展示价格 |

**电商场景：**
```yaml
columns:
  - column: product_name
    mock: product      # 产品名
  - column: price
    mock: price        # 售价（带符号）
  - column: cost
    mock: money        # 成本（数字）
  - column: company
    mock: company      # 供应商
  - column: website
    mock: url          # 详情链接
```

### 5. 状态类 (3 个)

用途：生成枚举状态值

| 模板 | 可能值 | 用途 | 特点 |
|------|--------|------|------|
| status | active, pending, inactive, deleted | 业务状态 | 多个有效值 |
| gender | male, female, other | 性别 | 真实分布 |
| boolean | true, false | 布尔值 | 等概率分布 |

**使用场景：**
```yaml
columns:
  - column: status
    mock: status       # 订单状态
  - column: is_active
    mock: boolean      # 激活状态
  - column: gender
    mock: gender       # 性别
```

### 6. 时间类 (3 个)

用途：生成时间相关数据

| 模板 | 格式 | 范围 | 说明 |
|------|------|------|------|
| birthday | YYYY-MM-DD | 18-78 岁 | 合理的年龄范围 |
| created_at | YYYY-MM-DD HH:MM:SS | 最近 365 天 | 创建时间戳 |
| updated_at | YYYY-MM-DD HH:MM:SS | 最近 365 天 | 更新时间戳 |

**时间管理：**
```yaml
columns:
  - column: birth_date
    type: date
    mock: birthday     # 生日（年-月-日）
  - column: created_at
    type: timestamp
    mock: created_at   # 创建时间（带时间）
  - column: updated_at
    type: timestamp_update
    mock: updated_at   # 更新时间（带时间）
```

### 7. 计数器类 (2 个)

用途：生成自增序列

| 模板 | 起始值 | 增量 | 特点 |
|------|--------|------|------|
| counter | 1 | 每次 +1 | 简单计数器 |
| sequence | 1000000000 | 每次 +1 | 大数字起点 |

**特殊性：**
```go
// 全局共享，在同一进程中持续递增
dg1 := NewDataGenerator()
dg1.GenerateValue(Column{Name: "counter"})  // 返回 1
dg1.GenerateValue(Column{Name: "counter"})  // 返回 2

dg2 := NewDataGenerator()
dg2.GenerateValue(Column{Name: "counter"})  // 返回 3（继续递增）
```

---

## 配置集成

### 配置文件示例

```yaml
# 完整的表配置示例
metadata:
  mysql:
    databases:
      ecommerce:
        - table: users
          columns:
            # 主键
            - column: id
              type: primary

            # 用户信息
            - column: username
              type: varchar
              mock: username
            - column: email
              type: varchar
              mock: email
            - column: phone
              type: varchar
              mock: phone
            - column: password
              type: varchar
              mock: password

            # 个人信息
            - column: real_name
              type: varchar
              mock: real_name
            - column: nickname
              type: varchar
              mock: nickname
            - column: gender
              type: varchar
              mock: gender
            - column: birth_date
              type: date
              mock: birthday

            # 地址信息
            - column: country
              type: varchar
              mock: country
            - column: province
              type: varchar
              mock: province
            - column: city
              type: varchar
              mock: city
            - column: address
              type: varchar_xlarge
              mock: address

            # 状态
            - column: status
              type: varchar
              mock: status
            - column: is_verified
              type: boolean
              mock: boolean

            # 时间戳
            - column: created_at
              type: timestamp
              mock: created_at
            - column: updated_at
              type: timestamp_update
              mock: updated_at

          indexes:
            - name: idx_username
              columns:
                - username
              is_unique: true
            - name: idx_email
              columns:
                - email
              is_unique: true
            - name: idx_status
              columns:
                - status
```

---

## API 参考

### 主要函数

```go
// 创建生成器
dg := common.NewDataGenerator()

// 生成单个值
value := dg.GenerateValue(Column{Name: "email"})

// 批量生成
values := dg.GenerateBatch(Column{Name: "username"}, 10)

// 设置种子（确保可复现）
dg.SetSeed(12345)

// 注册自定义模板
dg.RegisterTemplate("my_field", func() any {
    return "custom_value"
})

// 获取所有模板
templates := common.GetAvailableTemplates()
```

详见 [API 参考文档](./template_api_reference.md)

---

## 最佳实践

### 1. 命名规范

为列和模板使用清晰的名称：

```yaml
columns:
  - column: email              # ✓ 清晰表达含义
    mock: email

  - column: user_phone         # ✓ 包含关键词
    mock: phone

  - column: f1                 # ✗ 不清晰
    mock: email
```

### 2. 模板选择

根据列的实际含义选择合适的模板：

```yaml
columns:
  - column: creator_email
    mock: email           # ✓ 正确：含义为邮箱

  - column: creator_name
    mock: real_name       # ✓ 正确：含义为人名

  - column: creator_id
    mock: uuid            # ✓ 正确：含义为标识符
```

### 3. 一致性

对于相同类型的数据，保持模板一致：

```yaml
# ✓ 好的做法
columns:
  - column: user_email
    mock: email
  - column: admin_email
    mock: email      # 保持一致
  - column: support_email
    mock: email      # 保持一致

# ✗ 不好的做法
columns:
  - column: user_email
    mock: email
  - column: admin_email
    mock: username   # 不一致，应使用 email
```

### 4. 组合使用

在一个表中灵活组合多个模板：

```yaml
columns:
  # 身份识别
  - column: id
    type: primary

  # 用户认证
  - column: username
    mock: username
  - column: password_hash
    mock: password

  # 用户资料
  - column: email
    mock: email
  - column: phone
    mock: phone
  - column: real_name
    mock: real_name

  # 位置信息
  - column: city
    mock: city
  - column: address
    mock: address

  # 记录管理
  - column: created_at
    mock: created_at
  - column: updated_at
    mock: updated_at
```

### 5. 处理特殊需求

对于模板无法覆盖的特殊需求，使用自定义模板：

```go
dg := common.NewDataGenerator()

// 示例 1：订单号生成
orderNo := 0
dg.RegisterTemplate("order_no", func() any {
    orderNo++
    return fmt.Sprintf("ORD-%d-%d", time.Now().Year(), orderNo)
})

// 示例 2：邀请码生成
dg.RegisterTemplate("invite_code", func() any {
    return fmt.Sprintf("INV%08d", rand.Intn(100000000))
})

// 示例 3：自定义范围的数字
dg.RegisterTemplate("score", func() any {
    return dg.GenerateRandomInt(0, 100)
})
```

---

## 常见问题

### Q1: 模板生成的数据是真实的吗？

**A:** 是的，但仅用于测试。所有生成的数据都符合业务规范：
- 邮箱使用真实域名（gmail.com, qq.com 等）
- 手机号使用真实的中国运营商前缀
- 地址使用真实的城市名称
- 但具体的人名、地址等是生成的，非真实个人数据

### Q2: 可以在生产环境使用这个系统吗？

**A:** 不建议。这个系统设计用于测试和压力测试，不适合直接用于生产数据。如果需要生产数据，应该：
- 使用真实数据导入
- 实施数据脱敏
- 遵守隐私法规（如 GDPR）

### Q3: 如何确保生成的数据唯一？

**A:** 大多数模板会自动生成不同的值。对于需要保证唯一性的字段（如 uuid, counter），可以：
1. 在数据库中添加唯一索引
2. 在应用层进行去重
3. 使用计数器模板确保递增唯一

### Q4: 生成的数据可以复现吗？

**A:** 可以。使用 `SetSeed()` 方法：

```go
dg := NewDataGenerator()
dg.SetSeed(12345)
// 现在生成的序列是确定的
```

### Q5: 支持并发使用吗？

**A:** 大部分情况下支持，但需要注意：
- 计数器模板（counter, sequence）不是线程安全的
- 建议每个 goroutine 使用独立的 DataGenerator 实例
- 或在使用前加锁

### Q6: 如何添加新的模板？

**A:** 修改 `pkg/generator/common/template.go`：

```go
// 在对应的注册函数中添加
dg.RegisterTemplate("my_template", func() any {
    // 实现生成逻辑
    return generateValue()
})
```

### Q7: 模板的生成速度如何？

**A:** 非常快：
- 单个值：< 1ms
- 批量生成 1000 个值：< 100ms
- 平均每个值：< 0.1ms

### Q8: 内存占用多少？

**A:** 非常低：
- 生成器初始化：< 1MB
- 生成过程中无大额外分配
- 批量生成时内存占用线性增长（取决于批量大小）

---

## 文档索引

- [配置集成指南](./template_config_integration.md) - 详细的配置文件使用说明
- [API 参考](./template_api_reference.md) - 完整的 API 文档
- [测试输出示例](./template_test_outputs.md) - 各模板的输出示例

---

## 总结

Template 系统是 Kyogre 的强大功能，可以：

✅ **提高生产力**：自动生成真实感强的测试数据
✅ **确保数据质量**：生成符合业务规范的数据
✅ **支持大规模测试**：高效生成大量数据
✅ **易于扩展**：灵活的自定义机制
✅ **性能优秀**：毫秒级的生成速度

通过合理使用 Template 系统，可以大幅提升数据库压力测试的效率和质量。
