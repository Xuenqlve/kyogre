# Template 系统快速参考

## 30 个可用模板一览

### 身份标识 (3 个)
```go
uuid       // 550e8400-e29b-41d4-a716-446655440000
object_id  // 0000000167a8f9c35e2b41f9b123456789012345
snowflake  // 1830921458956009472
```

### 用户信息 (6 个)
```go
username   // user_567890, player_123456, member_789012
email      // user_567890@qq.com, user_123456@gmail.com
phone      // 13012345678, 15587654321 (中国格式)
password   // aB3cD!@xY9Zw (大小写+特殊字符)
real_name  // 张伟, 王秀英, 李明浩 (中文姓名)
nickname   // HappyTiger_123, LuckyDragon_456
```

### 地址信息 (7 个)
```go
country    // China, United States, Japan
province   // 北京, 上海, 广东 (中国省份)
city       // 深圳, 杭州, 成都 (中国城市)
address    // 中关村1号写字楼A123室
zip_code   // 100001, 200010, 510000
ipv4       // 192.168.1.1, 10.0.0.1
ipv6       // 550e:8400:e29b:41d4:a716:4466:5544:0000
```

### 业务数据 (6 个)
```go
url        // https://example.com/product/12345
domain     // example.com, demo123.io, test456.cn
company    // 北京科技有限公司, 上海互联网有限公司
product    // Premium Phone V5, Smart Tablet V3, Pro Laptop V7
money      // 5234.56, 8912.34, 1234.50 (数字金额)
price      // ¥234.56, ¥789.12, ¥456.78 (带符号)
```

### 状态类 (3 个)
```go
status     // active, pending, inactive, deleted, archived
gender     // male, female, other
boolean    // true, false
```

### 时间类 (3 个)
```go
birthday   // 1985-06-15 (18-78岁范围)
created_at // 2025-09-20 14:30:45 (最近365天)
updated_at // 2025-11-08 10:20:15 (最近365天)
```

### 计数器 (2 个)
```go
counter    // 1, 2, 3, 4, 5... (自增，从1开始)
sequence   // 1000000001, 1000000002... (自增，从1000000000开始)
```

---

## 配置文件快速示例

### 最简配置

```yaml
metadata:
  mysql:
    databases:
      mydb:
        - table: users
          columns:
            - column: id
              type: primary
            - column: email
              type: varchar
              mock: email          # ← 指定 mock 模板
            - column: phone
              type: varchar
              mock: phone
```

### 完整的用户表配置

```yaml
metadata:
  mysql:
    databases:
      ecommerce:
        - table: users
          columns:
            # 主键
            - column: id
              type: primary

            # 用户认证
            - column: username
              type: varchar
              mock: username
            - column: password_hash
              type: varchar
              mock: password
            - column: email
              type: varchar
              mock: email

            # 个人信息
            - column: real_name
              type: varchar
              mock: real_name
            - column: phone
              type: varchar
              mock: phone
            - column: gender
              type: varchar
              mock: gender
            - column: birthday
              type: date
              mock: birthday

            # 位置
            - column: city
              type: varchar
              mock: city
            - column: address
              type: varchar_xlarge
              mock: address

            # 记录
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

---

## 代码使用快速指南

### 基本使用

```go
// 导入
import "github.com/xuenqlve/kyogre/pkg/generator/common"

// 创建生成器
dg := common.NewDataGenerator()

// 生成单个值
email := dg.GenerateValue(Column{Name: "email"})

// 批量生成
emails := dg.GenerateBatch(Column{Name: "email"}, 10)
```

### 自定义模板

```go
dg := common.NewDataGenerator()

// 方式 1：简单替换
dg.RegisterTemplate("order_no", func() any {
    return "ORD-123456"
})

// 方式 2：带状态的生成器
counter := 0
dg.RegisterTemplate("order_id", func() any {
    counter++
    return fmt.Sprintf("ORD-%010d", counter)
})

