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

## Current Coverage Focus

- Configuration loading, environment fallbacks, and repository example validation.
- JWT and API key authentication middleware.
- Combined JWT plus client-certificate requirements.
- HTTP reverse proxy path rewriting and injected headers.
- Load balancer health-check URL selection and unhealthy upstream skipping.
- Response cache hit behavior and header preservation.
- Rate limiting rejection after burst exhaustion.

## What Still Needs Deeper Coverage

- mTLS with generated certificate chains.
- WebSocket and gRPC proxy behavior.
- Circuit breaker state transitions.
- Hot config reload behavior.
- Full Docker Compose quickstart smoke test in CI when Docker Compose v2 is available.
