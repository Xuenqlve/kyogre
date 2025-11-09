# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**Kyogre** is a high-performance, multi-database stress testing tool. Named after Kyogre from Pokémon, it can simulate massive data traffic to stress test multiple types of databases (MySQL, Redis, MongoDB, ClickHouse, Kafka).

- **Language**: Go 1.24.0
- **Architecture**: Plugin-based with factory pattern for extensibility
- **Core Focus**: MySQL (DML and DDL operations), with support for other databases

## Common Development Commands

### Build

```bash
# Build the binary
go build -o kyogre ./cmd/...

# Build with specific output path
go build -o ./bin/kyogre ./cmd/...
```

### Testing

```bash
# Run all tests
go test ./...

# Run tests with verbose output
go test -v ./...

# Run specific test file
go test -v ./test/pressure/mysql_row_test.go

# Run specific test function
go test -v -run TestMySQLRowPressure ./test/pressure/...

# Run tests with coverage
go test -cover ./...

# Run tests with race detector
go test -race ./...
```

### Linting & Formatting

```bash
# Format code
go fmt ./...

# Check for issues with gofmt
gofmt -l ./...

# Run vet to find suspicious code
go vet ./...

# Clean up dependencies
go mod tidy
```

### Dependency Management

```bash
# Download dependencies
go mod download

# Tidy and verify
go mod tidy

# Check for issues
go mod verify
```

## Codebase Architecture

### High-Level Architecture

The project uses a **plugin-based architecture** with factory pattern for extensibility:

```
Configuration → Server Init → Data Source Plugin → Pressure Plugin → Execution
     ↓              ↓              ↓                    ↓               ↓
YAML/TOML      Load Config    MySQL/Redis/        DML/DDL/          Run
Config File    + Logging      MongoDB/etc        Custom Engine      Pressure Test
```

### Core Components

#### 1. **Application Layer** (`internal/app/app.go`)
- `Server`: Main application controller managing lifecycle
- Handles initialization, configuration, and execution flow
- Key methods: `NewServer()`, `Configure()`, `Run()`

#### 2. **Configuration System** (Dual-layer)

**Internal Config** (`internal/config/`):
- Factory pattern: `RegisterConfigFactory()`, `GetConfig()`
- Type: `Config` struct with nested maps for flexible configuration
- Manages datasource, scenario, pressure, and monitor configs

**Public Config Implementation** (`pkg/config/file.go`):
- `FileConfig`: Reads YAML/TOML configuration files
- Thread-safe configuration loading

#### 3. **Data Source System** (`internal/data_source/` + `pkg/data_source/`)

Factory-based plugin system for database support:

| Database | Type Constant | Implementation | Features |
|----------|---------------|----------------|----------|
| MySQL | `mysql-row` | `pkg/data_source/mysql/` | Connection pooling, DML operations |
| Redis | `redis` | `pkg/data_source/redis/` | Standalone & Cluster support |
| MongoDB | `mongodb` | `pkg/data_source/mongodb/` | Document operations |
| ClickHouse | `clickhouse` | `pkg/data_source/clickhouse/` | Columnar queries |
| Kafka | `kafka` | `pkg/data_source/kafka/` | Message read/write |

**DataSource Interface**:
```go
type DataSource interface {
    Configure(pipelineName string, data map[string]any) error
    CreateDataSource(dataSourceName string) (any, error)
    DataSourceConfig(dataSourceName string) (any, error)
}
```

#### 4. **Message System** (`internal/message/`)

Abstract messaging interface for different database operations:

- `Message`: Base interface with `Type()` and `StartTime()`
- `MySQLRowMessage`: Single row DML operations (INSERT/UPDATE/DELETE/SELECT)
- `MySQLTransactionMessage`: Multi-row transaction operations with GTID
- `MySQLDDLMessage`: DDL operations (ALTER TABLE, etc.)

**Metadata Structure**: Contains operation, database, table, hint, write type, and timestamp

#### 5. **Pressure Testing Plugins** (`pkg/pressure/`)

**Plugin Interface**:
```go
type Pressure interface {
    Configure(pipeline string, data map[string]any) error
    Start(ctx context.Context) error
    Execute(msg message.Message)
    Close() error
}
```

##### MySQL DML Pressure (`pkg/pressure/mysql-row/`)

- **Type**: `mysql-dml`
- Worker thread pool architecture with configurable concurrency
- Features:
  - Supports INSERT, UPDATE, DELETE, SELECT
  - Multiple insert modes (INSERT IGNORE, INSERT ON DUPLICATE KEY, REPLACE)
  - SQL comment/hint support
  - Transaction handling
- **Config**: `DataSource`, `WorkerCount`, `WorkerQueueLength`

##### MySQL DDL Pressure (`pkg/pressure/mysql-ddl/`)

- **Type**: `mysql-ddl`
- Integrates gh-ost for online DDL migrations
- Features:
  - ALTER TABLE, RENAME TABLE, CREATE/DROP operations
  - Concurrent migration limit with dependency management
  - Graceful shutdown with timeout
  - MigrationTask tracking per schema.table
