# Kyogre 项目开发路线图

**最后更新**: 2026-03-18
**项目状态**: Alpha 版本，基础编排已打通，仍缺运行闭环
**优先级**: 修复真实 DB 初始化风险，补齐示例配置与端到端验证

---

## 当前判断

这份路线图基于当前仓库代码，而不是 2025-12 的初始评估。

### 已确认完成

- `Server.Run()` 已实现，具备 pipeline 启动、停止、API 生命周期和自动退出逻辑
- `Metadata.Initialize()` 已有 MySQL / mock 两条实现路径
- scenario -> pressure 的消息流已通过 `message.Point` channel 接通
- `go test -run '^$' ./cmd/... ./internal/... ./pkg/... ./test/...` 可通过，说明当前代码可编译

### 仍需收尾

- 真实 MySQL 路径的 metadata 初始化需要运行级验证
- `examples/` 目录与配置示例仍不存在
- `test/integration/` 尚不存在，缺少端到端验证
- `internal/metrics/metrics.go` 仍为空壳，P1 尚未开始

---

## P0 状态核实

### 1. Server.Run() 空实现

- 当前状态：已解决
- 证据：`internal/app/app.go` 已负责启动 engine、管理 API 和优雅停止
- 结论：旧文档中的该 P0 判断已过时

### 2. Metadata.Initialize() 缺实现

- 当前状态：已实现，但真实 DB 路径需要继续验证
- 证据：`pkg/metadata/mysql/metadata.go` 已存在完整实现，当前工作树中也已补上连接保存
- 结论：从“未实现”变为“已实现但需验证”

### 3. 消息管道缺失

- 当前状态：基础消息流已解决
- 证据：`internal/message/message.go` 的 `Point` + `internal/app/pipeline.go` + `internal/plugin/pressure/controller.go`
- 结论：路线图里“必须新建 MessageQueue”这条已经不是当前代码的真实状态；项目采用了更轻量的 channel 方案

### 4. 示例配置与快速开始

- 当前状态：未完成
- 证据：仓库中不存在 `examples/`
- 结论：这是当前最直接影响“可复现运行”的缺口

---

## 分阶段任务

### 第一阶段：运行闭环补齐

#### P0-1 真实 MySQL 初始化验证

- 目标：确认 metadata 在真实 MySQL 环境下可以完成建库/建表
- 文件：`pkg/metadata/mysql/metadata.go`，`test/metadata/...`
- 验证：

```bash
go test -v ./test/metadata/...
```

#### P0-2 添加示例配置

- 目标：补齐最小可运行示例，降低上手成本
- 需要新增：
  - `examples/config.yaml`
  - `examples/schema.yaml`
  - `examples/README.md`

#### P0-3 增加运行说明

- 目标：把“如何编译 / 如何运行 / 如何验证”写成固定流程
- 推荐验证命令：

```bash
env GOCACHE=/tmp/kyogre-go-build-cache go test -run '^$' ./cmd/... ./internal/... ./pkg/... ./test/...
./bin/kyogre -config examples/config.yaml
mysql -h localhost -u root -p kyogre_test -e "SHOW TABLES;"
```

### 第二阶段：P1 功能完善

- 实现 metrics 系统：`internal/metrics/metrics.go`
- 补结构化错误与运行日志
- 新建 `test/integration/` 做端到端测试
- 梳理配置校验与失败场景

### 第三阶段：P2 扩展能力

- 其他数据库压力引擎
- 报告与监控
- 基准测试与性能优化

---

## 文件状态清单

### 已完成

- `internal/app/app.go`
- `internal/app/pipeline.go`
- `internal/app/api.go`
- `pkg/metadata/mysql/metadata.go`

### 未完成

- `examples/config.yaml`
- `examples/schema.yaml`
- `examples/README.md`
- `test/integration/`
- `internal/metrics/metrics.go`

---

## 当前结论

Kyogre 已经不是“3 个 P0 全部未修”的项目了。更准确的状态是：

- 主流程已接通
- 编译已通过
- 真实 DB 初始化需要继续验证
- 缺少示例配置和端到端可复现入口

下一步应聚焦运行闭环，而不是重复处理已经完成的空实现问题。
