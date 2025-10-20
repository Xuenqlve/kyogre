# gh-ost 压力测试引擎集成指南

本文档说明如何在 Kyogre 框架中集成和使用 gh-ost 压力测试引擎。

## 架构概述

```
┌─────────────────────────────────────────────────────────┐
│                    Kyogre Framework                      │
├─────────────────────────────────────────────────────────┤
│                                                           │
│  ┌────────────────────────────────────────────────────┐ │
│  │           Message Pipeline / Orchestration         │ │
│  ├────────────────────────────────────────────────────┤ │
│  │                                                    │ │
│  │  ┌──────────────┐      ┌─────────────────────┐   │ │
│  │  │ DDL Source   │ ───▶ │  gh-ost Pressure    │   │ │
│  │  │ (Generator)  │      │  Test Engine        │   │ │
│  │  └──────────────┘      ├─────────────────────┤   │ │
│  │                        │ ┌─────────────────┐ │   │ │
│  │                        │ │ DDL Queue       │ │   │ │
│  │                        │ ├─────────────────┤ │   │ │
│  │                        │ │ Migration Tasks │ │   │ │
│  │                        │ ├─────────────────┤ │   │ │
│  │                        │ │ gh-ost Executor │ │   │ │
│  │                        │ └─────────────────┘ │   │ │
│  │                        └─────────────────────┘   │ │
│  │                               ▼                   │ │
│  │                        MySQL Database             │ │
│  │                                                    │ │
│  └────────────────────────────────────────────────────┘ │
│                                                           │
└─────────────────────────────────────────────────────────┘
```

## 数据流程

1. **DDL 源** - 数据生成器或场景定义提供 DDL 操作
2. **消息转换** - DDL 转换为 `DDLMessage` 对象
3. **队列缓冲** - 消息放入 DDL 队列
4. **处理执行** - 工作 goroutine 从队列取出消息
5. **gh-ost 调用** - 启动 gh-ost 子进程执行迁移
6. **监控与日志** - 记录迁移进度和结果

## 使用流程

### 1. 配置 YAML 文件

```yaml
# config.yaml
data-source:
  mysql-primary:
    host: localhost
    port: 3306
    user: root
    password: ""
    database: testdb
    max-idle-conns: 10
    max-open-conns: 20

pressure:
  gh-ost:
    source-data-source: "mysql-primary"
    execute-changes: true
    chunk-size: 1000
    max-load: 100
    timeout: 3600
```

### 2. 在 Kyogre 中初始化

```go
package main

import (
    "context"
    "github.com/xuenqlve/kyogre/pkg/pressure/gh-ost"
    // ... 其他导入
)

func main() {
    // 1. 加载配置
    config := loadConfig("config.yaml")

    // 2. 初始化压力测试引擎
    pressure := &gh_ost.Pressure{}
    if err := pressure.Configure("my-pipeline", config["pressure"]["gh-ost"]); err != nil {
        panic(err)
    }

    // 3. 启动引擎
    ctx := context.Background()
    if err := pressure.Start(ctx); err != nil {
        panic(err)
    }
    defer pressure.Close()

    // 4. 发送 DDL 消息
    ddlMsg := &gh_ost.DDLMessage{
        Database:       "testdb",
        Table:          "users",
        SQL:            "ADD COLUMN age INT DEFAULT 0",
        Operation:      "ALTER TABLE",
        StartTimeValue: time.Now(),
    }
    pressure.Execute(ddlMsg)

    // 5. 等待完成
    select {}
}
```

### 3. 集成到消息管道

```go
// 示例：将 gh-ost 与 MySQL 压力测试引擎集成

func setupPipeline(mysqlPressure, ghostPressure interface{}) {
    // MySQL 压力测试执行 DML 操作
    // gh-ost 压力测试执行 DDL 操作

    // 可以将两者组合：
    // 1. MySQL 压力测试持续执行 INSERT/UPDATE/DELETE
    // 2. gh-ost 定期执行 ALTER TABLE
    // 3. 监控两者的并发影响
}
```

## 高级用法

### 1. 自定义 DDL 源

```go
// 从数据库中读取 DDL 定义并转换为消息

func readDDLsFromDB(db *sql.DB, pressure *gh_ost.Pressure) {
    rows, _ := db.Query("SELECT database, table, sql FROM ddl_operations")
    defer rows.Close()

    for rows.Next() {
        var db, tbl, sql string
        rows.Scan(&db, &tbl, &sql)

        msg := &gh_ost.DDLMessage{
            Database:       db,
            Table:          tbl,
            SQL:            sql,
            Operation:      "ALTER TABLE",
            StartTimeValue: time.Now(),
        }
        pressure.Execute(msg)
    }
}
```

