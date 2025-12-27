## Configuration Examples

The `examples/` directory contains three complete configurations:

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

Validate before deploying:
```bash
./bin/gonk-cli validate -c examples/microservices/gonk.yaml
```
