<div align="center">
<img src="gonk.png" alt="GONK Logo" width="400">
<br><br>
</div>

GONK is a lightweight API gateway written in Go for edge computing, IoT, and air-gapped environments.

## What's New in v1.1

**Authorization System**
- Role-Based Access Control (RBAC)
- JWT scope validation
- Permission matrix combining roles and HTTP methods
- Support for different identity types (devices vs users)

**mTLS Support**
- Client certificate authentication
- Certificate-to-role mapping with wildcard support
- Dual authentication modes (mTLS + JWT)

**Load Balancing**
- Multiple upstreams per route
- Four strategies: round-robin, weighted, least-connections, ip-hash
- Active health checking with automatic failover

**CLI Tool**
Complete command-line interface for configuration, JWT/certificate generation, and monitoring.

## Installation

```bash
# Clone and build
git clone https://github.com/JustVugg/gonk
cd gonk
make build

# Binaries will be in bin/
./bin/gonk --version
./bin/gonk-cli --version
```

## Quick Start

Generate a basic configuration:

```bash
./bin/gonk-cli init --template basic --output gonk.yaml
```

Start the gateway:

```bash
./bin/gonk -config gonk.yaml
```

Generate a JWT token:

```bash
export JWT_SECRET=change-me
./bin/gonk-cli auth jwt generate --role admin --scopes "read:api,write:api" --user-id alice --expiry 24h
```

Test with the token:

```bash
curl -H "Authorization: Bearer <token>" http://localhost:8080/api/get
```

## Configuration Examples

### Authorization with Permission Matrix

```yaml
auth:
  jwt:
    enabled: true
    secret_key: "change-me-in-production"
    validate_roles: true
    validate_scopes: true

routes:
  - name: "sensor-api"
    path: "/api/sensors/*"
    upstreams:
      - url: "http://backend:3000"
    
    auth:
      type: "jwt"
      required: true
      allowed_roles: ["technician", "engineer", "admin"]
      required_scopes: ["read:sensors"]
      
      permissions:
        - role: "technician"
          methods: ["GET"]
        - role: "engineer"
          methods: ["GET", "POST"]
        - role: "admin"
          methods: ["GET", "POST", "DELETE"]
```

This setup gives technicians read-only access, engineers can read and calibrate, and admins have full control.

### mTLS for Device Authentication

```yaml
server:
  tls:
    enabled: true
    cert_file: "/certs/server.crt"
    key_file: "/certs/server.key"
    client_ca: "/certs/ca.crt"
    client_auth: "require"

routes:
  - name: "device-data"
    path: "/api/devices/*"
    upstreams:
      - url: "http://iot-backend:3000"
    
    auth:
      require_client_cert: true
      cert_to_role_mapping:
        "CN=PLC-001": "device"
        "CN=Sensor-*": "sensor"
        "CN=Admin-*": "admin"
      
      permissions:
        - identity_type: "device"
          methods: ["POST"]
        - role: "admin"
          methods: ["GET", "DELETE"]
```

Devices can only write data, while admins can read and delete.

### Load Balancing with Health Checks

```yaml
routes:
  - name: "api"
    path: "/api/*"
    upstreams:
      - url: "http://backend-1:3000"
        weight: 70
        health_check: "/health"
      - url: "http://backend-2:3000"
        weight: 30
        health_check: "/health"
    
    load_balancing:
      strategy: "weighted"
      health_check_interval: 10s
      health_check_timeout: 5s
```

Traffic is distributed 70/30 between backends. Health checks run every 10 seconds and failed upstreams are automatically removed from rotation.

## CLI Reference

### Server Operations

```bash
gonk -config gonk.yaml              # Start server
gonk-cli validate -c gonk.yaml      # Validate configuration
gonk-cli status                     # Check if server is running
gonk-cli health                     # Server health check
```

### JWT Management

```bash
# Generate token
gonk-cli auth jwt generate --role admin --scopes "read:*,write:*" --user-id alice --expiry 24h

# Validate token
gonk-cli auth jwt validate <token>

# Decode token (no validation)
gonk-cli auth jwt decode <token>
```

### API Keys

```bash
# Generate API key
gonk-cli auth apikey generate --client-id mobile-app --roles user --scopes "read:sensors"

# List configured keys
gonk-cli auth apikey list
```

### Certificate Management

```bash
# Generate CA
gonk-cli certs generate --cn "GONK CA" --type ca --output ./certs

# Generate server cert
gonk-cli certs generate --cn "localhost" --type server --output ./certs

# Generate client cert
gonk-cli certs generate --cn "Device-001" --type client --output ./certs

# Validate cert against CA
gonk-cli certs validate --cert ./certs/client.crt --ca ./certs/ca.crt

# Show cert details
gonk-cli certs info --cert ./certs/client.crt
```

