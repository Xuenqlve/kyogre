# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**Kyogre** is a high-performance database load testing tool that supports multiple databases: MySQL, MongoDB, Redis, Kafka, and ClickHouse. The project is written in Go 1.24.9+ and uses a plugin-based architecture for extensibility.

The project is approximately 50% complete with robust infrastructure but several core modules still pending implementation.

## Build and Development Commands

```bash
# Build the project
go build -o kyogre ./cmd

# Install dependencies (already vendored)
go mod download
go mod tidy

# Run all tests
go test ./...

# Run tests for a specific package
go test -v ./pkg/data_source/mysql

# Run a specific test
go test -run TestName ./path/to/package -v

# Check code quality
go fmt ./...
go vet ./...

# Run linter (if installed)
golangci-lint run ./...
```

## Project Architecture

### Plugin-Based Data Source System

The core architecture revolves around a plugin registration system in `internal/data_source/data_source.go`:

- Each data source (MySQL, MongoDB, Redis, Kafka, ClickHouse) registers itself as a plugin
- Plugins implement the `DataSource` interface with `Configure()` and `CreateDataSource()` methods
- Factory pattern (`GetDataSource()`) dynamically creates data source instances
- Each implementation located in `pkg/data_source/{database_type}/`

### Directory Structure

```
cmd/                          # CLI application (main.go is empty, needs implementation)
internal/
  ├── config/                 # Configuration parsing (YAML/JSON/TOML support)
  ├── data_source/            # Plugin system and registration
  ├── common/
  │   ├── log/               # Zerolog-based structured logging
  │   ├── errors/            # Error handling and wrapping
  │   └── utils/             # Utility functions
  ├── generator/             # Data generation (NOT IMPLEMENTED)
  ├── metrics/               # Metrics collection (NOT IMPLEMENTED)
  ├── report/                # Report generation (NOT IMPLEMENTED)
  ├── scenario/              # Test scenarios (NOT IMPLEMENTED)
  └── workload/              # Workload engine (NOT IMPLEMENTED)

pkg/data_source/
  ├── config.go              # Configuration helpers
  ├── connection.go           # Connection management
  └── {mysql,mongodb,redis,kafka,clickhouse}/
      ├── data_source.go     # Interface implementation
      └── {db}.go            # DB-specific config and connection
```

### Data Source Implementations

**Completed and functional:**
- MySQL: Full SQL support, transactions, DDL, connection pooling
- MongoDB: Document operations, replica sets, TLS, authentication
- Redis: Single-node and cluster modes, custom protocol implementation in `proto/`
- Kafka: Producer/consumer, TLS, SASL authentication
- ClickHouse: Column-oriented database support

### Execution Flow (Expected)

1. Parse CLI arguments and configuration file (YAML/JSON/TOML)
2. Initialize logging and error handlers
3. Load and validate data source configuration
4. Initialize data source connections via plugin system
5. Load test scenarios (when implemented)
6. Generate synthetic data (when implemented)
7. Execute workload with concurrent threads
8. Collect metrics in real-time
9. Generate HTML/JSON reports (when implemented)

### CLI Entry Point

The `cmd/main.go` file is currently empty and needs implementation. It should:
- Accept command-line flags: `--config`, `--scenario`, `--monitor`, `--report`
- Load configuration files and validate them
- Initialize the entire application lifecycle
- Orchestrate the plugin system, workload execution, and reporting

### Key Dependencies

| Package | Version | Purpose |
|---------|---------|---------|
| go-sql-driver/mysql | v1.9.3 | MySQL driver |
| go.mongodb.org/mongo-driver | v1.17.4 | MongoDB driver |
| redis/go-redis | v9.14.0 | Redis client |
| segmentio/kafka-go | v0.4.49 | Kafka client |
| ClickHouse/clickhouse-go | v2.40.3 | ClickHouse driver |
| rs/zerolog | v1.34.0 | Structured logging |

## Code Patterns and Conventions

### Logging

Use the wrapped zerolog functions in `internal/common/log/func.go`:
```go
log.Infof("message")
log.Errorf("error: %v", err)
log.Warnf("warning: %v", err)
log.Debugf("debug info")
```

Logs support console and file output with configurable levels.

### Plugin Registration

Each data source registers in its `init()` function:
```go
func init() {
    data_source.RegisterPlugin(MySQL, &DataSource{}, true)
}
```

The third parameter indicates if it's a singleton (true) or multi-instance (false).

### Configuration Handling

- Configurations are in `internal/config/` package
- Support YAML, JSON, TOML formats
- Use `mapstructure` for config deserialization
- Example configurations in README.md

### Error Handling

Use `internal/common/errors/` package for error wrapping with context from `pingcap/errors`.

## Implementation Status

**Complete:**
- ✅ Plugin system and data source framework
- ✅ All 5 database drivers (MySQL, MongoDB, Redis, Kafka, ClickHouse)
- ✅ Configuration parsing (YAML/JSON/TOML)
- ✅ Logging and error handling infrastructure
- ✅ Connection pooling and lifecycle management

**Pending Implementation:**
- ⏳ `cmd/main.go` - CLI entry point and argument parsing
- ⏳ `internal/generator/` - Synthetic data generation
- ⏳ `internal/metrics/` - Real-time metrics collection (QPS, latency, errors)
- ⏳ `internal/scenario/` - Test scenario parser and execution
- ⏳ `internal/workload/` - Concurrent workload execution engine
- ⏳ `internal/report/` - HTML and JSON report generation

## Common Development Tasks

### Adding a New Data Source

1. Create `pkg/data_source/{database_type}/` directory
2. Implement the `DataSource` interface in `data_source.go`
3. Create `{database_type}.go` for driver-specific config and connection
4. Register the plugin in `init()` function
5. Update imports in relevant files

### Testing Configuration

Since no test files exist yet, create tests following this pattern:
```bash
pkg/data_source/{database_type}/{database_type}_test.go
internal/{module}/{module}_test.go
```

Run with: `go test -v ./...` or `go test -run TestName ./...`

### Debugging

Enable debug logging by setting log level to DEBUG in configuration, or use:
```go
log.Debugf("debug message: %v", value)
```

Check `/tmp/kyogre.log` (or configured log file) for persistent logs.
