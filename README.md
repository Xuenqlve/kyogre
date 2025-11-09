# Kyogre - 数据库压测工具

<div align="center">

![Kyogre](https://img.pokemondb.net/sprites/home/normal/kyogre.png)

**像盖欧卡引发暴雨洪水一样，模拟海量数据流量的数据库压测工具**

[![Go Version](https://img.shields.io/badge/Go-1.19+-blue.svg)](https://golang.org)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)
[![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg)](CONTRIBUTING.md)

</div>

## 🚀 项目简介

Kyogre 是一个高性能、多数据库支持的压测工具，以宝可梦中的盖欧卡(Kyogre)命名，象征着它能够像引发暴雨洪水一样，模拟海量数据流量对多种数据库进行全方位的压力测试。

## ✨ 核心特性

### 🗄️ 多数据库支持
- **MySQL**: 完整的 SQL 操作支持
- **MongoDB**: 文档数据库压测
- **Redis**: 内存数据库压测
- **扩展性**: 易于添加新的数据库类型

### 📊 压测模式
- **一次性数据**: 预定义数据量的压测
- **持续性数据**: 持续生成流量的压测
- **混合模式**: 多种操作类型的组合压测

### 🔧 MySQL 深度支持
- **DML操作**: 增、删、改、查
- **DDL操作**: 表结构变更
- **大事务**: 复杂事务场景模拟
- **Ghost操作**: 在线DDL变更压测
- **连接池测试**: 数据库连接压力测试

### 📈 监控指标
- 吞吐量(QPS/TPS)
- 响应时间分布
- 错误率和异常统计
- 资源使用情况
- 实时性能图表

## 🛠️ 快速开始

### 安装

```bash
# 从源码安装
git clone https://github.com/your-org/kyogre.git
cd kyogre
go build -o kyogre cmd/kyogre/main.go

# 或使用 go install
go install github.com/your-org/kyogre@latest
```

### 基本使用

```bash
# 查看帮助
kyogre --help

# 运行MySQL压测
kyogre --config config/mysql-row.yaml --scenario scenario/basic.yaml

# 运行MongoDB压测
kyogre --config config/mongodb.yaml --scenario scenario/read-heavy.yaml

# 运行Redis压测
kyogre --config config/redis.yaml --scenario scenario/cache.yaml
```

## 📋 配置示例

### MySQL 配置示例

```yaml
# config/mysql-row.yaml
database:
  type: "mysql-row"
  mysql:
    host: "localhost"
    port: 3306
    username: "test_user"
    password: "test_password"
    database: "pressure_test"
    max_connections: 100
    max_idle_connections: 20

pressure:
  mode: "continuous"  # continuous | once
  duration: "10m"     # 持续时间
  threads: 50         # 并发线程数
  rate_limit: 1000    # 每秒操作数限制

scenario:
  name: "mixed-pressure"
  operations:
    - type: "insert"
      ratio: 40
      batch_size: 10
    - type: "update" 
      ratio: 30
      batch_size: 5
    - type: "delete"
      ratio: 20
      batch_size: 3
    - type: "select"
      ratio: 10
    - type: "ddl"
      ratio: 5
      operations: ["add_column", "create_index"]
    - type: "big_transaction"
      ratio: 2
      min_operations: 10
      max_operations: 50
```

### MongoDB 配置示例

```yaml
# config/mongodb.yaml
database:
  type: "mongodb"
  mongodb:
    uri: "mongodb://localhost:27017"
    database: "pressure_test"
    collection: "test_data"

pressure:
  mode: "continuous"
  duration: "5m"
  threads: 30

scenario:
  operations:
    - type: "insert"
      ratio: 50
    - type: "find"
      ratio: 30
    - type: "update"
      ratio: 15
    - type: "delete"
      ratio: 5
```

### Redis 配置示例

```yaml
# config/redis.yaml
database:
  type: "redis"
  redis:
    addr: "localhost:6379"
    password: ""
    db: 0

pressure:
  mode: "continuous"
  duration: "3m"
  threads: 100

scenario:
  operations:
    - type: "set"
      ratio: 40
    - type: "get"
      ratio: 40
    - type: "hset"
      ratio: 10
    - type: "lpush"
      ratio: 10
```

## 📊 场景文件示例

### 混合工作负载场景

```yaml
# scenario/mixed-pressure.yaml
name: "mixed-database-pressure"
description: "混合读写和DDL操作的压力测试"

data_schema:
  table_name: "user_behavior"
  columns:
    - name: "user_id"
      type: "int"
      generator: "sequential"
    - name: "username"
      type: "string"
      generator: "random"
      length: 8
    - name: "email"
      type: "string" 
      generator: "email"
    - name: "age"
      type: "int"
      generator: "range"
      min: 18
      max: 80
    - name: "created_at"
      type: "timestamp"
      generator: "current_time"

workload:
  phases:
    - name: "warm-up"
      duration: "1m"
      threads: 10
      operations:
        insert: 70
        select: 30
    
    - name: "peak-load"
      duration: "5m" 
      threads: 100
      operations:
        insert: 40
        update: 30
        select: 20
        delete: 10
    
    - name: "ddl-phase"
      duration: "2m"
      threads: 50
      operations:
        select: 60
        ddl: 40
```

### Ghost DDL 压测场景

```yaml
# scenario/mysql-ddl.yaml
name: "online-ddl-pressure"
description: "模拟在线DDL变更期间的数据库压力"

workload:
  phases:
    - name: "baseline"
      duration: "2m"
      threads: 50
      operations:
        insert: 40
        update: 30
        select: 30
    
    - name: "mysql-migration"
      duration: "10m"
      threads: 50
      ghost_operations:
        - table: "user_behavior"
          operation: "add_column"
          column: "last_login"
          type: "timestamp"
        - table: "user_behavior" 
          operation: "create_index"
          index: "idx_email"
          columns: ["email"]
      background_operations:
        insert: 35
        update: 25
        select: 40
```

## 📈 监控和报告

Kyogre 提供丰富的监控指标和报告：

```bash
# 实时监控
kyogre --config config/mysql-row.yaml --scenario scenario/mixed.yaml --monitor

# 生成HTML报告
kyogre --config config/mysql-row.yaml --scenario scenario/mixed.yaml --report html

# 导出JSON数据
kyogre --config config/mysql-row.yaml --scenario scenario/mixed.yaml --report json
```

报告内容包括：
- 📊 吞吐量趋势图
- ⏱️ 响应时间分布
- 🔴 错误统计和分析
- 📉 资源使用情况
- 🎯 性能瓶颈识别

## 🏗️ 架构设计

```
kyogre/
├── cmd/
│   └── main.go              # 主入口
├── internal/
│   ├── config/              # 配置管理
│   ├── data_source/         # 数据库接口和实现
│   ├── generator/           # 数据生成器
│   ├── scenario/            # 场景管理
│   ├── workload/            # 工作负载引擎
│   ├── metrics/             # 指标收集
│   └── report/              # 报告生成
├── pkg/
│   ├── types/               # 公共类型定义
│   └── utils/               # 工具函数
└── examples/                # 示例配置和场景
```

## 🤝 参与贡献

我们欢迎社区贡献！请参阅 [CONTRIBUTING.md](CONTRIBUTING.md) 了解如何参与项目开发。

### 开发环境设置

```bash
# 克隆项目
git clone https://github.com/your-org/kyogre.git
cd kyogre

# 安装依赖
go mod download

# 运行测试
go test ./...

# 构建项目
go build -o kyogre cmd/kyogre/main.go
```

## 📄 许可证

本项目采用 Apache 2.0 许可证 - 详见 [LICENSE](LICENSE) 文件。

## 🙏 致谢

感谢所有为这个项目做出贡献的开发者！

---

<div align="center">

**像盖欧卡掌控海洋一样，Kyogre 助你掌控数据库性能！**

</div>