# Kyogre TODO

## P0

- [x] 实现通用 selector 工具，支持顺序、随机、权重三种选择策略
  说明：已新增 `pkg/tool/selector`，作为后续 `base scenario` 的选择能力基础组件。

- [x] 设计并实现 `base scenario` 主流程
  说明：已完成通用 `base scenario`，负责策略编排、selector 驱动、`Sequencer` 接入和运行期错误回收，不再把主能力绑定到旧 `mysql scenario`。

- [x] `pkg/scenario/base/plan.go`
  说明：已定义 `Plan`、`Target`、mode 常量、校验与浅拷贝能力，作为后续 builder 和 base scenario 的通用契约。

- [x] `pkg/scenario/base/builder.go`
  说明：已定义 `ContextBuilder` 接口与 registry，支持按 builder 名称注册和获取数据库适配器实例。

- [x] `pkg/scenario/base/config.go`
  说明：已定义 `base scenario` 配置结构、默认值和校验逻辑，覆盖 builder、mode、selector 配置和 lookup 配置。

- [x] `pkg/scenario/base/selector_factory.go`
  说明：已实现 selector factory，可把 target、operation、row-count、transaction-size 配置转换成运行时选择器实例。

- [x] `pkg/scenario/base/lookup.go`
  说明：已封装 `Sequencer` 接入 binder，可按操作类型分发 reserve 调用并回填 provider。

- [x] `pkg/scenario/base/scenario.go`
  说明：已实现 `Scenario` 插件注册、`Configure`、`Start`、`Summary`，可以串起 selector、lookup 和 builder 产出 `generator.GenerationContext`。

- [x] `pkg/scenario/mysql/builder.go`
  说明：已实现 MySQL builder，支持从 metadata 加载 `Target + SequenceSpec`，并将 `Plan` 转为 `MySQLRowContext` / `MySQLTransactionContext`。

- [x] `pkg/scenario/mysql/registry.go`
  说明：已注册 MySQL builder 到 `base scenario` 的 builder registry，可通过 `builder=mysql` 获取适配器实例。

- [x] `test/scenario/base/`
  说明：已补 `test/scenario/base/` 下的主流程测试，覆盖 selector 驱动、context 产出、空目标、配置重置和运行期错误等关键路径。

- [x] `test/scenario/mysql_builder_test.go`
  说明：已补 MySQL builder 测试，覆盖 target 加载、SequenceSpec 绑定、row/transaction context 构造和注册校验。

- [x] 收敛现有 `pkg/scenario/mysql/scenario.go`
  说明：已将旧 `scenario.type=mysql` 收敛为兼容入口，内部转发到 `base scenario + builder=mysql`，避免继续维护两套 MySQL 主链路。

- [x] 补一份可直接运行的 MySQL 主配置示例
  说明：已补 `examples/mysql-config.yaml`，采用 `base scenario + builder=mysql` 组合，并在 `examples/README.md` 中补充了使用说明。

- [ ] 跑通 metadata 在真实 MySQL 下的建库建表验证
  说明：重点确认 `pkg/metadata/mysql/metadata.go` 的真实 DB 初始化、建库、建表和 schema 加载行为。

- [ ] 增加一条 MySQL 数据压测端到端集成测试
  说明：覆盖配置加载 -> metadata -> scenario -> generator -> pressure -> MySQL 写入全链路。

- [x] 明确一期只做 MySQL DML 压测，暂不把 DDL 作为必交付项
  说明：当前实现和示例都已收敛到 `base scenario + mysql builder + mysql-dml pressure` 的 DML 主链路，DDL 不再作为一期阻塞项。

- [x] 收敛 MySQL 测试配置并修复基础测试回归
  说明：已将真实 MySQL 测试数据源配置统一收敛到 `test/test_case/mysql.go`，修复 `RangeSequence` wrap 语义和 `varchar_xlarge` 类型回归，并避免 metadata 测试在初始化失败后继续触发 panic。已完成目标测试验证；真实 MySQL 依赖测试仍需在本地数据库环境下运行。

## P1

- [ ] 为 `base scenario` 补充插入、更新、删除三类操作选择策略
  说明：至少支持固定、随机、权重三种 operation selector 组合，并可稳定驱动 builder。

- [ ] 打通 lookup / sequencer 与 `base scenario` 的联动
  说明：让 update / delete 在通用场景层稳定拿到已有主键或唯一键范围，再交给具体 builder 消费。

- [ ] 补充 MySQL transaction 策略与回归测试
  说明：验证 `base scenario` 产出的 `MySQLTransactionContext` 到 `mysql-row` pressure 的事务执行能力。

- [ ] 补充失败场景验证
  说明：覆盖建表失败、数据源连接失败、pressure 执行异常、lookup 返回空窗口等情况。

- [ ] 补充 MySQL 示例文档
  说明：写清楚运行前置条件、配置说明、启动命令、验证方法。

## P2

- [ ] 决定是否把 MySQL DDL 压测纳入一期后续增强
  说明：当前已有部分实现，但不应阻塞 DML 主目标。

- [ ] 增加 metrics 能力
  说明：至少补充吞吐、错误数、延迟统计，方便判断压测结果。

- [ ] 增加压测结果输出或报告能力
  说明：便于观察执行结果，而不只是看日志。
