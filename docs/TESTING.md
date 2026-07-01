# Testing

GONK uses standard Go tests colocated with the package they validate. Test files use the `*_test.go` suffix.

## Commands

```bash
go test ./...
make test
make test-coverage
make test-race
```

Coverage output is written to `coverage.out` by `make test-coverage`.

When the host Go toolchain is not available, run the suite in Docker:

```bash
docker run --rm -v "$PWD:/app" -w /app golang:1.21 go test ./...
docker run --rm -v "$PWD:/app" -w /app golang:1.21 make build
```

Run the Docker Compose quickstart smoke test:

```bash
make demo-smoke
```

## Current Coverage Focus

- Configuration loading, environment fallbacks, and repository example validation.
- JWT and API key authentication middleware.
- Combined JWT plus client-certificate requirements.
- HTTP reverse proxy path rewriting and injected headers.
- Load balancer health-check URL selection and unhealthy upstream skipping.
- Response cache hit behavior and header preservation.
- Rate limiting rejection after burst exhaustion.
- Admin endpoint token/CIDR enforcement and route introspection.
- Cache statistics for entries, bytes, hits, and misses.
- Production secret guardrails.
- Operational status endpoint coverage.

## What Still Needs Deeper Coverage

- WebSocket and gRPC proxy behavior.
- Hot config reload behavior.
- Full mTLS Docker Compose smoke test in CI when runtime budget allows it.
