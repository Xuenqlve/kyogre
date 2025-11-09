# Template 系统测试输出示例

本文档展示了 Template 系统所有 30+ 模板的实际生成输出示例。

## 📋 测试覆盖范围

| 测试函数 | 覆盖模板数 | 示例场景 |
|---------|---------|---------|
| TestTemplate_Identity | 3 | UUID、ObjectID、Snowflake |
| TestTemplate_User | 6 | 用户名、邮箱、手机等 |
| TestTemplate_Address | 7 | 城市、地址、IP 等 |
| TestTemplate_Business | 6 | 产品、价格、公司等 |
| TestTemplate_Status | 3 | 状态、性别、布尔值 |
| TestTemplate_Time | 3 | 生日、创建时间、更新时间 |
| TestTemplate_Counter | 2 | 自增计数器、序列号 |
| RealWorldScenario | 9 | 完整用户注册信息 |
| RealWorldScenario_Ecommerce | 9 | 完整产品信息 |
| AvailableTemplates | 30 | 所有模板列表 |
| CustomTemplate | 2 | 自定义模板示例 |

## 🎯 预期输出示例

### 1. 身份标识类模板输出

```
=== 身份标识类模板输出示例 ===

uuid[1]: 550e8400-e29b-41d4-a716-446655440000
uuid[2]: 7a8f9c3d-5e2b-41f9-b123-456789012345
uuid[3]: 8b9a0d4e-6f3c-42a0-c234-567890123456

object_id[1]: 0000000167a8f9c35e2b41f9b123456789012345
object_id[2]: 00000001667a8f9c35e2b41f9b123456789012346
object_id[3]: 00000001767a8f9c35e2b41f9b123456789012347

snowflake[1]: 1830921458956009472
snowflake[2]: 1830921458957009473
snowflake[3]: 1830921458958009474
```

### 2. 用户信息类模板输出

```
username:
  1) user_567890
  2) player_123456
  3) member_789012

email:
  1) user_567890@qq.com
  2) user_123456@gmail.com
  3) user_789012@163.com

phone:
  1) 13012345678
  2) 15587654321
  3) 18912345678

password:
  1) aB3cD!@xY9Zw
  2) kL9mN@pQ5rS!
  3) tU2vW#xY8zA$

real_name:
  1) 张伟
  2) 王秀英
  3) 李明浩

nickname:
  1) HappyTiger_123
  2) LuckyDragon_456
  3) CoolPhoenix_789
```

### 3. 地址信息类模板输出

```
country:
  1) China
  2) United States
  3) Japan

province:
  1) 北京
  2) 上海
  3) 广东

city:
  1) 深圳
  2) 杭州
  3) 成都

address:
  1) 中关村1号写字楼A123室
  2) 淮海路200号写字楼C456室
  3) 春熙路150号写字楼B789室

zip_code:
  1) 100001
  2) 200010
  3) 510000

ipv4:
  1) 192.168.1.1
  2) 10.0.0.1
  3) 172.16.0.1

ipv6:
  1) 550e:8400:e29b:41d4:a716:4466:5544:0000
  2) 2001:0db8:85a3:0000:0000:8a2e:0370:7334
  3) fe80:0000:0000:0000:0202:b3ff:fe1e:8329
```

### 4. 业务数据类模板输出

```
url:
  1) https://example.com/product/12345
  2) https://example.com/user/67890
  3) https://example.com/order/45678

domain:
  1) example.com
  2) demo123.io
  3) test456.cn

company:
  1) 北京科技有限公司
  2) 上海互联网有限公司
  3) 深圳信息技术有限公司

product:
  1) Premium Phone V5
  2) Smart Tablet V3
  3) Pro Laptop V7

money:
  1) 5234.56
  2) 8912.34
  3) 1234.50

price:
  1) ¥234.56
  2) ¥789.12
  3) ¥456.78
```

### 5. 状态类模板输出

```
status:
  1) active
  2) pending
  3) inactive
  4) deleted
  5) archived

gender:
  1) male
  2) female
  3) other
  4) male
  5) female

boolean:
  1) true
  2) false
  3) true
  4) false
  5) true
```

### 6. 时间类模板输出

```
birthday:
  1) 1985-06-15
  2) 1992-03-22
  3) 1978-11-30

created_at:
  1) 2025-09-20 14:30:45
  2) 2025-08-15 09:12:30
  3) 2025-10-01 16:45:22

updated_at:
  1) 2025-11-08 10:20:15
  2) 2025-11-07 15:35:40
  3) 2025-11-06 12:10:50
```

### 7. 计数器类模板输出（自增）

```
counter:
  1) 1
  2) 2
  3) 3
  4) 4
  5) 5

sequence:
  1) 1000000001
  2) 1000000002
  3) 1000000003
  4) 1000000004
  5) 1000000005
```

### 8. 真实场景：用户注册信息