### Monitoring

```bash
gonk-cli metrics                    # Show Prometheus metrics
gonk-cli metrics --route api-v1     # Filter by route
gonk-cli cache stats                # Cache statistics
gonk-cli cache clear                # Clear cache
```

### Configuration Templates

```bash
# Basic template
gonk-cli init --template basic --output gonk.yaml

# Industrial IoT template
gonk-cli init --template industrial --output gonk.yaml

# Microservices template
gonk-cli init --template microservices --output gonk.yaml
```

## Industrial IoT Example

This configuration handles a typical industrial setup with PLCs writing sensor data and engineers monitoring/controlling the system.

```yaml
server:
  listen: ":8443"
  tls:
    enabled: true
    cert_file: "/certs/server.crt"
    key_file: "/certs/server.key"
    client_ca: "/certs/device-ca.crt"
    client_auth: "request"

auth:
  jwt:
    enabled: true
    secret_key: "${JWT_SECRET}"
    validate_roles: true
    validate_scopes: true
  
  api_key:
    enabled: true
    header: "X-API-Key"
    keys:
      - key: "${DEVICE_KEY}"
        client_id: "plc-001"
        roles: ["device"]

routes:
  # Devices write sensor data using mTLS or API key
  - name: "sensor-ingestion"
    path: "/api/sensors/*"
    methods: ["POST"]
    upstreams:
      - url: "http://timeseries-db:8086"
    auth:
      require_either: ["client_cert", "api_key"]
      permissions:
        - identity_type: "device"
          methods: ["POST"]

  # Users read sensor data with JWT
  - name: "sensor-read"
    path: "/api/sensors/*"
    methods: ["GET"]
    upstreams:
      - url: "http://timeseries-db:8086"
    auth:
      type: "jwt"
      required: true
      permissions:
        - role: "technician"
          methods: ["GET"]
        - role: "engineer"
          methods: ["GET"]
    cache:
      enabled: true
      ttl: 30s

  # Only engineers can control actuators
  - name: "actuator-control"
    path: "/api/actuators/*"
    methods: ["POST", "PUT"]
    upstreams:
      - url: "http://plc-gateway:502"
    auth:
      type: "jwt"
      required: true
      allowed_roles: ["engineer", "admin"]
      required_scopes: ["write:actuators"]
    rate_limit:
      requests_per_second: 10
```

## Comparison with Other Gateways

| Feature            | GONK v1.1    | Kong          | NGINX         | Traefik       |
|--------------------|--------------|---------------|---------------|---------------|
| Authorization      | Built-in     | Plugin        | No            | Limited       |
| mTLS               | Built-in     | Enterprise    | Complex setup | Yes           |
| Load Balancing     | Built-in     | Yes           | Yes           | Yes           |
| Binary Size        | Under 20MB   | Over 100MB    | Around 20MB   | Around 70MB   |
| Memory (idle)      | 12MB         | 512MB         | 32MB          | 40MB          |
| Dependencies       | None         | PostgreSQL    | None          | None          |
| Edge/IoT Ready     | Yes          | No            | Yes           | Yes           |
| CLI Tool           | Full         | Limited       | No            | Limited       |

GONK is optimized for environments where you need authorization + mTLS + load balancing in a single lightweight package, particularly edge and IoT deployments.

## Building

```bash
# Both server and CLI
make build

# Just the server
make build-server

# Just the CLI
make build-cli

# All platforms (for releases)
make build-all

# Clean
make clean

# Run tests
make test
```

On Windows without make, use Go directly:

```cmd
go build -o bin\gonk.exe .\cmd\gonk
go build -o bin\gonk-cli.exe .\cmd\gonk-cli
```

## Project Structure

```
gonk/
├── cmd/
│   ├── gonk/          # Server binary
│   └── gonk-cli/      # CLI tool
├── internal/
│   ├── auth/          # Authorization (RBAC, scopes, mTLS)
│   ├── loadbalancer/  # Load balancing strategies
│   ├── config/        # Configuration loading
│   ├── server/        # HTTP server
│   ├── proxy/         # Proxy handlers (HTTP/WS/gRPC)
│   ├── cache/         # Response caching
│   ├── metrics/       # Prometheus metrics
│   └── middleware/    # Rate limiting, logging, etc
├── gonk.example.yaml  # Example configuration
└── Makefile
```

## License

Apache License 2.0

## Acknowledgments

This version was driven by feedback from industrial IoT users who needed lightweight authorization capabilities. Thanks to the Go ecosystem libraries that make this possible: Gorilla Mux, JWT-Go, Prometheus client, and others.