- **Config**: `DataSource`, `GhostBinary`, `MaxConcurrent`, `ChunkSize`, `MaxLoad`, `ExecuteChanges`, `Timeout`, `CutOver`, `DropOldTable`

### Key Design Patterns

1. **Factory Pattern**: All plugins (Config, DataSource, Pressure) registered and retrieved via factory methods
2. **Plugin Architecture**: Easy to add new databases, pressure engines, or config sources
3. **Worker Pool**: MySQL DML uses worker threads for concurrent operations
4. **Context-based Cancellation**: Extensive use of `context.Context` for timeout and cancellation
5. **Resource Management**: `Close()` methods for graceful cleanup

## Testing Patterns

**Location**: `/test/pressure/`

**Test Files**:
- `pressure_test.go`: Base test setup and utilities
- `mysql_row_test.go`: Row-level DML pressure tests
- `mysql_ddl_test.go`: DDL migration tests

**Test Setup**:
- MySQL connection: `127.0.0.1:3306`
- Log level: DEBUG
- Uses `TestMain` for environment initialization

## File Organization

```
kyogre/
├── cmd/
│   └── main.go                      # Entry point
├── internal/                         # Unexported packages
│   ├── app/
│   │   └── app.go                   # Server struct and lifecycle
│   ├── config/
│   │   ├── config.go                # Config factory and loading
│   │   └── base.go                  # ConfigManager interface
│   ├── data_source/
│   │   └── data_source.go           # DataSource factory and registry
│   ├── message/
│   │   ├── message.go               # Message interface
│   │   └── mysql.go                 # MySQL message types
│   └── plugin/
│       ├── pressure.go              # Pressure plugin interface
│       ├── generator.go             # Generator interface
│       └── scenario.go              # Scenario interface (placeholder)
├── pkg/                             # Exported packages
│   ├── config/
│   │   └── file.go                  # FileConfig implementation
│   ├── data_source/
│   │   ├── data_source.go           # Initializer
│   │   ├── mysql/
│   │   ├── redis/
│   │   ├── mongodb/
│   │   ├── clickhouse/
│   │   └── kafka/
│   └── pressure/
│       ├── mysql-row/               # DML pressure engine
│       │   ├── pressure.go
│       │   ├── worker.go
│       │   └── config.go
│       └── mysql-ddl/               # DDL pressure engine
│           ├── pressure.go
│           └── config.go
└── test/
    └── pressure/
        ├── pressure_test.go
        ├── mysql_row_test.go
        └── mysql_ddl_test.go
```

## Important Dependencies

- `github.com/xuenqlve/common`: Common library (data sources, error handling, logging, schema management)
- `github.com/mitchellh/mapstructure`: Struct mapping from maps
- `github.com/pingcap/tidb/pkg/parser`: SQL parsing
- Database drivers: mysql, redis, mongodb, clickhouse, kafka
- Logging: zerolog and zap via common library

## Execution Flow

```
main() in cmd/main.go
  ↓
config.NewConfig()          # Load YAML/TOML configuration
  ↓
app.NewServer(cfg)          # Create server instance
  ├── log.Init()            # Initialize logging
  └── server.Configure()    # Register data sources
      └── DataSource plugins configure connections
  ↓
server.Run()                # Start application
  └── Launch pressure plugins to accept and process messages
```

## When Adding New Features

### Adding a New Data Source
1. Create directory: `pkg/data_source/{database-name}/`
2. Implement `DataSource` interface in two files:
   - `data_source.go`: Configuration and initialization
   - `{database-name}.go`: Client and operations
3. Register in `init()` function: `data_source.RegisterPlugin(type, implementation, singleton)`
4. Reference: See `pkg/data_source/mysql/` for pattern

### Adding a New Pressure Engine
1. Create directory: `pkg/pressure/{engine-name}/`
2. Implement `Pressure` interface:
   - `Configure()`: Parse configuration
   - `Start()`: Initialize resources
   - `Execute()`: Process messages
   - `Close()`: Cleanup
3. Create `config.go` for config struct
4. Register in `init()`: `pressure.RegisterPressure(type, implementation, singleton)`
5. Reference: See `pkg/pressure/mysql-row/` for simple pattern or `mysql-ddl/` for complex

### Adding Tests
- Follow patterns in `/test/pressure/`
- Use `TestMain` for setup/teardown
- Tests should be runnable in isolation with proper database connection

## Debugging Tips

1. **Logging**: Set log level in config (DEBUG shows detailed execution)
2. **Context Timeouts**: Check `context.Context` cancellation in pressure engines
3. **Database Connections**: Verify MySQL is running on `localhost:3306` for tests
4. **Worker Deadlocks**: Monitor worker queue lengths in MySQL DML pressure
5. **Ghost Migrations**: Ensure gh-ost binary is accessible and MySQL binary log is enabled

## Current Development Focus

The codebase is actively refactoring towards:
- Plugin-based architecture with cleaner separation of concerns
- Transitioning from gh-ost to native MySQL DDL pressure testing
- Enhanced message system for different database types
- Improved configuration management with factory pattern

Recent changes show restructuring of pressure test engines and migration from older patterns to the current factory-based plugin system.