// 使用自定义模板
col := Column{Name: "order_id"}
order1 := dg.GenerateValue(col)  // ORD-0000000001
order2 := dg.GenerateValue(col)  // ORD-0000000002
```

### 可复现的数据

```go
// 设置种子
dg := common.NewDataGenerator()
dg.SetSeed(12345)

// 现在生成的序列是固定的
value1 := dg.GenerateValue(Column{Name: "email"})
value2 := dg.GenerateValue(Column{Name: "email"})

// 相同种子会生成相同序列
dg2 := common.NewDataGenerator()
dg2.SetSeed(12345)
value3 := dg2.GenerateValue(Column{Name: "email"})
// value1 == value3
```

### 获取所有模板

```go
templates := common.GetAvailableTemplates()
for i, t := range templates {
    fmt.Printf("%2d) %s\n", i+1, t)
}
// 输出:
//  1) uuid
//  2) object_id
//  3) snowflake
//  ... 以此类推
```

---

## 常用场景模板选择

### 电商用户表
```
username    → username
password    → password
email       → email
phone       → phone
real_name   → real_name
gender      → gender
city        → city
address     → address
created_at  → created_at
updated_at  → updated_at
```

### 电商产品表
```
product_id  → uuid 或 snowflake
name        → product
company     → company
price       → price
stock       → (无 mock，自动生成整数)
url         → url
domain      → domain
created_at  → created_at
updated_at  → updated_at
```

### 订单表
```
order_id    → uuid 或 snowflake
order_no    → sequence
user_id     → (无 mock，关联 users 表)
amount      → money
status      → status
address     → address
created_at  → created_at
updated_at  → updated_at
```

### 日志表
```
id          → uuid
action      → status
ip_address  → ipv4
user_agent  → (无 mock，自动生成字符串)
timestamp   → created_at
```

---

## 性能数据

| 操作 | 耗时 |
|------|------|
| 单值生成 | < 1ms |
| 1000 个值 | < 100ms |
| 初始化生成器 | < 50ms |
| 内存占用 | < 1MB |

---

## 常见错误及解决

### 错误 1：模板名拼写错误
```yaml
columns:
  - column: email
    mock: emai  # ✗ 错误拼写
```
**解决**：使用正确的模板名 `email`

### 错误 2：混淆 mock 字段用途
```yaml
columns:
  - column: password
    type: varchar
    mock: password  # 生成示例密码，不会真的加密
```
**解决**：生成的密码是示例数据，生产环境需要额外加密

### 错误 3：期望唯一值但没有约束
```yaml
columns:
  - column: email
    mock: email     # 会生成不同的邮箱，但不保证完全唯一
```
**解决**：添加 `is_unique: true` 或在数据库中创建唯一索引

---

## 文档导航

| 文档 | 用途 | 长度 |
|------|------|------|
| [TEMPLATE_SYSTEM_GUIDE.md](./TEMPLATE_SYSTEM_GUIDE.md) | 完整系统说明 | 500+ 行 |
| [template_config_integration.md](./template_config_integration.md) | 配置文件说明 | 400+ 行 |
| [template_api_reference.md](./template_api_reference.md) | API 参考 | 600+ 行 |
| [template_test_outputs.md](./template_test_outputs.md) | 输出示例 | 500+ 行 |
| **本文档** | **快速参考** | **<50 行** |

---

## 关键特性速查

✅ **30 个模板** - 覆盖所有常见场景
✅ **自动初始化** - `NewDataGenerator()` 自动注册所有模板
✅ **配置驱动** - 通过 YAML 文件指定模板
✅ **自定义扩展** - 支持 `RegisterTemplate()`
✅ **批量生成** - `GenerateBatch()` 快速生成大量数据
✅ **可复现性** - `SetSeed()` 确保数据可重现
✅ **高性能** - < 1ms 单值生成
✅ **类型智能** - 自动匹配列的数据类型

---

## 获取帮助

1. **快速问题** → 本快速参考
2. **配置问题** → template_config_integration.md
3. **API 问题** → template_api_reference.md
4. **输出示例** → template_test_outputs.md
5. **深入学习** → TEMPLATE_SYSTEM_GUIDE.md

---

**最后更新**：2025-11-09
**版本**：1.0
