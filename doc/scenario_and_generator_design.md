# Scenario 与 Generator 设计方案

## 概述

根据架构梳理，`Scenario`（场景）和 `Generator`（生成器）是压测框架的核心组件：
- **Generator**: 根据元数据生成单条消息（Message）
- **Scenario**: 管理多个 Generator，控制消息流和压测流程

本文提供 4 个具体实现方案，从简单到复杂，可根据需求选择或组合使用。

---

## 核心概念关系

```
Metadata → Generator → Scenario → Pressure
   ↓          ↓           ↓         ↓
表结构定义  生成单条消息  编排压测流程  执行压测
```

### 初始化顺序（按照架构梳理）
1. 初始化所有 DataSource
2. 初始化所有 Metadata（创建表结构）
3. 初始化所有 Generator（传入依赖的 Metadata）
4. 初始化所有 Pressure
5. 初始化所有 Scenario（注册所需的 Generator）
6. 启动 Pressure
7. 启动 Scenario（开始生成和推送消息）

---

## 方案 1：简单顺序生成器（推荐新手）

**适用场景**: 单一压测目标，简单业务逻辑

### 设计特点
- Generator 按顺序生成消息
- Scenario 简单地循环调用 Generator 生成消息
- 适合单表的 INSERT、UPDATE、DELETE 等简单操作

### 核心接口

```go
// Generator 接口保持不变
type Generator interface {
    Configure(pipelineName string, data map[string]any) error
    MockMessage() message.Message
    Close()
}

// Scenario 简化版本
type Scenario interface {
    Configure(pipeline string, data map[string]any) error
    Start(ctx context.Context) error  // 启动生成消息的循环
    Close() error
}
```

### 实现示例

```go
type SimpleScenario struct {
    generators []plugin.Generator
    messageOut message.InPoint
    interval   time.Duration  // 生成消息的间隔
    stopChan   chan struct{}
}

func (s *SimpleScenario) Configure(pipeline string, data map[string]any) error {
    // 解析配置：生成器列表、间隔时间等
    return nil
}

func (s *SimpleScenario) Start(ctx context.Context) error {
    ticker := time.NewTicker(s.interval)
    defer ticker.Stop()

    for {
        select {
        case <-ctx.Done():
            return ctx.Err()
        case <-s.stopChan:
            return nil
        case <-ticker.C:
            // 轮流调用所有 Generator 生成消息
            for _, gen := range s.generators {
                msg := gen.MockMessage()
                s.messageOut <- msg
            }
        }
    }
}

func (s *SimpleScenario) Close() error {
    close(s.stopChan)
    for _, gen := range s.generators {
        gen.Close()
    }
    return nil
}
```

### 配置示例

```yaml
scenario:
  type: simple-sequence
  generators:
    - name: mysql-insert-gen
      type: mysql-row
      config:
        operation: INSERT
        # ... 其他配置
    - name: mysql-update-gen
      type: mysql-row
      config:
        operation: UPDATE
  interval: 10ms  # 每 10ms 生成一条消息
```

### 优点
- 实现简单，易于理解
- 适合初期快速验证
- 消息生成有序可预测

### 缺点
- 无法模拟复杂业务逻辑
- 消息生成与实际业务不符
- 不支持条件判断、概率分布等

---

## 方案 2：权重随机生成器（推荐实际应用）

**适用场景**: 真实业务模拟，需要按比例的混合操作

### 设计特点
- 每个 Generator 配置权重（概率）
- Scenario 按权重随机选择 Generator
- 模拟真实业务中各种操作的比例

### 核心接口扩展

```go
// Generator 扩展：支持权重
type WeightedGenerator struct {
    Generator plugin.Generator
    Weight    int  // 权重，用于随机选择
    Name      string
}

type RandomScenario struct {
    generators []*WeightedGenerator
    messageOut message.InPoint
    totalWeight int
    rand       *rand.Rand
    rps        int64  // 消息/秒
}
```

### 实现示例

