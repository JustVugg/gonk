# GONK Startup Brief

## Positioning

GONK should be positioned as an edge-native API gateway for industrial, IoT, robotics, and air-gapped environments where teams need secure service exposure without running a heavy control plane.

The narrow wedge is: "single-binary zero-control-plane gateway for secure edge networks."

## Ideal Customer Profile

- Industrial automation teams connecting PLCs, sensors, HMIs, and backend services.
- Robotics and field-device teams that deploy services on gateways with limited compute.
- Defense, utilities, logistics, and healthcare teams with air-gapped or intermittently connected networks.
- Platform engineers who need mTLS, JWT, API keys, rate limits, and observability near the device layer.

## Why It Can Be Appealing

- Lightweight deployment: one Go binary plus YAML.
- Security-first story: mTLS, JWT, API keys, RBAC, scopes, and certificate-to-role mapping.
- Edge-ready story: works without Kubernetes, cloud accounts, or centralized control planes.
- Practical operator workflow: templates, validation, metrics, health endpoints, hot reload.

## Product Gaps To Close

- Reproducible builds and CI on every PR.
- End-to-end examples that can run locally with Docker Compose.
- Clear docs for the three primary use cases: industrial IoT, microservices edge, and air-gapped admin access.
- Hardening tests for auth combinations, route matching, load balancing, cache behavior, and config reload.
- A small dashboard or CLI status view that makes the gateway feel operationally polished.

## Commercial Roadmap

1. Community Edition: single-node gateway, config templates, Docker image, local examples.
2. Team Edition: dashboard, audit logs, config history, signed bundles, policy packs.
3. Enterprise Edition: fleet management, certificate lifecycle automation, SSO, HA coordination, support contracts.

## First 30-Day Milestones

- Make `make build`, `make test`, and `make docker-build` pass in a clean environment.
- Add GitHub Actions for test, lint, Docker build, and release artifacts.
- Publish a one-command demo with backend, gateway, Prometheus, and sample tokens.
- Rewrite README around the edge/industrial wedge instead of a generic API gateway.
- Add a security model document that explains JWT, API key, mTLS, and combined auth modes.
