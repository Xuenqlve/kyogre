# Kyogre TODO

## P0

- [x] 实现通用 selector 工具，支持顺序、随机、权重三种选择策略
  说明：已新增 `pkg/tool/selector`，作为后续 `base scenario` 的选择能力基础组件。

- [ ] 设计并实现 `base scenario` 主流程
  说明：一期目标调整为通用 `base scenario`，负责策略编排、selector 驱动和 `Sequencer` 接入，不再把主能力绑定到 `mysql scenario`。

- [x] `pkg/scenario/base/plan.go`
  说明：已定义 `Plan`、`Target`、mode 常量、校验与浅拷贝能力，作为后续 builder 和 base scenario 的通用契约。

- [x] `pkg/scenario/base/builder.go`
  说明：已定义 `ContextBuilder` 接口与 registry，支持按 builder 名称注册和获取数据库适配器实例。

- [ ] `pkg/scenario/base/config.go`
  说明：定义 `base scenario` 配置结构，覆盖 `builder`、`mode`、`message-count`、`interval-ms` 以及 target / operation / row-count / transaction-size selector 配置。

- [ ] `pkg/scenario/base/selector_factory.go`
  说明：把配置转换成 `pkg/tool/selector` 选择器实例，统一创建 target、operation、row-count、transaction-size 四类 selector。

- [ ] `pkg/scenario/base/lookup.go`
  说明：封装 `Sequencer` 接入逻辑，根据 `insert/update/delete` 分别调用 `ReserveInsert`、`ReserveUpdate`、`ReserveDelete`。

- [ ] `pkg/scenario/base/scenario.go`
  说明：实现 `Scenario` 插件注册、`Configure`、`Start`、`Summary`，串起 selector、lookup 和 builder，最终产出 `generator.GenerationContext`。

- [ ] `pkg/scenario/mysql/builder.go`
  说明：实现 MySQL builder，把 `Plan` 转为 `MySQLRowContext` / `MySQLTransactionContext`，作为 `base scenario` 的第一个数据库适配器。

- [ ] `pkg/scenario/mysql/registry.go`
  说明：注册 MySQL builder 到 `base scenario` 的 builder registry，避免把 MySQL 逻辑写死在 `base scenario` 中。

- [ ] `test/scenario/base_scenario_test.go`
  说明：补 base scenario 主流程测试，覆盖 selector 驱动、context 产出、空目标和非法配置等关键路径。

- [ ] `test/scenario/mysql_builder_test.go`
  说明：补 MySQL builder 测试，验证 `Plan -> MySQLRowContext/MySQLTransactionContext` 的映射行为。

- [ ] 收敛现有 `pkg/scenario/mysql/scenario.go`
  说明：在 `base scenario + mysql builder` 跑通后，决定将旧 mysql scenario 保留兼容、转发到 base scenario，或删除。

- [ ] 补一份可直接运行的 MySQL 主配置示例
  说明：需要切到 `base scenario + builder=mysql` 的配置方式，包含 `data-source`、`metadata.type=mysql`、`generator.type=mysql`、`pressure.type=mysql-dml`、`scenario.type=base`。

- [ ] 跑通 metadata 在真实 MySQL 下的建库建表验证
  说明：重点确认 `pkg/metadata/mysql/metadata.go` 的真实 DB 初始化、建库、建表和 schema 加载行为。

- [ ] 增加一条 MySQL 数据压测端到端集成测试
  说明：覆盖配置加载 -> metadata -> scenario -> generator -> pressure -> MySQL 写入全链路。

- [ ] 明确一期只做 MySQL DML 压测，暂不把 DDL 作为必交付项
  说明：避免目标漂移，先把“数据压测”闭环做稳。

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
