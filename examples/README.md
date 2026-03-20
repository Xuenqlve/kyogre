# Examples

## Quick Start

The fastest way to verify the current pipeline wiring is the mock example:

```bash
env GOCACHE=/tmp/kyogre-go-build-cache go build -o ./bin/kyogre ./cmd/...
./bin/kyogre -config examples/config.yaml
```

This example does not require MySQL. It starts a mock scenario, generates messages, routes them through the in-memory pipeline, and logs them through the mock pressure plugin.

Notes:

- `log-file` is treated as a directory by the shared logging package, so the example uses `/tmp`
- the example enables the HTTP API on `:18080`

## MySQL Base Scenario Example

The repository now also includes a full MySQL example that uses `scenario.type: base` with `builder: mysql`:

```bash
env GOCACHE=/tmp/kyogre-go-build-cache go build -o ./bin/kyogre ./cmd/...
./bin/kyogre -config examples/mysql-config.yaml
```

Before running it, adjust these fields in `examples/mysql-config.yaml`:

- `data-source.source.host`
- `data-source.source.port`
- `data-source.source.username`
- `data-source.source.password`
- `data-source.source.schema_store`

This example will:

- create `kyogre_demo` and the example tables if they do not exist
- use `scenario.type: base`
- load MySQL targets through `builder: mysql`
- generate MySQL row messages with `generator.type: mysql`
- execute them with `pressure.type: mysql-dml`

## Files

- `config.yaml`: runnable mock pipeline example for local smoke testing
- `mysql-config.yaml`: runnable MySQL example using `scenario.type: base` and `builder: mysql`
- `schema.yaml`: reference snippet for `metadata.type: mysql` schema configuration

## Adapting To MySQL

If you are building your own MySQL config, start from `mysql-config.yaml` and adjust:

- datasource credentials
- metadata schema definitions
- target selector / row count strategy
- pressure worker settings

After that, validate with:

```bash
go test -v ./test/metadata/...
mysql -h localhost -u root -p kyogre_demo -e "SHOW TABLES;"
```
