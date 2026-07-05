# Threat Model

This document describes the security boundaries GONK is designed for and the risks operators still need to manage.

## Scope

GONK is an edge API gateway. It can enforce local routing, authentication, authorization, mTLS, rate limits, cache policy, circuit breakers, audit logs, and operational endpoint protection.

GONK is not a hosted identity provider, certificate authority, secret manager, SIEM, WAF, or multi-tenant developer portal.

## Assets

- Upstream services behind GONK.
- Device, user, and service identities.
- JWT secrets, API keys, admin tokens, and private keys.
- Routing, authorization, and rate-limit policy in `gonk.yaml`.
- Admin endpoints under `/_gonk/*` and the configured metrics path.
- Audit logs and metrics that may reveal topology or identities.

## Trust Boundaries

- External clients are untrusted until authenticated by JWT, API key, or client certificate.
- Upstream services are trusted to process requests forwarded by GONK.
- The local host, container runtime, mounted config files, and secret injection mechanism are trusted infrastructure.
- Admin endpoints are trusted operations surfaces and should be reachable only from operator networks.
- Forwarded headers from clients are not a substitute for network-level trust.

## Main Threats

### Exposed Admin Endpoints

Admin endpoints can reveal route topology and operational state, and some endpoints can change state. In production:

- Enable `admin.require_auth`.
- Set `GONK_ADMIN_TOKEN` through a secret manager or environment injection.
- Restrict `admin.allowed_cidrs`.
- Keep `/metrics` on a trusted network or protected by admin auth.

### Weak Or Leaked Secrets

Demo secrets are rejected in production mode, but operators still own secret lifecycle.

- Use `runtime.environment: production`.
- Keep JWT secrets, API keys, and admin tokens out of git.
- Prefer environment expansion such as `${JWT_SECRET}`.
- Rotate API keys and admin tokens through your deployment system.

### Incorrect mTLS Bootstrap

If the root CA is not installed before services start, internal TLS failures can look like application outages.

- Establish the trust anchor first.
- Use a dedicated device CA or intermediate.
- Run `gonk-cli certs doctor` before deploy.
- Keep CA private keys offline where possible.

### Over-Broad Route Authorization

A route protected only by a broad role can still be too permissive for industrial or device workflows.

- Use route-local permissions for write, actuator, admin, and control-plane paths.
- Prefer method-specific rules for device identities.
- Use `require_client_cert` or `require_either` only when every accepted mechanism grants equivalent risk.

### Untrusted Upstreams

GONK does not sandbox upstream responses. A compromised upstream can still return malicious payloads.

- Treat upstream security as part of the same trust boundary.
- Use dedicated networks and least-privilege service access.
- Keep audit and upstream health signals enabled for critical routes.

### Denial Of Service

GONK can rate-limit and shed failing upstreams, but host resources are finite.

- Configure global or route-level rate limits.
- Use circuit breakers for fragile upstreams.
- Put public deployments behind network firewalls or load balancers.
- Monitor request rates, status codes, memory, CPU, and upstream health.

## Production Baseline

Run these checks before deploying:

```bash
gonk-cli validate -c gonk.yaml
gonk-cli doctor -c gonk.yaml
gonk-cli certs doctor -c gonk.yaml --server-name edge-gateway.local
```

When the gateway is running:

```bash
export GONK_ADMIN_TOKEN="..."
gonk-cli --url http://localhost:8080 doctor -c gonk.yaml --check-admin --check-upstreams
gonk-cli --url http://localhost:8080 status
```

## Known Limitations

- No built-in secret storage or encryption-at-rest for config files.
- No certificate revocation list or OCSP enforcement.
- No hosted control plane, fleet inventory, or policy distribution service.
- No WAF signature engine.
- No automatic release pipeline; releases are manual by design.

These are deliberate boundaries for the current project. Keep deployments small, explicit, and observable.