```go
func (s *RandomScenario) Configure(pipeline string, data map[string]any) error {
    // 解析配置：生成器列表及其权重、RPS 等
    // 初始化随机数生成器，计算 totalWeight
    return nil
}

func (s *RandomScenario) Start(ctx context.Context) error {
    ticker := time.NewTicker(time.Second / time.Duration(s.rps))
    defer ticker.Stop()

    for {
        select {
        case <-ctx.Done():
            return ctx.Err()
        case <-ticker.C:
            // 按权重随机选择一个 Generator
            selectedGen := s.selectByWeight()
            msg := selectedGen.MockMessage()

            select {
            case s.messageOut <- msg:
            case <-ctx.Done():
                return ctx.Err()
            }
        }
    }
}

func (s *RandomScenario) selectByWeight() plugin.Generator {
    // 基于权重的加权随机选择
    randVal := s.rand.Intn(s.totalWeight)
    cumulative := 0

    for _, wg := range s.generators {
        cumulative += wg.Weight
        if randVal < cumulative {
            return wg.Generator
        }
    }

    return s.generators[len(s.generators)-1].Generator
}

func (s *RandomScenario) Close() error {
    for _, wg := range s.generators {
        wg.Generator.Close()
    }
    return nil
}
```

### 配置示例

```yaml
scenario:
  type: random-weighted
  rps: 1000  # 1000 条消息/秒
  generators:
    - name: mysql-insert-gen
      weight: 50  # 50%
      config:
        operation: INSERT
        # ...
    - name: mysql-update-gen
      weight: 30  # 30%
      config:
        operation: UPDATE
        # ...
    - name: mysql-delete-gen
      weight: 20  # 20%
      config:
        operation: DELETE
        # ...
```

### 优点
- 接近真实业务模式
- 可调整各操作比例
- 易于配置和理解
- 性能可控（支持 RPS 限流）

### 缺点
- 无法处理有依赖关系的操作序列
- 不支持状态转移

---

## 方案 3：状态机生成器（推荐复杂业务）

**适用场景**: 有业务流程的压测，如用户生命周期、订单流程等

### 设计特点
- 定义状态和状态转移规则
- Scenario 根据当前状态和转移概率选择 Generator
- 支持复杂的业务流程模拟

### 核心接口扩展

```go
// 状态转移规则
type StateTransition struct {
    FromState string
    ToState   string
    Generator plugin.Generator
    Probability float32  // 转移概率
}

type StatefulScenario struct {
    states        map[string]bool
    transitions   map[string][]*StateTransition
    currentState  string
    messageOut    message.InPoint
    rps           int64
}
```

### 实现示例

```go
func (s *StatefulScenario) Configure(pipeline string, data map[string]any) error {
    // 解析状态定义和转移规则
    return nil
}

func (s *StatefulScenario) Start(ctx context.Context) error {
    ticker := time.NewTicker(time.Second / time.Duration(s.rps))
    defer ticker.Stop()

    for {
        select {
        case <-ctx.Done():
            return ctx.Err()
        case <-ticker.C:
            // 基于当前状态和转移规则选择下一个操作
            transition := s.selectTransition()
            if transition == nil {
                continue
            }

            msg := transition.Generator.MockMessage()
            select {
            case s.messageOut <- msg:
            case <-ctx.Done():
                return ctx.Err()
            }

            // 更新状态
            s.currentState = transition.ToState
        }
    }
}

func (s *StatefulScenario) selectTransition() *StateTransition {
    transitions, ok := s.transitions[s.currentState]
    if !ok || len(transitions) == 0 {
        return nil
    }

    // 基于概率随机选择转移
    randVal := rand.Float32()
    cumulative := float32(0)

    for _, t := range transitions {
        cumulative += t.Probability
        if randVal < cumulative {
            return t
        }
    }

    return transitions[len(transitions)-1]
}
```

### 配置示例

