# RangePool 调用时序与流程说明

本文档描述 RangePool 在典型使用场景下的调用时序和补货流程。

## 组件概览

- RangePool：对外入口，分成 live/free 两个分区（live=已存在范围，free=可插入范围）。
- tierPartition：分区内部管理窗口与多层 tier 库存。
- windowManager：维护分配窗口(start/end/cursor/loop)与顺序切分。
- tierManager：维护各 tier 的 segment 列表，支持随机取段与拆分。
- refillScheduler：串行执行 refill 回调，避免在业务路径中做 IO。

## 典型调用时序（ReserveInsert/Update/Delete）

```text
Caller
  |
  | ReserveInsert/Update/Delete(need)
  v
RangePool
  |
  | pickTierSize(need)
  v
tierPartition
  |
  | Update: peek(size)             Delete/Insert: consume(size)
  v
tierManager
  |
  | 随机选择 segment
  | Update: 不消费            Delete/Insert: 从库存中移除
  v
RangePool 返回 IntRange/false
```

## consume 后的补货流程（低库存触发）

```text
Caller
  |
  | ReserveInsert/Delete(need)
  v
tierPartition.consume
  |
  | 库存 <= Threshold ?
  |   |-- 否 --> 返回
  |   |
  |   '-- 是 --> 发送 refillSignal(非阻塞)
  v
后台 refillerLoop
  |
  | checkAndRefill(tier)
  |   |-- 顶层 tier: window.allocateSequential -> refillScheduler
  |   '-- 中/底层 tier: 从上层拆分 segment
  v
补到 MaxCount (或 Threshold+1)
```

## 窗口扩展（refill 回调）

```text
tierPartition/windowManager
  |
  | ensureCapacity / allocateSequential 发现窗口不足
  v
refillScheduler.schedule(wait=true)
  |
  | 调用 RangePoolRefillFunc(partition, need)
  v
refill 回调返回 (enableLoop, window)
  |
  | windowManager.applyRefillWindow 更新窗口
  v
继续分配或返回错误
```

## 关键点说明

- live/free 分区完全独立，补货仅依赖同一个 refill 回调，通过 partition 名区分。
- free 分区表示可插入窗口，既可覆盖“从未写入的范围”，也可在 enableLoop 时覆盖“可复用范围”。
- 顶层 tier 从窗口顺序切分；中/底层 tier 通过拆分上层 segment 补货。
- enableLoop 为 true 时允许 cursor 回绕到 start，且本次 refill 可切换到新的分配域(start/end 可变)。
