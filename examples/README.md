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

## Files

- `config.yaml`: runnable mock pipeline example for local smoke testing
- `schema.yaml`: reference snippet for `metadata.type: mysql` schema configuration

## Adapting To MySQL

To build a real MySQL config, keep the overall shape from `config.yaml` and replace the mock sections with:

- a MySQL datasource under `data-source`
- `metadata.type: mysql`
- a MySQL generator and pressure engine
- the schema fragment from `schema.yaml`

After that, validate with:

```bash
go test -v ./test/metadata/...
mysql -h localhost -u root -p kyogre_demo -e "SHOW TABLES;"
```
