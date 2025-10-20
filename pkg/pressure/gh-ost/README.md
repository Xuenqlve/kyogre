# gh-ost 压力测试引擎

Kyogre 中的 gh-ost 压力测试引擎用于执行 MySQL DDL（数据定义语言）迁移操作。该引擎集成了 GitHub 开源的 [gh-ost](https://github.com/github/gh-ost) 工具，用于在线 DDL 变更。

## 功能特性

- ✅ **非阻塞式 DDL 迁移** - 使用 gh-ost 进行 ALTER TABLE 操作，不锁表
- ✅ **并发控制** - 支持配置最大并发迁移数
- ✅ **性能调优** - 支持配置 chunk size、max load 等参数
- ✅ **优雅关闭** - 支持安全的任务取消和资源清理
- ✅ **详细日志** - 完整的迁移过程记录
- ✅ **DDL 消息处理** - 支持接收和处理 DDL 消息队列

## 安装依赖

### 1. 安装 gh-ost

gh-ost 是一个独立的二进制程序，需要单独安装：

```bash
# macOS
brew install gh-ost

# Linux (Ubuntu/Debian)
wget https://github.com/github/gh-ost/releases/download/v1.1.6/gh-ost-linux-amd64 -O /usr/local/bin/gh-ost
chmod +x /usr/local/bin/gh-ost

# 验证安装
gh-ost --version
```

## 配置说明

### 基础配置

```yaml
pressure:
  gh-ost:
    # 必需：源数据库配置（来自 data-source 定义）
    source-data-source: "mysql-src"

    # 可选：gh-ost 二进制路径（默认从 PATH 中查找）
    ghost-binary: "/usr/local/bin/gh-ost"

    # 性能配置
    max-concurrent-migrations: 2      # 最大并发迁移数（默认 1）
    chunk-size: 1000                  # 批处理大小（默认 1000）
    max-load: 100                     # 最大负载（Threads_running，默认 100）

    # 迁移行为配置
    execute-changes: true             # 是否立即执行（true）或先测试（false）
    allow-on-master: false            # 是否允许在主库上运行
    cut-over: true                    # 是否自动切换表
    timeout: 3600                     # 超时时间（秒，默认 1 小时）
```

### 完整配置示例

```yaml
data-source:
  mysql-src:
    host: localhost
    port: 3306
    user: root
    password: root
    max-idle-conns: 10
    max-open-conns: 20
    conn-max-lifetime: 3600

pressure:
  gh-ost:
    source-data-source: "mysql-src"
    ghost-binary: "gh-ost"
    max-concurrent-migrations: 2
    chunk-size: 1000
    max-load: 100
    execute-changes: true
    allow-on-master: false
    cut-over: true
    timeout: 3600
```

## 使用方法

### 创建 DDL 消息

```go
import "github.com/xuenqlve/kyogre/pkg/pressure/gh-ost"

// 创建 DDL 消息
msg := &gh_ost.DDLMessage{
    Database:       "mydb",
    Table:          "users",
    SQL:            "ADD COLUMN age INT DEFAULT 0",
    Operation:      "ALTER TABLE",
    StartTimeValue: time.Now(),
}

// 发送给压力测试引擎
pressure.Execute(msg)
```

### 初始化和启动

```go
import (
    "context"
    "github.com/xuenqlve/kyogre/pkg/pressure/gh-ost"
)

// 配置
var config map[string]any = map[string]any{
    "source-data-source": "mysql-src",
    "execute-changes": true,
}

pressure := &gh_ost.Pressure{}

// 初始化
if err := pressure.Configure("my-pipeline", config); err != nil {
    panic(err)
}

// 启动
ctx := context.Background()
if err := pressure.Start(ctx); err != nil {
    panic(err)
}

// 发送 DDL 消息
msg := &gh_ost.DDLMessage{...}
pressure.Execute(msg)

// 关闭
defer pressure.Close()
```

## gh-ost 工作原理

1. **创建 Ghost 表** - 创建一个与原表结构相同的影子表
2. **应用 DDL** - 在影子表上应用 DDL 变更
3. **数据复制** - 逐步将原表数据复制到影子表
4. **日志追踪** - 在复制期间捕获对原表的 DML 变更
5. **自动切换** - 完成时原子性地切换原表和影子表
6. **清理** - 删除原表（通常改名为备份表）

## 配置最佳实践

### 性能优化

```yaml
pressure:
  gh-ost:
    # 对于大表，增加 chunk size
    chunk-size: 5000

    # 对于高负载系统，降低 max-load
    max-load: 50

    # 允许多个小表并发迁移
    max-concurrent-migrations: 3
```

### 安全配置

```yaml
pressure:
  gh-ost:
    # 在测试环境先验证
    execute-changes: false  # 先测试而不是直接执行

    # 不在主库上运行（使用副本库）
    allow-on-master: false

    # 手动控制切换时机
    cut-over: false

    # 设置合理的超时
    timeout: 7200
```

## 故障排查

### 问题：gh-ost 找不到

```
Error: gh-ost binary not found
```

**解决方案**：确保 gh-ost 已安装并在 PATH 中，或在配置中指定完整路径。

### 问题：权限不足

```
Error: Access denied for user
```

**解决方案**：确保 MySQL 用户有以下权限：
- `ALTER` - 修改表结构
- `SELECT` - 读取数据
- `INSERT, DELETE, UPDATE` - 应用 DML 变更
- `CREATE` - 创建影子表

### 问题：超时

```
Error: context deadline exceeded
```

**解决方案**：
1. 增加 `timeout` 配置
2. 降低 `chunk-size` 避免单个操作耗时过长
3. 调整 `max-load` 以适应系统负载

## 监控

在迁移过程中，可通过日志查看进度：

```
INFO: starting gh-ost migration: mydb.users, command: gh-ost --host=localhost --port=3306 ...
INFO: gh-ost migration completed for mydb.users, output: ...
```

## 参考资源

- [gh-ost GitHub 仓库](https://github.com/github/gh-ost)
- [gh-ost 文档](https://github.com/github/gh-ost/wiki)
- [MySQL DDL 最佳实践](https://dev.mysql.com/doc/)
