# Template 系统配置集成指南

## 概述

本指南介绍如何在 Kyogre 的配置文件中使用 Mock 模板系统为数据库列生成真实、有意义的测试数据。

## 快速开始

### 配置文件格式

在 YAML/TOML 配置文件中，为列添加 `mock` 字段来指定使用的模板：

```yaml
metadata:
  mysql:
    databases:
      test_db:
        - table: users
          columns:
            - column: id
              type: primary
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
            - column: real_name
              type: varchar
              mock: real_name
            - column: city
              type: varchar
              mock: city
            - column: address
              type: varchar
              mock: address
            - column: created_at
              type: timestamp
              mock: created_at
          indexes:
            - name: idx_username
              columns:
                - username
              is_unique: true
            - name: idx_email
              columns:
                - email
              is_unique: true
```

## 可用模板列表

### 1. 身份标识类模板 (3 个)

| 模板名 | 说明 | 示例输出 |
|--------|------|--------|
| `uuid` | UUID 标识符 | `550e8400-e29b-41d4-a716-446655440000` |
| `object_id` | MongoDB ObjectID | `0000000167a8f9c35e2b41f9b123456789012345` |
| `snowflake` | 雪花 ID | `1830921458956009472` |

### 2. 用户信息类模板 (6 个)

| 模板名 | 说明 | 示例输出 |
|--------|------|--------|
| `username` | 用户名 | `user_567890`, `player_123456` |
| `email` | 邮箱地址 | `user_567890@qq.com`, `user_123456@gmail.com` |
| `phone` | 手机号码（中国） | `13012345678`, `15587654321` |
| `password` | 密码 | `aB3cD!@xY9Zw` |
| `real_name` | 真实姓名（中文） | `张伟`, `王秀英` |
| `nickname` | 昵称 | `HappyTiger_123`, `LuckyDragon_456` |

### 3. 地址信息类模板 (7 个)

| 模板名 | 说明 | 示例输出 |
|--------|------|--------|
| `country` | 国家 | `China`, `United States`, `Japan` |
| `province` | 省份（中国） | `北京`, `上海`, `广东` |
| `city` | 城市（中国） | `深圳`, `杭州`, `成都` |
| `address` | 详细地址 | `中关村1号写字楼A123室` |
| `zip_code` | 邮编 | `100001`, `200010` |
| `ipv4` | IPv4 地址 | `192.168.1.1`, `10.0.0.1` |
| `ipv6` | IPv6 地址 | `550e:8400:e29b:41d4:a716:4466:5544:0000` |

### 4. 业务数据类模板 (6 个)

| 模板名 | 说明 | 示例输出 |
|--------|------|--------|
| `url` | URL 地址 | `https://example.com/product/12345` |
| `domain` | 域名 | `example.com`, `demo123.io` |
| `company` | 公司名称 | `北京科技有限公司`, `上海互联网有限公司` |
| `product` | 产品名称 | `Premium Phone V5`, `Smart Tablet V3` |
| `money` | 金额（数字） | `5234.56`, `8912.34` |
| `price` | 价格（带符号） | `¥234.56`, `¥789.12` |

### 5. 状态类模板 (3 个)

| 模板名 | 说明 | 示例输出 |
|--------|------|--------|
| `status` | 状态 | `active`, `pending`, `inactive`, `deleted` |
| `gender` | 性别 | `male`, `female`, `other` |
| `boolean` | 布尔值 | `true`, `false` |

### 6. 时间类模板 (3 个)

| 模板名 | 说明 | 示例输出 | 范围 |
|--------|------|--------|------|
| `birthday` | 生日 | `1985-06-15` | 18-78 岁 |
| `created_at` | 创建时间 | `2025-09-20 14:30:45` | 最近 365 天 |
| `updated_at` | 更新时间 | `2025-11-08 10:20:15` | 最近 365 天 |

### 7. 计数器类模板 (2 个)

