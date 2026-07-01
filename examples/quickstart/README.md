# Quickstart Demo

This demo runs GONK, two demo upstream services, and Prometheus.

Prerequisite: Docker Compose v2 (`docker compose`).

## Start

```bash
docker compose -f examples/quickstart/docker-compose.yml up --build
```

GONK listens on `http://localhost:8080`.
Prometheus listens on `http://localhost:9090`.

## Try A Public Route

```bash
curl http://localhost:8080/public/ping
```

## Generate A Demo JWT

```bash
docker compose -f examples/quickstart/docker-compose.yml run --rm --entrypoint gonk-cli gonk \
  auth jwt generate \
  --role user \
  --scopes read:api \
  --user-id demo-user \
  --expiry 24h
```

Copy the token from the command output, then call the protected route:

```bash
TOKEN="<paste-token-here>"
curl -H "Authorization: Bearer $TOKEN" http://localhost:8080/api/ping
```

## Useful Endpoints

```bash
curl http://localhost:8080/_gonk/health
curl http://localhost:8080/_gonk/info
curl http://localhost:8080/metrics
```

## Stop

```bash
docker compose -f examples/quickstart/docker-compose.yml down --remove-orphans
```

## Smoke Test

```bash
make demo-smoke
```

The smoke test starts the stack, calls a public route, generates a JWT, calls a protected route, checks metrics, and then tears the stack down.
