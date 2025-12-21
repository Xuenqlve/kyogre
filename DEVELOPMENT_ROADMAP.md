# Kyogre 项目开发路线图

**最后更新**: 2025-12-20
**项目状态**: Alpha版本 (62% 完成)
**优先级**: P0缺陷需立即修复，2-3天内完成基础集成

---

## 📊 项目评估速览

| 指标 | 评分 | 说明 |
|------|------|------|
| 架构设计 | ⭐⭐⭐⭐⭐ | 工厂模式应用完美 |
| 代码完整性 | 62% | 框架完整，执行缺失 |
| **可运行性** | ❌ | **3个P0缺陷阻塞** |
| 测试覆盖 | 60% | 26个测试，缺集成测试 |

---

## 🔴 P0 致命缺陷清单

必须在2-3天内修复，否则项目无法运行。

### 缺陷1: Server.Run() 为空实现
- **文件**: `internal/app/app.go:51`
- **当前**: `func (s *Server) Run() error { return nil }`
- **影响**: 程序启动后立即退出，无任何执行
- **修复时间**: 2-4小时
- **优先级**: 🔴🔴🔴 (最高)

**需要实现**:
1. 调用 `Metadata.Initialize()` 创建表
2. 启动所有组件 (Generator, Pressure, IQuery)
3. 管理生成→反查→执行的消息流
4. 处理优雅关闭

### 缺陷2: Metadata.Initialize() 缺实现
- **文件**: `internal/plugin/metadata/metadata.go`
- **当前**: 只有接口定义，无具体实现
- **影响**: 无法创建测试数据库和表
- **修复时间**: 1-2小时
- **优先级**: 🔴🔴 (高)

**需要实现**:
1. 创建数据库
2. 根据Schema创建表
3. 创建索引和主键
4. 初始化种子数据

### 缺陷3: 消息管道缺失
- **文件**: 全局架构
- **当前**: Generator生成的消息无处理流程
- **影响**: 各组件独立，无法协同
- **修复时间**: 3-5小时
- **优先级**: 🔴🔴 (高)

**需要实现**:
1. 定义 `MessageQueue` 接口
2. 实现 `ChannelQueue` 基于channel的队列
3. 在Server中整合消息流处理
4. 处理队列背压

---

## 📋 分阶段任务清单

### 第一阶段：P0 基础集成 (2-3天) ⏰ **立即开始**

#### Week1-Day1: 实现Server.Run()
```
子任务:
  ✅ 定义ServerState结构体
  ✅ 实现initializeMetadata()
  ✅ 实现initializeComponents()
  ✅ 实现startComponents()
  ✅ 实现generatorLoop()
  ✅ 实现messageProcessor()
  ✅ 实现shutdown()
  ✅ 编写单元测试

所需时间: 2-4小时
相关文件: internal/app/app.go
验证: go test -v -run TestServerRun ./test/...
```

#### Week1-Day1/2: 实现Metadata.Initialize()
```
子任务:
  ✅ 创建MySQLMetadata结构体
  ✅ 实现Configure()方法
  ✅ 实现Initialize()方法
  ✅ 实现createDatabase()
  ✅ 实现createTable()
  ✅ 实现insertSeedData()
  ✅ 编写单元测试

所需时间: 1-2小时
相关文件: 新建internal/plugin/metadata/mysql_metadata.go
验证: go test -v -run TestMetadataInitialize ./test/metadata/...
```

#### Week1-Day2: 实现消息管道
```
子任务:
  ✅ 定义MessageQueue接口
  ✅ 实现ChannelQueue类型
  ✅ 实现Send()方法
  ✅ 实现Recv()方法
  ✅ 集成到Server中
  ✅ 处理背压逻辑
  ✅ 编写单元测试

所需时间: 3-5小时
相关文件: 新建internal/message/queue.go
验证: go test -v -run TestMessageQueue ./test/...
```

#### Week1-Day3: 添加配置示例
```
子任务:
  ✅ 创建examples/目录
  ✅ 编写examples/config.yaml
  ✅ 编写examples/schema.yaml
  ✅ 编写examples/README.md
  ✅ 创建快速开始指南

所需时间: 1-2小时
相关文件: 新建examples/config.yaml等
验证: 文件是否正确，配置是否完整
```

**P0完成的预期结果**:
- ✅ 程序可以启动
- ✅ 自动创建测试表
- ✅ 生成消息并执行压力测试
- ✅ 完整的执行流程贯通

---

### 第二阶段：P1 功能完善 (5-7天) ⏰ **第2-3周**

#### Week2-Day1/2: 完成Generator DDL实现
```
子任务:
  ⬜ 创建MySQLDDLGenerator类
  ⬜ 实现Configure()方法
  ⬜ 实现CollectDependencies()方法
  ⬜ 实现MockMessage()方法
  ⬜ 支持ALTER、CREATE、DROP、RENAME
  ⬜ 编写单元测试

所需时间: 3-4小时
相关文件: 新建pkg/pressure/mysql-ddl/generator.go
```

