# Secure Edge API Example

This example shows a realistic edge deployment:

- public status route without auth;
- JWT-protected machine data route;
- stricter command route for engineer/admin roles;
- weighted load balancing across two upstream services;
- health checks, rate limiting, circuit breaker, response cache, audit logs, metrics, and protected admin endpoints.

## Run

```bash
docker compose -f examples/secure-edge-api/docker-compose.yml up --build
```

If your system uses the legacy Compose binary:

```bash
docker-compose -f examples/secure-edge-api/docker-compose.yml up --build
```

## Test Public Access

```bash
curl http://localhost:8080/public/status
```

## Generate A JWT

```bash
docker compose -f examples/secure-edge-api/docker-compose.yml run --rm --entrypoint gonk-cli gonk \
  auth jwt generate \
  --role engineer \
  --scopes read:api,write:commands \
  --user-id demo-engineer \
  --expiry 1h
```

Copy the token, then call protected routes:

```bash
curl -H "Authorization: Bearer <token>" http://localhost:8080/api/machines/line-1
curl -X POST -H "Authorization: Bearer <token>" http://localhost:8080/api/commands/open-valve
```

## Admin Status

Admin and metrics endpoints require `X-Gonk-Admin-Token`:

```bash
curl -H "X-Gonk-Admin-Token: demo-admin-token" http://localhost:8080/_gonk/status
curl -H "X-Gonk-Admin-Token: demo-admin-token" http://localhost:8080/metrics
```

## Stop

```bash
docker compose -f examples/secure-edge-api/docker-compose.yml down --remove-orphans
```