| 模板名 | 说明 | 特性 |
|--------|------|------|
| `counter` | 自增计数器 | 从 1 开始，每次调用递增 |
| `sequence` | 序列号 | 从 1000000000 开始，每次调用递增 |

## 完整配置示例

### 示例 1：电商用户表

```yaml
metadata:
  mysql:
    databases:
      ecommerce:
        - table: users
          columns:
            - column: id
              type: primary
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
            - column: real_name
              type: varchar
              mock: real_name
            - column: nickname
              type: varchar
              mock: nickname
            - column: gender
              type: varchar
              mock: gender
            - column: birthday
              type: date
              mock: birthday
            - column: city
              type: varchar
              mock: city
            - column: address
              type: varchar_xlarge
              mock: address
            - column: phone_verified
              type: boolean
              mock: boolean
            - column: status
              type: varchar
              mock: status
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
```

### 示例 2：电商产品表

```yaml
metadata:
  mysql:
    databases:
      ecommerce:
        - table: products
          columns:
            - column: id
              type: uuid
              mock: uuid
            - column: product_name
              type: varchar
              mock: product
            - column: company
              type: varchar
              mock: company
            - column: price
              type: decimal
              mock: price
            - column: stock
              type: int
            - column: description
              type: text
            - column: website
              type: varchar
              mock: url
            - column: domain
              type: varchar
              mock: domain
            - column: status
              type: varchar
              mock: status
            - column: created_at
              type: timestamp
              mock: created_at
            - column: updated_at
              type: timestamp_update
              mock: updated_at
          indexes:
            - name: idx_product_name
              columns:
                - product_name
            - name: idx_status
              columns:
                - status
```

### 示例 3：订单表

```yaml
metadata:
  mysql:
    databases:
      ecommerce:
        - table: orders
          columns:
            - column: id
              type: uuid
              mock: uuid
            - column: order_no
              type: varchar
              mock: sequence
            - column: user_id
              type: bigint
            - column: amount
              type: decimal
              mock: money
            - column: status
              type: varchar
              mock: status
            - column: shipping_address
              type: varchar_xlarge
              mock: address
            - column: created_at
              type: timestamp
              mock: created_at
            - column: updated_at
              type: timestamp_update
              mock: updated_at
          indexes:
            - name: idx_user_id
              columns:
                - user_id
            - name: idx_status
              columns:
                - status
            - name: idx_created_at
              columns:
                - created_at
```

## 配置工作流程

### 1. 定义表结构

在配置文件中定义数据库和表的结构：

```yaml
metadata:
  mysql:
    databases:
      myapp:
        - table: users
          columns:
            # ... 列定义
```

### 2. 为列添加 Mock 模板

为需要生成测试数据的列添加 `mock` 字段：

```yaml
- column: email
  type: varchar
  mock: email  # 指定使用 email 模板
```

### 3. 系统自动生成数据

当 DML Generator 生成 INSERT 消息时，它会：
1. 读取列的 Mock 字段
2. 从模板系统中检索对应的生成器
3. 为该列生成对应的测试数据

## 编程使用示例

### 直接使用模板

```go
package main

import (
	"fmt"
	"github.com/xuenqlve/kyogre/pkg/generator/common"
	mysql_schema "github.com/xuenqlve/common/relational_database/mysql"
)

func main() {
	// 创建数据生成器
	dg := common.NewDataGenerator()

	// 获取所有可用模板
	templates := common.GetAvailableTemplates()
	fmt.Printf("Available templates: %v\n", templates)

	// 为特定列生成数据
	col := mysql_schema.Column{
		Name:     "email",
		DataType: "varchar",
	}
	email := dg.GenerateValue(col)
	fmt.Printf("Generated email: %v\n", email)
}
```

### 自定义模板

```go
func main() {
	dg := common.NewDataGenerator()

	// 注册自定义模板
	dg.RegisterTemplate("custom_field", func() any {
		return "CUSTOM_VALUE_123"
	})

	// 使用自定义模板
	col := mysql_schema.Column{Name: "custom_field"}
	value := dg.GenerateValue(col)
	fmt.Printf("Custom value: %v\n", value)
}
```

