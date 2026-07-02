## Configuration Examples

The `examples/` directory contains runnable demos and complete configurations:

**Quickstart Demo** (`examples/quickstart/`)
Runs GONK, two backend services, and Prometheus with Docker Compose. This is the fastest way to try the project end to end.

**Secure Edge API** (`examples/secure-edge-api/`)
Runs a realistic protected API with public status, JWT authorization, admin protection, audit logs, metrics, rate limiting, cache, circuit breaker, and two upstream services.

**mTLS Demo** (`examples/mtls/`)
Generates a local CA plus signed server/client certificates, then runs GONK with client certificate authentication.

**Basic Setup** (`examples/basic/gonk.yaml`)  
Simple configuration for development and testing. Uses JWT authentication with role-based permissions.

**Industrial IoT** (`examples/industrial-iot/gonk.yaml`)  
Production setup for factory environments. Devices authenticate with mTLS certificates, users with JWT tokens. Includes rate limiting and caching for sensor data.

**Microservices** (`examples/microservices/gonk.yaml`)  
Load balancing across multiple service instances with health checks, circuit breakers, and response caching. Supports HTTP, gRPC, and WebSocket protocols.

Run any example:
```bash
./bin/gonk -config examples/industrial-iot/gonk.yaml
```

Run the quickstart demo:
```bash
make demo-up
```

Run the quickstart smoke test:
```bash
make demo-smoke
```

Run the mTLS demo:
```bash
make mtls-demo
```

Validate before deploying:
```bash
./bin/gonk-cli validate -c examples/microservices/gonk.yaml
```
