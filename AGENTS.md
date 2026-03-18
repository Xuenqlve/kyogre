# Repository Guidelines

## Project Structure & Module Organization
Kyogre is a Go 1.24 stress-testing tool for MySQL, Redis, MongoDB, ClickHouse, and Kafka. The entrypoint is `cmd/main.go`. Core orchestration lives under `internal/` (`app/`, `config/`, `plugin/`, `message/`, `metrics/`). Reusable implementations live in `pkg/`, including data sources, generators, metadata, pressure engines, scenarios, and helper tooling. Integration-style and package tests are grouped under `test/`. Runtime config examples currently live in `config/`, and vendored dependencies are checked into `vendor/`.

## Build, Test, and Development Commands
- `go build -o ./bin/kyogre ./cmd/...`: build the binary locally.
- `go test ./...`: run the full test suite.
- `go test -v ./test/...`: run repository tests with verbose output.
- `go test -cover ./...`: check package coverage before larger changes.
- `go test -race ./...`: catch concurrency issues in plugin and pressure code.
- `go fmt ./... && go vet ./...`: format code and detect suspicious constructs.

## Coding Style & Naming Conventions
Follow standard Go formatting with tabs and `gofmt` output as the source of truth. Keep packages lowercase (`pkg/data_source/mysql`), exported identifiers in `CamelCase`, and unexported helpers in `camelCase`. Match existing naming patterns such as `Manager`, `Controller`, `Generator`, and `Pressure` for plugin-facing types. Prefer small structs, explicit error returns, and minimal comments that explain intent rather than mechanics.

## Testing Guidelines
Add tests alongside the current suite under `test/` and use Go’s `*_test.go` naming. Prefer table-driven tests for generators, metadata builders, and config parsing. Run targeted tests while iterating, for example `go test -v -run TestMySQLRowPressure ./test/pressure/...`, then finish with `go test ./...`. Any change to concurrency, scheduling, or queue behavior should also be checked with `go test -race ./...`.

## Commit & Pull Request Guidelines
Recent history uses short messages such as `up` and `iquery`; for new work, keep commits brief but descriptive, for example `fix mysql ddl builder` or `add generator snapshot tests`. Scope each commit to one logical change. Pull requests should explain the problem, summarize the approach, list verification commands, and note config or behavior changes. Include sample output or screenshots only when a CLI flow or generated artifact changes.
