# Kyogre - 高性能多数据库压力测试工具

**项目状态**: Alpha版本（62%完成）| **架构评分**: ⭐⭐⭐⭐⭐ | **优先级**: P0缺陷需立即修复

---

## 📋 文档导航

### 快速了解（推荐先看）
- **[CURRENT_STATUS.md](./CURRENT_STATUS.md)** ⭐ - **1分钟速览**
  - 项目当前状态
  - 三大缺陷清单
  - 快速行动计划

### 详细开发指南（开发必读）
- **[DEVELOPMENT_ROADMAP.md](./DEVELOPMENT_ROADMAP.md)** ⭐⭐⭐ - **30分钟详读**
  - 完整的分阶段开发计划
  - P0/P1/P2任务详情
  - 代码实现指南和模板
  - 测试验证清单

### 项目架构文档（参考）
- **[CLAUDE.md](./CLAUDE.md)** - 项目原始架构文档
  - 项目概述和设计模式
  - 模块架构
  - 核心组件说明

---

## 🎯 项目概况

**Kyogre** 是一个高性能、多数据库压力测试工具，支持以下数据库：
- ✅ MySQL（完整支持DML/DDL）
- ✅ Redis、MongoDB、ClickHouse、Kafka（框架已建，引擎待实现）

**核心特性**:
- 🏗️ 工厂模式的插件化架构，易于扩展
- 🔄 生成→反查→执行的完整流程设计
- 📊 配置驱动，支持YAML/TOML
- 🧪 26个单元测试，框架完整

---

## 🔴 关键信息：三个P0致命缺陷

项目因以下三个缺陷**无法运行**，需要立即修复（共需**2-3天**）：

### 1. Server.Run() 为空
- **文件**: `internal/app/app.go:51`
- **影响**: 程序启动后立即返回，无任何执行
- **修复时间**: 2-4小时
- **需要**: 实现完整的执行流程和组件协调

### 2. Metadata.Initialize() 为空
- **文件**: `internal/plugin/metadata/`
- **影响**: 无法创建测试表结构
- **修复时间**: 1-2小时
- **需要**: 实现数据库、表、索引的创建逻辑

### 3. 消息管道缺失
- **文件**: 全局架构
- **影响**: Generator、IQuery、Pressure无法协同
- **修复时间**: 3-5小时
- **需要**: 实现MessageQueue和消息处理流程

---

## ✅ 项目优秀之处

✅ **架构设计** (A级) - 工厂模式应用一致，扩展性优秀
✅ **数据源层** (100%) - 6个数据库都已注册
✅ **MySQL压力引擎** (90%) - DML/DDL都支持，完整实现
✅ **IQuery反查系统** (85%) - 内存和MySQL模式都实现
✅ **配置系统** (90%) - YAML/TOML双层架构
✅ **测试覆盖** (60%) - 26个测试函数

---

## 📊 完成度概览

| 模块 | 完成度 | 状态 |
|------|--------|------|
| 架构设计 | 100% | ✅ |
| 数据源层 | 100% | ✅ |
| MySQL DML/DDL引擎 | 90% | ✅ |
| IQuery反查 | 85% | ✅ |
| 消息系统 | 90% | ⚠️ 缺队列 |
| 配置系统 | 90% | ✅ |
| Generator生成器 | 75% | ⚠️ |
| 应用层(Server) | **30%** | 🔴 缺Run() |
| Metrics指标 | 10% | ❌ |
| 其他数据库引擎 | 0% | ❌ |
| **总体** | **62%** | ⚠️ Alpha |

---

## 🚀 立即行动

### 今天（1小时）
1. 📖 阅读 [CURRENT_STATUS.md](./CURRENT_STATUS.md)
2. 📋 浏览 [DEVELOPMENT_ROADMAP.md](./DEVELOPMENT_ROADMAP.md)

### 本周（2-3天）
1. 🔴 实现 `Server.Run()` 方法（2-4小时）
2. 🔴 实现 `Metadata.Initialize()` 方法（1-2小时）
3. 🔴 实现消息队列 `MessageQueue`（3-5小时）
4. 🟡 添加配置示例和文档（1-2小时）

### 验证（1小时）
```bash
# 编译
go build -o ./bin/kyogre ./cmd/...

# 运行
./bin/kyogre -config examples/config.yaml

# 验证
mysql -h localhost -u root kyogre_test -e "SHOW TABLES;"
go test -v ./test/...
```

---

## 📁 核心文件速查

### 需要修改
| 文件 | 内容 | 优先级 |
|------|------|--------|
| `internal/app/app.go` | 实现Run()方法 | 🔴 P0 |

### 需要新建
| 文件 | 内容 | 优先级 |
|------|------|--------|
| `internal/message/queue.go` | MessageQueue实现 | 🔴 P0 |
| `internal/plugin/metadata/mysql_metadata.go` | Metadata实现 | 🔴 P0 |
| `examples/config.yaml` | 配置示例 | 🔴 P0 |
| `examples/schema.yaml` | Schema示例 | 🔴 P0 |

### 参考实现（学习）
- `pkg/pressure/mysql-row/pressure.go` - Pressure接口实现
- `pkg/data_source/mysql/data_source.go` - DataSource接口实现
- `test/pressure/mysql_row_test.go` - 测试写法

---

## 🎓 开发规范

### 代码风格
```go
// 参考现有实现的风格
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
- 参考 `test/pressure/mysql_row_test.go` 的写法
- 使用 `TestMain` 进行环境设置

---

## 📚 相关资源

- **项目架构**: 详见 [CLAUDE.md](./CLAUDE.md)
- **开发指南**: 详见 [DEVELOPMENT_ROADMAP.md](./DEVELOPMENT_ROADMAP.md)
- **当前状态**: 详见 [CURRENT_STATUS.md](./CURRENT_STATUS.md)

---

## 💡 常见问题

**Q: 项目能用吗?**
A: 不能。需要修复3个P0缺陷后才能使用。预计2-3天。

**Q: 需要重写什么?**
A: 不需要。架构已经很好，只需完成缺失的执行流程。

**Q: 能跳过某个阶段吗?**
A: 不能。P0缺陷阻塞整个项目，必须先修复。

**Q: 修复后能用于生产吗?**
A: P0完成后可用于测试。P1完成后可用于小规模生产。

---

## ✨ 总结

**Kyogre** 是一个架构A级、设计优秀的项目。只需修复3个P0缺陷（2-3天），就能从"无法运行"升级到"完全可用"。

**立即开始**: 阅读 [CURRENT_STATUS.md](./CURRENT_STATUS.md) 了解项目现状，然后按 [DEVELOPMENT_ROADMAP.md](./DEVELOPMENT_ROADMAP.md) 的计划逐步推进。

---

**最后更新**: 2025-12-20
**项目状态**: Alpha版本，62%完成
**优先级**: 🔴 P0缺陷需立即修复
