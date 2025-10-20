# Repository Guidelines

## Project Structure & Module Organization
Kyogre is a Go 1.24 project centered on a plugin-driven workload engine. CLI entry lives in `cmd/main.go`. Core logic is under `internal/`: `config` handles multi-format config parsing, `data_source` registers database plugins, `common` wraps logging/errors, and `generator`, `metrics`, `scenario`, `workload`, `report` are scaffolds awaiting full implementation. Shared helpers and concrete client code sit in `pkg/` (for example `pkg/data_source/mysql`). Vendored dependencies are kept in `vendor/` to guarantee reproducible builds. Operational notes and examples are stored alongside localized docs in `运行步骤/`.

## Build, Test, and Development Commands
```bash
go mod download            # sync Go modules with vendor
go build -o kyogre ./cmd   # compile CLI binary
GO111MODULE=on go run ./cmd --help  # inspect CLI flags once wired
go test ./...              # run all package tests
go fmt ./...               # format sources before commits
go vet ./...               # static analysis for common mistakes
```
Run targeted tests with `go test -v ./pkg/data_source/mysql` (swap path per plugin).

## Coding Style & Naming Conventions
Always format with `gofmt` (tabs, goimports ordering). Keep package names lowercase and concise (`metrics`, not `Metrics`). Exported symbols use PascalCase, unexported helpers camelCase. New plugins should register themselves via `init()` in `internal/data_source`. Use `internal/common/log` for structured logging and avoid direct `fmt.Print` in production paths.

## Testing Guidelines
Follow Go’s standard `*_test.go` pattern colocated with the package under test (e.g., `internal/config/parser_test.go`). Prefer table-driven tests and mock configurable dependencies via small interfaces. Validate configuration edge cases and connection error paths before layering scenario logic. Use `go test -race ./...` when touching concurrency primitives once the race detector is wired into CI.

## Commit & Pull Request Guidelines
This workspace lacks Git history, so adopt Conventional Commits (`feat: add redis workload tracer`). Keep commits focused and gofmt-clean. Pull requests should explain motivation, summarize impacted directories (`internal/metrics`, `pkg/data_source/redis`), list test commands executed, and link related issues (`Fixes #123`). Include log snippets or screenshots when altering reports or monitoring output.

## Security & Configuration Tips
Never commit live credentials; keep sample configs under `config/` with placeholder values. Document required environment variables and TLS knobs in README updates when adding new data sources. Clean up experimental load data or debug binaries before submitting a PR to avoid leaking sensitive details.