### 批量生成数据

```go
func main() {
	dg := common.NewDataGenerator()

	col := mysql_schema.Column{
		Name:     "username",
		DataType: "varchar",
	}

	// 批量生成 10 个用户名
	usernames := dg.GenerateBatch(col, 10)
	for i, username := range usernames {
		fmt.Printf("%d: %v\n", i+1, username)
	}
}
```

## 最佳实践

### 1. 选择合适的模板

根据列的实际用途选择对应的模板：

| 列用途 | 推荐模板 |
|--------|--------|
| 用户标识 | `uuid` 或 `snowflake` |
| 邮箱 | `email` |
| 手机 | `phone` |
| 地址 | `address` 或 `city` |
| 时间戳 | `created_at` 或 `updated_at` |

### 2. 保持数据一致性

对于相同含义的列，使用相同的模板：

```yaml
- column: creator_email
  type: varchar
  mock: email  # 与 user.email 使用相同模板

- column: updater_email
  type: varchar
  mock: email  # 保持一致
```

### 3. 处理唯一约束

对于有唯一约束的列（如 username, email），模板会自动生成不同的值：

```yaml
- column: username
  type: varchar
  mock: username  # 每次生成不同的用户名

indexes:
  - name: idx_username
    columns:
      - username
    is_unique: true  # 唯一约束
```

### 4. 组合使用多个模板

在一个表中可以混合使用多个模板：

```yaml
columns:
  - column: user_id
    type: primary
  - column: username
    mock: username   # 模板 1
  - column: email
    mock: email      # 模板 2
  - column: city
    mock: city       # 模板 3
```

## 扩展系统

### 添加新的模板

如果需要新的模板，可以在 `pkg/generator/common/template.go` 中添加：

```go
// 在 registerCustomTemplates 或新的注册函数中
func (dg *DataGenerator) RegisterTemplate("my_custom", func() any {
    // 实现自定义逻辑
    return generateCustomValue()
})
```

### 使用种子确保可复现性

```go
dg := common.NewDataGenerator()
dg.SetSeed(12345)  // 设置相同的种子会生成相同的数据序列

// 现在生成的数据是确定的，可用于测试
```

## 配置验证

系统会自动验证：
- ✅ Mock 字段值是否为有效的模板名
- ✅ 列的数据类型是否与模板兼容
- ✅ 模板是否存在

## 常见问题

### Q1: 如何列出所有可用的模板？

使用 `GetAvailableTemplates()` 函数：

```go
templates := common.GetAvailableTemplates()
for _, t := range templates {
    fmt.Println(t)
}
```

### Q2: 可以为同一个列指定多个模板吗？

不可以。每个列只能指定一个 `mock` 字段。如果需要多种数据类型，可以使用自定义模板。

### Q3: Mock 字段是必须的吗？

不是必须的。如果没有指定 `mock` 字段，系统会根据列的数据类型生成通用的随机值。

### Q4: 计数器和序列号的初始值是多少？

- `counter`: 从 1 开始
- `sequence`: 从 1000000000 开始

两者都是全局的，在同一进程中会持续递增。

### Q5: 如何生成真实的业务数据而不是随机测试数据？

模板系统已经生成的是真实、符合业务规范的数据（如真实的邮箱域名、中国手机号前缀等）。如果需要更具体的数据，可以：
1. 使用自定义模板
2. 从数据库或文件中导入数据
3. 修改模板的数据源

## 性能考虑

- **生成速度**: 每个模板的生成耗时 < 1ms
- **内存占用**: 最小化，使用无状态的生成器
- **并发安全**: 计数器除外，大多数模板是线程安全的

## 总结

Mock 模板系统提供了：
- ✅ 30+ 预定义的常用模板
- ✅ 真实、有意义的测试数据
- ✅ 灵活的自定义扩展机制
- ✅ 配置文件集成支持
- ✅ 高效的数据生成性能