```yaml
scenario:
  type: stateful
  rps: 1000
  initialState: "new_user"
  states:
    - new_user
    - active_user
    - archived_user
  transitions:
    - from: "new_user"
      to: "active_user"
      probability: 0.9
      generator:
        type: mysql-row
        operation: UPDATE
        # 更新用户状态为活跃
    - from: "new_user"
      to: "new_user"
      probability: 0.1
      generator:
        type: mysql-row
        operation: INSERT
        # 继续插入新用户
    - from: "active_user"
      to: "archived_user"
      probability: 0.5
      generator:
        type: mysql-row
        operation: DELETE
        # 归档用户
```

### 优点
- 支持复杂业务流程
- 能模拟真实的用户行为链
- 灵活的状态管理

### 缺点
- 配置相对复杂
- 状态增多时难以维护
- 调试相对困难

---

## 方案 4：Pipeline 链式生成器（推荐高级用户）

**适用场景**: 多个有序或并行的操作流，如 INSERT → UPDATE → DELETE

### 设计特点
- 定义 Generator Pipeline：多个 Generator 按特定顺序或逻辑执行
- 支持串行、并行、条件分支
- 每个 Pipeline 作为一个完整的业务事务

### 核心接口扩展

```go
// Pipeline 节点
type PipelineNode interface {
    Execute(ctx context.Context) (message.Message, error)
}

type GeneratorNode struct {
    generator plugin.Generator
    repeat    int  // 重复次数
}

type ConditionalNode struct {
    condition func() bool
    trueBranch PipelineNode
    falseBranch PipelineNode
}

type ParallelNode struct {
    nodes []PipelineNode
    count int  // 并行数
}

type PipelineScenario struct {
    pipelines  []PipelineNode
    messageOut message.InPoint
    rps        int64
}
```

### 实现示例

```go
// 串行 Pipeline 示例
func BuildInsertUpdateDeletePipeline(
    insertGen, updateGen, deleteGen plugin.Generator) PipelineNode {

    return &SequentialNode{
        nodes: []PipelineNode{
            &GeneratorNode{generator: insertGen},
            &GeneratorNode{generator: updateGen},
            &GeneratorNode{generator: deleteGen},
        },
    }
}

type SequentialNode struct {
    nodes []PipelineNode
}

func (n *SequentialNode) Execute(ctx context.Context) (message.Message, error) {
    var lastMsg message.Message

    for _, node := range n.nodes {
        msg, err := node.Execute(ctx)
        if err != nil {
            return nil, err
        }
        lastMsg = msg
    }

    return lastMsg, nil
}

func (s *PipelineScenario) Start(ctx context.Context) error {
    ticker := time.NewTicker(time.Second / time.Duration(s.rps))
    defer ticker.Stop()

    for {
        select {
        case <-ctx.Done():
            return ctx.Err()
        case <-ticker.C:
            // 循环执行每个 Pipeline
            for _, pipeline := range s.pipelines {
                msg, err := pipeline.Execute(ctx)
                if err != nil {
                    log.Error("pipeline execute failed", err)
                    continue
                }

                select {
                case s.messageOut <- msg:
                case <-ctx.Done():
                    return ctx.Err()
                }
            }
        }
    }
}
```

### 配置示例

```yaml
scenario:
  type: pipeline
  rps: 100  # 每秒 100 个完整 Pipeline
  pipelines:
    - name: user_lifecycle
      type: sequential
      steps:
        - generator: mysql-insert-user
          repeat: 1
        - generator: mysql-update-user
          repeat: 5
        - generator: mysql-delete-user
          repeat: 1
    - name: order_processing
      type: sequential
      steps:
        - generator: mysql-insert-order
          repeat: 1
        - generator: mysql-update-order-status
          repeat: 3
        - generator: mysql-delete-order
          repeat: 1
```

### 优点
- 支持复杂的操作序列
- 可模拟完整的业务事务
- 易于扩展新的 Node 类型
- 清晰的流程定义

### 缺点
- 实现相对复杂
- 配置较为冗长
- 性能开销较大

---

