# Operations

This guide covers the commands and endpoints operators use after GONK is running.

## Admin Access

If `admin.require_auth` is enabled, export the same token used by the gateway config:

```bash
export GONK_ADMIN_TOKEN="change-me"
```

Use `--url` when the gateway is not on `http://localhost:8080`:

```bash
gonk-cli --url http://edge-gateway.local:8080 routes list
```

## Health

```bash
gonk-cli status
gonk-cli health
curl http://localhost:8080/_gonk/health
curl http://localhost:8080/_gonk/ready
curl http://localhost:8080/_gonk/live
```

Health endpoints are intentionally simple for process supervisors and container orchestrators.

`gonk-cli status` uses `/_gonk/status` and includes runtime mode, admin protection, audit state, route summaries, upstreams, cache totals, and circuit breaker state.

## Routes

Runtime route introspection is available through the CLI:

```bash
gonk-cli routes list
gonk-cli routes describe protected-api
```

To append a simple route to a YAML file:

```bash
gonk-cli routes add \
  -c gonk.yaml \
  --name public-status \
  --path /status \
  --upstream http://status-service:3000 \
  --methods GET \
  --auth none
```

This rewrites the YAML file, so use it for simple operational edits. For heavily commented configs, prefer editing by hand and validating afterward.

## Cache

```bash
gonk-cli cache stats
gonk-cli cache clear
```

Cache stats include entries, fresh/expired counts, cached bytes, hits, and misses per route.

## Metrics

```bash
gonk-cli metrics
gonk-cli metrics --route protected-api
curl http://localhost:8080/metrics
```

Request metrics use stable route labels instead of raw request paths. This keeps Prometheus cardinality bounded when URLs contain IDs.

## Audit

Enable audit logs in config:

```yaml
audit:
  enabled: true
```

Audit records are written through the configured logger. They include route, method, path, status, duration, client IP, identity type, identity, roles, and scopes.

## Production Mode

Use production mode to reject demo secrets:

```yaml
runtime:
  environment: production
```

For demos that intentionally use sample secrets, set `runtime.allow_demo_secrets: true`. Do not carry that flag into real deployments.

## Logs

When logging to a file:

```bash
gonk-cli logs tail --file /var/log/gonk/gonk.log
gonk-cli logs tail --file /var/log/gonk/gonk.log --route protected-api
```

When running under systemd or Docker, use the platform log collector directly.

## Validation

Before restart, reload, or rollout:

```bash
gonk-cli validate -c gonk.yaml
go test ./...
```

For a clean Linux verification environment:

```bash
docker run --rm -v "$PWD:/app" -w /app golang:1.21 go test ./...
```

## Production Deploy

See [DEPLOYMENT.md](DEPLOYMENT.md) for the release archives, GHCR image, Docker Compose production template, and systemd unit.

## Smoke Tests

Run the quickstart smoke test locally:

```bash
make demo-smoke
```

This starts the Docker Compose quickstart, calls a public route, generates a JWT, calls a protected route, and verifies Prometheus metrics.

Run the mTLS walkthrough:

```bash
make mtls-demo
```

For offline PKI bootstrap, validation, and certificate rotation, see [AIRGAP_PKI.md](AIRGAP_PKI.md).
