# Pulsewatch server

The Go server contains the REST API, monitor execution, background producer/worker/ingester services, database migrations, and the Uptime Kuma importer.

## Development

Run commands from this directory:

```bash
go test ./...
go build ./...
```

The API serves Swagger UI at `http://localhost:8034/swagger/index.html` when running. Generate or watch the OpenAPI files with `pnpm --dir apps/server run docs:watch` from the repository root.

## Uptime Kuma import

Preview an import before applying changes:

```bash
go run ./cmd/import-uptime-kuma --help
```

The importer defaults to preview mode. Use its explicit apply option only after reviewing the preview; notification-channel secrets require a separate opt-in.

See the repository [README](../../README.md) for the application overview and deployment layout.