## 比较总结

| 方案 | 复杂度 | 业务模拟程度 | 推荐场景 | 配置难度 |
|------|--------|------------|--------|--------|
| 1. 简单顺序 | ⭐ | ⭐ | 简单压测、学习 | ⭐ |
| 2. 权重随机 | ⭐⭐ | ⭐⭐⭐ | 混合操作、真实业务 | ⭐⭐ |
| 3. 状态机 | ⭐⭐⭐ | ⭐⭐⭐⭐ | 有流程的复杂业务 | ⭐⭐⭐ |
| 4. Pipeline 链式 | ⭐⭐⭐⭐ | ⭐⭐⭐⭐⭐ | 多步骤事务、复杂流程 | ⭐⭐⭐ |

---

## 建议实现路径

### 第一阶段（MVP）
实现**方案 2：权重随机生成器**，能够满足大多数场景：
- 配置相对简单
- 功能足够实用
- 作为其他方案的基础

### 第二阶段（增强）
添加**方案 3：状态机生成器**，支持有流程的业务：
- 复用权重随机的基础设施
- 新增状态转移逻辑
- 兼容两种使用模式

### 第三阶段（高级）
可选实现**方案 4：Pipeline 链式**或**方案 1：简单顺序**：
- 根据实际需求选择
- Pipeline 用于复杂事务
- 简单顺序用于调试和学习

---

## 技术细节补充

### Generator 实现建议

```go
// MySQL DML Generator 示例
type MySQLRowGenerator struct {
    metadata metadata.Metadata
    operation string  // INSERT, UPDATE, DELETE, SELECT
    // ...
}

func (g *MySQLRowGenerator) Configure(pipelineName string, data map[string]any) error {
    // 配置 Metadata、操作类型、WHERE 条件等
    return nil
}

func (g *MySQLRowGenerator) MockMessage() message.Message {
    // 根据 Metadata 生成单条消息
    return &message.MySQLRowMessage{
        Operation: g.operation,
        // ... 其他字段
    }
}
```

### Scenario 与 Pressure 的交互

```go
type Scenario interface {
    // 将消息点注入，用于推送生成的消息
    SetMessageOut(out message.InPoint)

    Configure(pipeline string, data map[string]any) error
    Start(ctx context.Context) error
    Close() error
}

// 在 Server.Run() 中：
scenario.SetMessageOut(messageChannel)
pressure.SetMessageIn(messageChannel)
```

### 性能优化建议

1. **消息缓冲**：使用带缓冲的 channel
   ```go
   messageChannel := make(message.Point, 1000)
   ```

2. **生成器池**：对于高并发场景，可为每个 Generator 创建多个实例

3. **预生成缓存**：提前生成消息队列以减少生成延迟

4. **监控指标**：
   - 消息生成速率（生成的消息/秒）
   - 队列深度（缓冲消息数）
   - 生成延迟（消息生成耗时）

---

## 文件结构建议

```
internal/plugin/
├── scenario.go              # 场景接口定义
├── generator.go             # 生成器接口定义
└── metadata.go              # 已存在

pkg/scenario/
├── simple/                  # 方案 1
│   └── scenario.go
├── random/                  # 方案 2（推荐首选）
│   ├── scenario.go
│   └── weighted.go
├── stateful/                # 方案 3
│   ├── scenario.go
│   └── state_machine.go
└── pipeline/                # 方案 4
    ├── scenario.go
    ├── node.go
    └── executor.go

pkg/generator/
├── registry.go              # Generator 注册
└── mysql/
    ├── row_generator.go
    └── ddl_generator.go
```

---

## 总结

根据当前项目状态和实际需求，**推荐优先实现方案 2（权重随机生成器）**，因为：

1. ✅ 足以支持大多数压测场景
2. ✅ 实现复杂度适中
3. ✅ 配置直观易懂
4. ✅ 作为其他方案的基础
5. ✅ 易于后期扩展

后续可根据实际需求逐步增加其他方案支持。