# Kyogre TODO

## P0

- [x] 实现通用 selector 工具，支持顺序、随机、权重三种选择策略
  说明：已新增 `pkg/tool/selector`，作为后续 `base scenario` 的选择能力基础组件。

- [x] 实现 `mysql` 场景插件，产出 `MySQLRowContext` / `MySQLTransactionContext`
  说明：已补齐 `pkg/scenario/mysql`，当前支持 `insert` 的 row / transaction 两种 context 产出；`update/delete` 与 lookup 联动继续放在后续任务。

- [ ] 补一份可直接运行的 MySQL 主配置示例
  说明：需要包含 `data-source`、`metadata.type=mysql`、`generator.type=mysql`、`pressure.type=mysql-dml`、`scenario.type=mysql`。

- [ ] 跑通 metadata 在真实 MySQL 下的建库建表验证
  说明：重点确认 `pkg/metadata/mysql/metadata.go` 的真实 DB 初始化、建库、建表和 schema 加载行为。

- [ ] 增加一条 MySQL 数据压测端到端集成测试
  说明：覆盖配置加载 -> metadata -> scenario -> generator -> pressure -> MySQL 写入全链路。

- [ ] 明确一期只做 MySQL DML 压测，暂不把 DDL 作为必交付项
  说明：避免目标漂移，先把“数据压测”闭环做稳。

## P1

- [ ] 为 `mysql` 场景补充插入、更新、删除三类数据生成策略
  说明：至少支持最基础的 insert / update / delete 压测模型。

- [ ] 打通 lookup / sequencer 与 MySQL 场景的联动
  说明：让 update / delete 能稳定拿到已有主键或唯一键范围。

- [ ] 补充 MySQL transaction 场景与回归测试
  说明：验证 `MySQLTransactionContext` 到 `mysql-row` pressure 的事务执行能力。

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
