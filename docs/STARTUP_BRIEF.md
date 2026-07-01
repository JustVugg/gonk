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

- Signed release artifacts and a documented release checklist.
- Deeper docs for air-gapped admin access and certificate lifecycle operations.
- Hardening tests for WebSocket/gRPC proxy behavior, config reload cleanup, and long-running health checks.
- Fleet-level status/dashboard for multi-node deployments.
- CI mTLS smoke test once runtime budget allows an additional Docker Compose stack.

## Commercial Roadmap

1. Community Edition: single-node gateway, config templates, Docker image, local examples.
2. Team Edition: dashboard, audit logs, config history, signed bundles, policy packs.
3. Enterprise Edition: fleet management, certificate lifecycle automation, SSO, HA coordination, support contracts.

## First 30-Day Milestones

- Keep `make build`, `make test`, `make demo-smoke`, and `make docker-build` passing in CI.
- Publish a tagged release using the release workflow and verify checksums.
- Add one complete industrial IoT walkthrough with mTLS, API key fallback, JWT operators, and Prometheus.
- Add config reload stress tests for repeated route/upstream changes.
- Add WebSocket and gRPC smoke tests with real upstream services.
