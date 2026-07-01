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

- Release-grade artifacts with checksums and a documented release process.
- End-to-end Docker Compose smoke test in CI.
- Deeper docs for the three primary use cases: industrial IoT, microservices edge, and air-gapped admin access.
- Hardening tests for generated certificate chains, WebSocket/gRPC proxy behavior, and config reload.
- A small dashboard or richer terminal status view for operators.

## Commercial Roadmap

1. Community Edition: single-node gateway, config templates, Docker image, local examples.
2. Team Edition: dashboard, audit logs, config history, signed bundles, policy packs.
3. Enterprise Edition: fleet management, certificate lifecycle automation, SSO, HA coordination, support contracts.

## First 30-Day Milestones

- Keep `make build`, `make test`, and `make docker-build` passing in a clean environment.
- Add release artifacts with checksums for Linux, macOS, and Windows.
- Add a CI smoke test for the quickstart Docker Compose stack.
- Publish one complete industrial IoT walkthrough with mTLS, API key fallback, JWT operators, and Prometheus.
- Add a small operator status view over route, cache, upstream, and circuit breaker state.