#### Week2-Day2/3: 实现Metrics收集系统
```
子任务:
  ⬜ 实现DefaultMetrics类
  ⬜ RecordOperation()方法
  ⬜ RecordError()方法
  ⬜ 计算P50/P90/P99/P999延迟
  ⬜ 统计QPS、成功率、失败率
  ⬜ 编写单元测试

所需时间: 4-5小时
相关文件: 完善internal/metrics/metrics.go
```

#### Week2-Day4/5: 改进错误处理和日志
```
子任务:
  ⬜ 定义结构化错误类型
  ⬜ 添加详细错误日志
  ⬜ 实现优雅降级
  ⬜ 添加日志级别控制

所需时间: 2-3小时
相关文件: internal/app/app.go, internal/config/config.go
```

#### Week3-Day1/2: 编写集成测试
```
子任务:
  ⬜ 创建test/integration/目录
  ⬜ 编写完整的端到端测试
  ⬜ 编写多数据源集成测试
  ⬜ 编写并发场景测试
  ⬜ 编写失败恢复测试

所需时间: 4-5小时
相关文件: 新建test/integration/...
验证: go test -v ./test/integration/...
```

**P1完成的预期结果**:
- ✅ 支持DDL操作
- ✅ 收集性能指标
- ✅ 完整的集成测试
- ✅ 错误处理和日志完善

---

### 第三阶段：P2 扩展功能 (10-15天) ⏰ **第4-6周**

#### Week4-5: 实现其他数据库压力引擎
```
子任务:
  ⬜ Redis压力引擎 (3小时)
  ⬜ MongoDB压力引擎 (3小时)
  ⬜ ClickHouse压力引擎 (3小时)
  ⬜ Kafka压力引擎 (3小时)
  ⬜ 各引擎的单元测试 (4小时)

所需时间: 8-10小时
相关文件: pkg/pressure/redis/, pkg/pressure/mongodb/等
```

#### Week5-6: 添加监控和报告系统
```
子任务:
  ⬜ 实时监控数据收集
  ⬜ HTML报告生成
  ⬜ 性能对比分析
  ⬜ 瓶颈识别
  ⬜ 优化建议生成

所需时间: 8-10小时
相关文件: 新建internal/report/, internal/monitor/
```

#### Week6+: 性能优化和维护
```
子任务:
  ⬜ 内存优化 (goroutine管理)
  ⬜ 并发优化 (锁竞争)
  ⬜ 连接复用
  ⬜ 缓存优化
  ⬜ 基准测试

所需时间: 8-10小时
```

**P2完成的预期结果**:
- ✅ 支持6种数据库的压力测试
- ✅ 完整的性能报告
- ✅ 实时监控能力
- ✅ 生产级性能

---

## 📁 文件修改检查清单

### 需要修改的文件

- [ ] **internal/app/app.go**
  - 实现 `Run()` 方法
  - 添加 `ServerState` 结构体
  - 添加初始化和启动方法

- [ ] **internal/config/config.go**
  - 完善参数解析
  - 支持更多配置选项
  - 添加配置验证

### 需要新建的文件

- [ ] **internal/message/queue.go**
  - `MessageQueue` 接口
  - `ChannelQueue` 实现

- [ ] **internal/plugin/metadata/mysql_metadata.go**
  - `MySQLMetadata` 结构体
  - `Configure()` 和 `Initialize()` 方法

- [ ] **examples/config.yaml**
  - 完整的配置示例

- [ ] **examples/schema.yaml**
  - Schema定义示例

- [ ] **examples/README.md**
  - 快速开始指南

- [ ] **internal/metrics/default.go**
  - `DefaultMetrics` 实现

- [ ] **pkg/pressure/mysql-ddl/generator.go**
  - `MySQLDDLGenerator` 实现

- [ ] **test/integration/server_test.go**
  - 集成测试

### 参考实现文件（保持不变）

- `pkg/pressure/mysql-row/pressure.go` - 参考Pressure实现
- `pkg/data_source/mysql/data_source.go` - 参考DataSource实现
- `test/pressure/mysql_row_test.go` - 参考测试写法

---

## 🧪 测试验证清单

### P0验证
```bash
# 1. 编译验证
go build -o ./bin/kyogre ./cmd/...

# 2. 运行程序
./bin/kyogre -config examples/config.yaml

# 3. 检查数据库表
mysql -h localhost -u root -p123456 kyogre_test -e "SHOW TABLES;"

# 4. 检查是否有执行记录
# 查看日志和压力引擎计数

# 5. 单元测试
go test -v -run TestServer ./test/...
go test -v -run TestMetadata ./test/metadata/...
go test -v -run TestMessageQueue ./test/...
```

### P1验证
```bash
# 6. DDL测试
go test -v -run TestDDLGenerator ./test/...

# 7. Metrics测试
go test -v -run TestMetrics ./test/...

# 8. 集成测试
go test -v ./test/integration/...

# 9. 性能基准
go test -bench=. ./test/...
```

---

## 📈 进度跟踪

### 第一阶段 P0 (目标: 完成日期)
- Week1 Day1: Server.Run() 实现
- Week1 Day2: Metadata.Initialize() 实现
- Week1 Day2: 消息管道实现
- Week1 Day3: 配置示例和文档