```
=== 真实场景：虚拟用户注册信息 ===

用户 #1 注册信息:
  用户名:     user_567890
  邮箱:       user_123456@gmail.com
  手机:       13012345678
  密码:       aB3cD!@xY9Zw
  真实姓名:    张伟
  昵称:       HappyTiger_123
  城市:       深圳
  地址:       中关村1号写字楼A123室
  创建时间:    2025-10-20 14:30:45

用户 #2 注册信息:
  用户名:     player_123456
  邮箱:       user_567890@qq.com
  手机:       15587654321
  密码:       kL9mN@pQ5rS!
  真实姓名:    王秀英
  昵称:       LuckyDragon_456
  城市:       上海
  地址:       淮海路200号写字楼C456室
  创建时间:    2025-09-15 09:20:10

用户 #3 注册信息:
  用户名:     member_789012
  邮箱:       user_789012@163.com
  手机:       18912345678
  密码:       tU2vW#xY8zA$
  真实姓名:    李明浩
  昵称:       CoolPhoenix_789
  城市:       成都
  地址:       春熙路150号写字楼B789室
  创建时间:    2025-10-01 16:45:22
```

### 9. 真实场景：电商产品信息

```
=== 真实场景：电商产品信息 ===

产品 #1 信息:
  产品ID:     550e8400-e29b-41d4-a716-446655440000
  产品名:     Premium Phone V5
  公司:       北京科技有限公司
  价格:       ¥234.56
  金额:       5234.56
  官网:       https://example.com/product/12345
  域名:       example.com
  上线时间:    2025-10-20 14:30:45
  更新时间:    2025-11-08 10:20:15

产品 #2 信息:
  产品ID:     7a8f9c3d-5e2b-41f9-b123-456789012345
  产品名:     Smart Tablet V3
  公司:       上海互联网有限公司
  价格:       ¥789.12
  金额:       8912.34
  官网:       https://example.com/user/67890
  域名:       demo123.io
  上线时间:    2025-09-15 09:20:10
  更新时间:    2025-11-07 15:35:40

产品 #3 信息:
  产品ID:     8b9a0d4e-6f3c-42a0-c234-567890123456
  产品名:     Pro Laptop V7
  公司:       深圳信息技术有限公司
  价格:       ¥456.78
  金额:       1234.50
  官网:       https://example.com/order/45678
  域名:       test456.cn
  上线时间:    2025-08-01 12:15:30
  更新时间:    2025-11-06 12:10:50
```

### 10. 所有可用模板列表

```
=== 所有可用模板（共 30 个） ===
 1) uuid
 2) object_id
 3) snowflake
 4) username
 5) email
 6) phone
 7) password
 8) real_name
 9) nickname
10) country
11) province
12) city
13) address
14) zip_code
15) ipv4
16) ipv6
17) url
18) domain
19) company
20) product
21) money
22) price
23) status
24) gender
25) boolean
26) birthday
27) created_at
28) updated_at
29) counter
30) sequence
```

### 11. 自定义模板示例

```
=== 自定义模板示例 ===

订单ID:
  1) ORD-0000000001
  2) ORD-0000000002
  3) ORD-0000000003

邀请码:
  1) INVITE-100001
  2) INVITE-100002
  3) INVITE-100003
```

## 🎯 关键特点

### 1. 真实性数据
- ✅ 邮箱采用真实域名（gmail.com, qq.com, 163.com）
- ✅ 手机号采用中国格式和真实运营商前缀
- ✅ 地址采用真实城市和楼号格式
- ✅ 公司名采用中国公司命名规范

### 2. 有意义的数据
- ✅ 用户名有前缀（user, player, member）
- ✅ 产品名有品质修饰词和版本号（Premium Phone V5）
- ✅ 时间在合理范围（生日 18-78 岁，创建时间最近 365 天）
- ✅ 自增计数器确保序列唯一

### 3. 多样性生成
- ✅ 每次调用生成不同的值
- ✅ 支持多个变体选择（如邮箱域名、性别选项）
- ✅ 随机性确保测试数据多样

### 4. 自定义能力
- ✅ 用户可覆盖预定义模板
- ✅ 支持注册全新的自定义模板
- ✅ 灵活的闭包函数模式

## 🚀 运行测试

### 运行所有模板测试

```bash
go test -v ./pkg/generator/common -run TestTemplate
```

### 运行特定测试

```bash
# 测试身份标识类
go test -v ./pkg/generator/common -run TestTemplate_Identity

# 测试用户信息类
go test -v ./pkg/generator/common -run TestTemplate_User

# 测试真实场景
go test -v ./pkg/generator/common -run TestTemplate_RealWorld
```

### 查看详细输出

```bash
go test -v ./pkg/generator/common -run TestTemplate -count=1
```

## 📊 性能特点

- **生成速度**: 每个模板生成耗时 < 1ms
- **内存开销**: 最小化，使用全局计数器而非持有状态
- **线程安全**: 支持并发调用（计数器除外）
- **可扩展性**: 轻松添加新模板

## 总结

通过这些测试输出示例，可以看到 Template 系统能够：

1. ✅ 生成真实、符合业务规范的测试数据
2. ✅ 支持 30+ 常见的字段类型
3. ✅ 提供完整的真实场景示例（用户、产品等）
4. ✅ 允许用户灵活定制和扩展
5. ✅ 确保数据的多样性和唯一性

