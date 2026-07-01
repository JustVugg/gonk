# Security Model

GONK is designed to run close to edge services in constrained or disconnected networks. Treat the gateway as security-sensitive infrastructure: keep secrets outside the repository, restrict operational endpoints, and test policy changes before deploying them.

## Authentication Layers

GONK supports three local authentication mechanisms:

- JWT for user and service identities that already have a token issuer.
- API keys for simple device or service authentication.
- mTLS for devices or operators that can present client certificates.

Routes can require one mechanism, combine JWT with a client certificate, or accept one of several mechanisms with `require_either`.

## Authorization

Authorization is route-local and YAML-driven:

- `allowed_roles` restricts route access to identities with at least one listed role.
- `required_scopes` requires all listed scopes.
- `permissions` maps roles or identity types to allowed HTTP methods and optional scopes.

Prefer route-level permissions for industrial control paths. A broad JWT role check is usually not enough for actuator, admin, or write-heavy device workflows.

## Admin Endpoints

Operational endpoints live under `/_gonk/*`; metrics are exposed at the configured metrics path. These endpoints can reveal routing topology or perform state changes, so production deployments should protect them:

```yaml
admin:
  require_auth: true
  header: "X-Gonk-Admin-Token"
  token: "${GONK_ADMIN_TOKEN}"
  allowed_cidrs: ["127.0.0.1/32", "10.0.0.0/8"]
```

When `require_auth` is enabled, CLI commands send the token from `GONK_ADMIN_TOKEN`. CIDR allowlists are checked against the direct client address, not forwarded headers.

Health, liveness, and readiness endpoints remain public by default so orchestrators can probe the gateway. Put the gateway behind a local network boundary if those probes should not be internet-visible.

## Secret Handling

Use environment expansion in YAML:

```yaml
auth:
  jwt:
    secret_key: "${JWT_SECRET}"
```

Avoid copying demo secrets into production. The CLI can generate demo JWTs with a fallback secret for local testing, but production services should always set `JWT_SECRET`.

Set production mode to make this enforceable:

```yaml
runtime:
  environment: production
```

In production mode, GONK rejects known demo secrets such as `change-me`, `change-me-in-production`, `change-me-admin-token`, and generated example placeholders. Use `runtime.allow_demo_secrets: true` only for demos that intentionally look like production configs.

API key comparison is constant-time inside the gateway. Store API keys in environment variables or injected config bundles, not in committed files.

## mTLS

For device fleets, prefer a dedicated device CA and `client_auth: "require"` when every caller should present a certificate. Use certificate-to-role mapping for coarse device categories, then route permissions for method-level control.

For mixed user/device routes, use `require_either` only when both accepted mechanisms grant the same operational risk. For admin routes, prefer JWT plus `require_client_cert`.

The CLI can generate a simple local chain for demos:

```bash
gonk-cli certs generate --type ca --cn "GONK CA" --output ./certs
gonk-cli certs generate --type server --cn localhost --output ./certs --ca-cert ./certs/ca.crt --ca-key ./certs/ca.key
gonk-cli certs generate --type client --cn Device-001 --output ./certs --ca-cert ./certs/ca.crt --ca-key ./certs/ca.key
```

For a complete runnable example, use `make mtls-demo`.

## Audit Logging

Enable route-level audit logs:

```yaml
audit:
  enabled: true
```

Audit entries include route, method, path, status, duration, client IP, identity type, identity, roles, and scopes. They use the configured logging output, so send logs to your platform collector or a protected file path.

## Deployment Checklist

- Set `GONK_ADMIN_TOKEN`, `JWT_SECRET`, and API keys through the environment or secret manager.
- Enable `admin.require_auth` and restrict `admin.allowed_cidrs`.
- Keep `/metrics` on a trusted network or protect it through admin auth.
- Use TLS and mTLS for untrusted networks.
- Set `runtime.environment: production` and remove `runtime.allow_demo_secrets` for real deployments.
- Enable `audit.enabled` for controlled environments and write logs to a protected sink.
- Validate config with `gonk-cli validate -c gonk.yaml` before restart or hot reload.
- Run `gonk-cli status`, `gonk-cli routes list`, `gonk-cli routes describe <name>`, and `gonk-cli cache stats` after deployment.