**验证检查**: 程序能成功执行一个完整的压力测试周期

### 第二阶段 P1 (目标: Week2-3)
- Week2 Day1-2: DDL生成器
- Week2 Day2-3: Metrics系统
- Week2 Day4-5: 错误处理
- Week3 Day1-2: 集成测试

**验证检查**: 所有新功能都有测试覆盖

### 第三阶段 P2 (目标: Week4-6)
- Week4-5: 其他数据库引擎
- Week5-6: 监控和报告
- Week6+: 性能优化

**验证检查**: 各数据库引擎功能完整，性能达标

---

## 🎓 代码规范和指导

### 命名约定
- 接口: `I开头` 或 `er后缀` (已遵循)
- 结构体: `PascalCase` (已遵循)
- 方法: `mixedCase` (已遵循)
- 常量: `UPPER_CASE` (已遵循)

### 代码风格
```go
// 参考现有实现的风格
// 位置: pkg/pressure/mysql-row/pressure.go

// 使用错误包装
return errors.Trace(err)

// 使用结构化日志
log.Info("message", "key", value)

// 广泛使用context
func (m *Metadata) Initialize(ctx context.Context) error

// 并发安全
mu sync.Mutex
defer mu.Unlock()
```

### 测试要求
- 每个新功能都需要单元测试
- 使用 `TestMain` 进行测试环境设置
- 参考 `test/pressure/mysql_row_test.go` 的写法

---

## 🔍 关键问题和答案

**Q: 需要修改MySQL压力引擎吗?**
A: 不需要，已经90%完整。只需集成到Server中。

**Q: 其他数据源可以先跳过吗?**
A: 是的。先完成MySQL的完整流程，再扩展其他数据源。

**Q: 需要修改IQuery反查吗?**
A: 不需要。已经85%完整。直接使用即可。

**Q: 可以先做P2的功能吗?**
A: 不建议。P0缺陷阻塞整个项目，必须先修复。

**Q: 这个项目能用于生产吗?**
A: 完成P0后可以用于测试。完成P1后可以用于小规模生产。

---

## 📞 帮助资源

### 参考代码位置
- **Pressure实现参考**: `pkg/pressure/mysql-row/pressure.go`
- **DataSource实现参考**: `pkg/data_source/mysql/data_source.go`
- **测试写法参考**: `test/pressure/mysql_row_test.go`
- **配置解析参考**: `internal/config/config.go`

### 学习资源
- CLAUDE.md: 项目架构文档
- go.mod: 依赖清单
- test/: 现有测试示例

### 关键依赖
- github.com/xuenqlve/common: 通用库（日志、错误、schema）
- github.com/mitchellh/mapstructure: 配置解析
- github.com/pingcap/tidb/pkg/parser: SQL解析

---

## ✅ 完成标准

### P0完成标准
- ✅ 程序可启动并运行
- ✅ 自动创建表和数据库
- ✅ 生成消息并执行压力测试
- ✅ 有完整的执行日志
- ✅ 单元测试通过率 >90%

### P1完成标准
- ✅ DDL操作支持
- ✅ 性能指标收集
- ✅ 集成测试通过
- ✅ 文档示例完整

### P2完成标准
- ✅ 6个数据库都有压力引擎
- ✅ 完整的监控报告
- ✅ 性能达到基准
- ✅ 生产就绪

---

## 🚀 立即行动

### 今天应该做
1. 阅读本文档
2. 查看 `pkg/pressure/mysql-row/pressure.go` 理解代码风格
3. 查看 `test/pressure/mysql_row_test.go` 理解测试方式

### 本周应该做
1. 实现 Server.Run()
2. 实现 Metadata.Initialize()
3. 实现消息队列
4. 添加配置示例

### 完成后验证
```bash
# 最终验证命令
go build -o ./bin/kyogre ./cmd/...
./bin/kyogre -config examples/config.yaml
# 应该看到:
# - 创建数据库的日志
# - 创建表的日志
# - 生成消息的日志
# - 压力测试执行的日志
```

---

## 📊 项目模块完成度总结

```
数据源层                 ████████████ 100%
消息系统                 ███████████░  90%
配置系统                 ███████████░  90%
MySQL DML压力引擎        ███████████░  90%
MySQL DDL压力引擎        ███████████░  90%
IQuery反查系统           ██████████░░  85%
元数据系统               ██████████░░  85%
Generator生成器          ███████░░░░░  75%
应用层(Server)           ███░░░░░░░░░  30% ⚠️
Scenario场景             ██░░░░░░░░░░  20%
Metrics指标              █░░░░░░░░░░░  10%
其他数据库引擎           ░░░░░░░░░░░░   0%
监控和报告               ░░░░░░░░░░░░   0%
────────────────────────────────────────
平均完成度               ███████░░░░░   62%
```

---

**最后更新**: 2025-12-20
**项目状态**: 需立即修复P0缺陷
**预期完成**: P0 3天，P1 7天，P2 15天
**总投入时间**: 约25-30天达到生产就绪