### 2. 监控迁移进度

```go
// 自定义监控器

type MigrationMonitor struct {
    startTime time.Time
    pressure  *gh_ost.Pressure
}

func (m *MigrationMonitor) track() {
    for {
        select {
        case <-time.Tick(5 * time.Second):
            // 检查活跃的迁移任务
            // 记录指标
        }
    }
}
```

### 3. 错误处理和重试

```go
func executeWithRetry(pressure *gh_ost.Pressure, msg *gh_ost.DDLMessage, maxRetries int) error {
    for i := 0; i < maxRetries; i++ {
        pressure.Execute(msg)

        // 等待完成或超时
        time.Sleep(100 * time.Millisecond)

        // 检查结果...
        if success {
            return nil
        }
    }
    return fmt.Errorf("failed after %d retries", maxRetries)
}
```

## 与其他压力测试引擎的协同

### MySQL 压力测试 + gh-ost 并发

```yaml
# 配置文件：同时运行两个压力测试

pipelines:
  - name: "concurrent-test"
    stages:
      # 阶段 1：MySQL 压力测试
      - type: "mysql"
        config:
          worker-count: 10
          # ... MySQL 配置

      # 阶段 2：gh-ost DDL 迁移（并发进行）
      - type: "gh-ost"
        config:
          source-data-source: "mysql-primary"
          # ... gh-ost 配置
```

## 性能调优建议

| 参数 | 小表 | 中表 | 大表 | 说明 |
|------|------|------|------|------|
| chunk-size | 5000 | 2000 | 500 | 越大越快，但占用内存越多 |
| max-load | 150 | 100 | 50 | 超载时降低，避免影响业务 |
| max-concurrent-migrations | 5 | 2 | 1 | 并发数越高，资源占用越大 |
| timeout | 600 | 1800 | 3600+ | 大表需要更长的超时 |

## 故障排查

### 问题 1：gh-ost 进程无法启动

```
Error: gh-ost binary not found
```

**解决方案**：
```bash
# 检查安装
which gh-ost

# 添加到 PATH
export PATH=$PATH:/usr/local/bin

# 或在配置中指定完整路径
ghost-binary: "/usr/local/bin/gh-ost"
```

### 问题 2：迁移超时

```
Error: context deadline exceeded
```

**解决方案**：
1. 增加 `timeout` 配置
2. 降低 `chunk-size`
3. 检查系统资源（CPU、内存、磁盘 I/O）

### 问题 3：数据库权限不足

```
Error: Access denied for user
```

**所需权限**：
```sql
GRANT ALTER, SELECT, INSERT, UPDATE, DELETE, CREATE
ON database_name.* TO 'user'@'host';
```

## 与 CI/CD 集成

### GitHub Actions 示例

```yaml
name: DDL Migration Test

on: [push]

jobs:
  test:
    runs-on: ubuntu-latest
    services:
      mysql:
        image: mysql:8.0
        env:
          MYSQL_ROOT_PASSWORD: root

    steps:
      - uses: actions/checkout@v2

      - name: Install gh-ost
        run: |
          wget https://github.com/github/gh-ost/releases/download/v1.1.6/gh-ost-linux-amd64
          chmod +x gh-ost-linux-amd64
          sudo mv gh-ost-linux-amd64 /usr/local/bin/gh-ost

      - name: Run DDL migration test
        run: go test -v ./pkg/pressure/gh-ost/...
```

## 监控指标

建议监控以下指标：

- **迁移成功率** - 成功迁移 / 总迁移数
- **平均迁移时间** - 单次迁移耗时
- **吞吐量** - 每分钟完成的迁移数
- **资源占用** - CPU、内存、磁盘使用率
- **错误率** - 失败迁移 / 总迁移数

## 安全建议

1. **备份** - 迁移前备份数据库
2. **测试环境** - 先在测试环境验证
3. **权限控制** - 使用最小权限原则
4. **速率限制** - 避免过度并发
5. **监控告警** - 设置异常告警

## 参考资源

- [gh-ost 官方文档](https://github.com/github/gh-ost/wiki)
- [MySQL DDL 优化](https://dev.mysql.com/doc/)
- [Kyogre 框架文档](../../README.md)
