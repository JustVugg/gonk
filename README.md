<div align="center">
<img src="gonk.png" alt="GONK Logo" width="400">
<br><br>
  
[![Go Version](https://img.shields.io/badge/go-1.21+-blue.svg)](https://go.dev/)
[![License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](LICENSE)

[Features](#-features) • [Quick Start](#-quick-start) • [Documentation](#-documentation) • [Performance](#-performance) • [Contributing](#-contributing)

</div>

---

**GONK** is an ultra-lightweight, edge-native API Gateway designed for enterprise, industrial, IoT, and privacy-first environments. Built in Go, GONK provides enterprise-grade features in a single binary under 20MB, with no external dependencies.

## 🎯 Why GONK?

Traditional API gateways are bloated, cloud-dependent, and complex. GONK is different:

- **🏭 Industrial-Grade**: Designed for air-gapped networks, factory floors, and edge deployments
- **🔒 Privacy-First**: No telemetry, no phone-home, no cloud requirements
- **⚡ Blazing Fast**: Sub-millisecond overhead, handles 10K+ req/s on a Raspberry Pi
- **🎯 Simple**: YAML config, single binary, no database required
- **🔧 Flexible**: HTTP/WebSocket/gRPC support with hot-reload configuration

## ✨ Features

### Core Gateway Features
- **🚀 Protocol Support**
  - HTTP/1.1, HTTP/2, HTTPS with TLS termination
  - WebSocket (ws/wss) with full bidirectional proxying
  - gRPC proxy with HTTP/2 support
  - Automatic protocol detection

- **🛣️ Advanced Routing**
  - Path-based routing with wildcards (`/api/*`)
  - Method-based filtering (GET, POST, etc.)
  - Header-based routing rules
  - Host-based virtual hosting
  - Route priorities and weights

- **🔐 Security & Authentication**
  - JWT validation with RS256/HS256 support
  - API Key authentication with client identification
  - Rate limiting per IP/client/route
  - CORS configuration per route
  - Request size limiting

- **💪 Resilience & Performance**
  - Circuit breakers with configurable thresholds
  - Automatic retries with exponential backoff
  - Response caching with TTL control
  - Connection pooling and reuse
  - Graceful shutdown handling

- **📊 Observability**
  - Prometheus metrics endpoint
  - Structured JSON/text logging
  - Request tracing with correlation IDs
  - Health check endpoints
  - Real-time statistics

- **🔄 Transformation**
  - Request/Response header manipulation
  - Path rewriting and stripping
  - Query parameter injection
  - Custom header injection with variables

## 🚀 Quick Start

### 1. Download and Run

```bash
# Download the latest release
wget https://github.com/JustVugg/gonk/releases/latest/download/gonk-linux-amd64
chmod +x gonk-linux-amd64

# Create a basic configuration
cat > gonk.yaml << 'EOF'
server:
  listen: ":8080"

routes:
  - name: "httpbin"
    path: "/*"
    upstream: "https://httpbin.org"
EOF

# Run GONK
./gonk-linux-amd64 -config gonk.yaml
```

### 2. Test Your Gateway

```yaml
# gonk.yaml
server:
  listen: ":8080"
  read_timeout: 30s
  write_timeout: 30s

logging:
  level: info
  format: json

routes:
  - name: "api"
    path: "/api/*"
    upstream: "http://backend:3000"
    strip_path: true
```

### Advanced Example with All Features

```yaml
# Advanced configuration example
server:
  listen: ":8080"
  http2: true  # Enable HTTP/2
  hot_reload: true  # Auto-reload on config changes
  
  # CORS configuration
  cors:
    enabled: true
    allowed_origins: ["https://app.example.com"]
    allowed_methods: ["GET", "POST", "PUT", "DELETE"]
    allowed_headers: ["Authorization", "Content-Type"]
    max_age: 3600

# Authentication configuration
auth:
  jwt:
    enabled: true
    secret_key: "${JWT_SECRET}"  # From environment
    header: "Authorization"
    prefix: "Bearer"
    expiry_check: true
  
  api_key:
    enabled: true
    header: "X-API-Key"
    keys:
      - key: "${API_KEY_CLIENT1}"
        client_id: "mobile-app"
      - key: "${API_KEY_CLIENT2}"
        client_id: "web-dashboard"

# Global rate limiting
rate_limit:
  enabled: true
  requests_per_second: 1000
  burst: 2000
  by: "ip"  # or "client_id"

# Prometheus metrics
metrics:
  enabled: true
  path: "/metrics"

# Route definitions
routes:
  # REST API with JWT auth and caching
  - name: "api-v1"
    path: "/api/v1/*"
    methods: ["GET", "POST", "PUT", "DELETE"]
    upstream: "http://api-service:3000"
    strip_path: false
    
    auth:
      type: "jwt"
      required: true
    
    rate_limit:
      enabled: true
      requests_per_second: 100
      burst: 200
    
    circuit_breaker:
      enabled: true
      max_failures: 5
      reset_timeout: 60s
    
    cache:
      enabled: true
      ttl: 300s
      methods: ["GET"]
    
    transform:
      request:
        add_headers:
          X-Gateway: "gonk"
          X-Request-ID: "${request_id}"
          X-Real-IP: "${remote_addr}"
        remove_headers: ["Cookie"]
      response:
        add_headers:
          X-Cache-Status: "${cache_status}"
        remove_headers: ["Server"]
    
    timeout:
      connect: 5s
      read: 30s
      write: 30s

  # WebSocket endpoint with API key auth
  - name: "websocket"
    path: "/ws/*"
    methods: ["GET"]
    upstream: "ws://websocket-service:8080"
    protocol: "ws"
    strip_path: true
    
    auth:
      type: "api_key"
      required: true

  # gRPC service proxy
  - name: "grpc-users"
    path: "/grpc/users/*"
    methods: ["POST"]
    upstream: "grpc-service:50051"
    protocol: "grpc"
    
    auth:
      type: "jwt"
      required: true
    
    timeout:
      read: 60s
      write: 60s

  # Public health endpoint (no auth)
  - name: "health"
    path: "/health"
    methods: ["GET"]
    upstream: "http://api-service:3000/health"
    
    cache:
      enabled: false  # Don't cache health checks

```

### Environment Variables

### GONK supports environment variable substitution in configuration:

```yaml
auth:
  jwt:
    secret_key: "${JWT_SECRET:-default-secret}"  # With default value
    
database:
  url: "${DATABASE_URL}"  # Required env var
```

## 🐳 Docker Deployment

### Basic Docker Run

```bash
# Pull the image
docker pull ghcr.io/JustVugg/gonk:latest

# Run with custom config
docker run -d \
  --name gonk \
  -p 8080:8080 \
  -v $(pwd)/gonk.yaml:/etc/gonk/gonk.yaml \
  -e JWT_SECRET=your-secret-key \
  ghcr.io/JustVugg/gonk:latest

```

### Docker Compose

```yaml
# docker-compose.yml
version: '3.8'

services:
  gonk:
    image: ghcr.io/JustVugg/gonk:latest
    ports:
      - "8080:8080"
    volumes:
      - ./gonk.yaml:/etc/gonk/gonk.yaml:ro
    environment:
      - JWT_SECRET=${JWT_SECRET}
      - API_KEY_1=${API_KEY_1}
    restart: unless-stopped
    networks:
      - gateway

  # Example backend service
  api:
    image: your-api:latest
    networks:
      - gateway

networks:
  gateway:
    driver: bridge

```

### ☸️ Kubernetes Deployment

### Basic Deployment

```yaml
# gonk-deployment.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: gonk-config
data:
  gonk.yaml: |
    server:
      listen: ":8080"
    routes:
      - name: "api"
        path: "/api/*"
        upstream: "http://api-service:3000"
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: gonk
spec:
  replicas: 3
  selector:
    matchLabels:
      app: gonk
  template:
    metadata:
      labels:
        app: gonk
    spec:
      containers:
      - name: gonk
        image: ghcr.io/JustVugg/gonk:latest
        ports:
        - containerPort: 8080
        volumeMounts:
        - name: config
          mountPath: /etc/gonk
        env:
        - name: JWT_SECRET
          valueFrom:
            secretKeyRef:
              name: gonk-secrets
              key: jwt-secret
        livenessProbe:
          httpGet:
            path: /_gonk/live
            port: 8080
          initialDelaySeconds: 10
        readinessProbe:
          httpGet:
            path: /_gonk/ready
            port: 8080
          initialDelaySeconds: 5
        resources:
          requests:
            memory: "64Mi"
            cpu: "100m"
          limits:
            memory: "256Mi"
            cpu: "500m"
      volumes:
      - name: config
        configMap:
          name: gonk-config
---
apiVersion: v1
kind: Service
metadata:
  name: gonk
spec:
  selector:
    app: gonk
  ports:
  - port: 80
    targetPort: 8080
  type: LoadBalancer

```

### Helm Chart

```bash
# Install GONK using Helm
helm repo add gonk https://charts.gonk.io
helm install my-gateway gonk/gonk \
  --set config.jwt.secret=$JWT_SECRET \
  --set ingress.enabled=true \
  --set ingress.host=api.example.com
```

### 📊 Performance Benchmarks
### Test Environment
### Hardware: Raspberry Pi 4 (4GB RAM), Intel i7-9700K (32GB RAM)
### Network: 1 Gbps local network
### Backend: Simple HTTP echo server
### Tool: Apache Bench, wrk, k6

### Results on Raspberry Pi 4

```bash
# Simple HTTP routing (no auth)
$ wrk -t4 -c100 -d30s http://localhost:8080/api/echo
Running 30s test @ http://localhost:8080/api/echo
  4 threads and 100 connections
  Thread Stats   Avg      Stdev     Max   +/- Stdev
    Latency     9.82ms    4.23ms  89.91ms   71.23%
    Req/Sec     2.59k   312.45     3.21k    70.83%
  309,875 requests in 30.01s, 41.23MB read
Requests/sec:  10,325.42
Transfer/sec:      1.37MB

# With JWT validation
$ wrk -t4 -c100 -d30s -H "Authorization: Bearer $TOKEN" http://localhost:8080/api/echo
Running 30s test @ http://localhost:8080/api/echo
  4 threads and 100 connections
  Thread Stats   Avg      Stdev     Max   +/- Stdev
    Latency    18.45ms    7.82ms  124.3ms   68.92%
    Req/Sec     1.38k   189.23     1.82k    71.25%
  165,234 requests in 30.02s, 21.98MB read
Requests/sec:   5,504.89
Transfer/sec:    750.12KB

# WebSocket connections
$ wscat-benchmark -c 1000 -d 60 ws://localhost:8080/ws
Connected: 1000 clients
Messages sent: 2,456,123
Messages received: 2,456,123
Average latency: 2.3ms
Memory usage: 47MB
CPU usage: 35%
```

### Results on Intel i7-9700K

```bash
# HTTP routing with caching
$ wrk -t8 -c500 -d30s http://localhost:8080/api/cached
Running 30s test @ http://localhost:8080/api/cached
  8 threads and 500 connections
  Thread Stats   Avg      Stdev     Max   +/- Stdev
    Latency     4.32ms    2.91ms  45.23ms   82.15%
    Req/Sec    14.52k     1.23k   18.91k    73.33%
  3,478,234 requests in 30.03s, 462.81MB read
Requests/sec: 115,839.23
Transfer/sec:     15.41MB

# gRPC proxying
$ ghz --proto ./user.proto --call UserService.GetUser \
    --insecure --duration 30s --connections 50 \
    --rps 10000 localhost:8080
Summary:
  Count:        298,453
  Total:        30.00s
  Slowest:      45.23ms
  Fastest:      0.89ms
  Average:      4.98ms
  Requests/sec: 9,948.43
```

### Memory Usage

| Scenario        | Memory Usage | Notes                        |
|----------------|--------------|------------------------------|
| Idle           | 12MB         | Base memory with config loaded |
| 100 req/s      | 18MB         | Light load                   |
| 1K req/s       | 35MB         | Moderate load                |
| 10K req/s      | 48MB         | Heavy load                   |
| 1K WebSocket   | 67MB         | 1000 concurrent connections  |


### 🏗️ Architecture

```text

┌─────────────────┐
│   Client Apps   │
└────────┬────────┘
         │
    ┌────▼────┐
    │  GONK   │
    │ Gateway │
    └────┬────┘
         │
   ┌─────┴─────┬──────────┬──────────┐
   │           │          │          │
┌──▼──┐  ┌────▼───┐  ┌───▼───┐  ┌───▼───┐
│HTTP │  │WebSocket│  │ gRPC  │  │Static │
│ API │  │Service  │  │Service│  │Files  │
└─────┘  └─────────┘  └───────┘  └───────┘

Components:
├── Router (Gorilla Mux)
├── Auth Middleware (JWT/API Key)
├── Rate Limiter (Token Bucket)
├── Circuit Breaker (State Machine)
├── Cache Layer (In-Memory LRU)
├── Proxy Handlers (HTTP/WS/gRPC)
└── Metrics Collector (Prometheus)
```

# 🔧 Advanced Usage

## Custom Headers with Variables (YAML)

```yaml
transform:
  request:
    add_headers:
      X-Request-ID: "${request_id}"
      X-Real-IP: "${remote_addr}"
      X-Forwarded-Host: "${host}"
      X-Timestamp: "${timestamp}"
      X-Method: "${method}"
```

## 📈 Rate Limiting Strategies (YAML)

```yaml
# Global rate limit
rate_limit:
  enabled: true
  requests_per_second: 1000
  by: "ip"

# Per-route with different strategies
routes:
  - name: "public-api"
    rate_limit:
      requests_per_second: 10
      by: "ip"
      
  - name: "premium-api"
    rate_limit:
      requests_per_second: 1000
      by: "client_id"  # Requires auth
```

## ⚡ Circuit Breaker Patterns (YAML)

```yaml
circuit_breaker:
  enabled: true
  max_failures: 5        # Open after 5 failures
  reset_timeout: 60s     # Try again after 60s
  half_open_max_reqs: 3  # Test with 3 requests

  # Advanced patterns (coming soon)
  failure_rate_threshold: 0.5  # 50% failure rate
  slow_call_duration: 5s       # Consider slow if > 5s
  slow_call_rate_threshold: 0.5
```

## 🗃️ Multi-Stage Caching (YAML)

```yaml
cache:
  enabled: true
  ttl: 300s
  methods: ["GET", "HEAD"]

  # Cache key customization
  key_headers: ["Accept", "Accept-Language"]

  # Conditional caching
  cache_control_respect: true
  vary_headers: ["Accept-Encoding"]
```

## 🚀 Building from Source

### Prerequisites

- Go 1.21 or higher  
- Make (optional but recommended)  
- Docker (for container builds)

### Build Steps

```bash
# Clone the repository
git clone https://github.com/JustVugg/gonk
cd gonk

# Download dependencies
go mod download

# Build for current platform
make build

# Build for all platforms
make build-all

# Run tests
make test

# Run with hot reload during development
air -c .air.toml
```

### Cross-Compilation

```bash
# Linux AMD64
GOOS=linux GOARCH=amd64 make build

# macOS AMD64
GOOS=darwin GOARCH=amd64 make build

# Windows AMD64
GOOS=windows GOARCH=amd64 make build

# Raspberry Pi (ARM)
GOOS=linux GOARCH=arm GOARM=7 make build

# Docker multi-arch
docker buildx build --platform linux/amd64,linux/arm64,linux/arm/v7 -t gonk:latest .
```

## 🧪 Testing

### Unit Tests

```bash
# Run all tests
go test ./...

# Run with coverage
go test -cover -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Run specific tests
go test -run TestJWTAuth ./internal/auth
```

### Integration Tests

```bash
# Start test environment
docker-compose -f test/docker-compose.test.yml up -d

# Run integration tests
go test ./test/integration -tags=integration

# Load testing
k6 run test/load/scenario.js
```

### Example Test Script (JavaScript)

```js
// test/load/scenario.js
import http from 'k6/http';
import { check } from 'k6';

export let options = {
  stages: [
    { duration: '30s', target: 100 },
    { duration: '1m', target: 1000 },
    { duration: '30s', target: 0 },
  ],
};

export default function() {
  let res = http.get('http://localhost:8080/api/test', {
    headers: { 'Authorization': 'Bearer test-token' }
  });
  
  check(res, {
    'status is 200': (r) => r.status === 200,
    'latency < 50ms': (r) => r.timings.duration < 50,
  });
}
```

## 🐛 Troubleshooting

### Common Issues

#### 1. Port Already in Use

```bash
Error: listen tcp :8080: bind: address already in use

# Solution: Change port or kill process
lsof -i :8080
kill -9 <PID>
```

### 2. Config Not Loading

```bash
Failed to load config: yaml: line 10: found tab character

# Solution: Use spaces, not tabs in YAML
# Validate YAML:
yamllint gonk.yaml
```

### 3. JWT Validation Failing

```bash
{"error":"unauthorized"}

# Debug JWT:
jwt decode $TOKEN
# Check secret key matches
# Check expiry time
```

### 4. High Memory Usage

```yaml
# Tune cache settings
cache:
  max_size: 52428800  # 50MB limit
  max_items: 1000     # Max entries

# Adjust connection pools
upstream:
  max_idle_conns: 10
  max_conns_per_host: 100
```

## 🚀 Migrating from Other Gateways

### From nginx
```nginx
# nginx.conf
location /api/ {
    proxy_pass http://backend:3000/;
}
```
### To gonk

# gonk.yaml equivalent
```yaml
routes:
  - name: "api"
    path: "/api/*"
    upstream: "http://backend:3000"
    strip_path: true
```


### Debug Mode

```bash
# Run with debug logging
GONK_LOG_LEVEL=debug ./gonk -config gonk.yaml

# Enable pprof for profiling
GONK_PPROF=true ./gonk -config gonk.yaml
# Then visit http://localhost:6060/debug/pprof/
```

## 🆚 Comparison with Other Gateways

| Feature         | GONK         | Kong              | Nginx         | Envoy         | Traefik       |
|-----------------|--------------|-------------------|---------------|---------------|---------------|
| Binary Size     | < 20MB       | > 100MB           | > 20MB        | > 50MB        | > 70MB        |
| Memory (Idle)   | 12MB         | 512MB             | 32MB          | 128MB         | 40MB          |
| Dependencies    | None         | PostgreSQL/Cassandra | Modules     | None          | None          |
| Configuration   | YAML         | Database + API    | Config files  | YAML          | YAML/Labels   |
| Hot Reload      | ✅           | ✅                | ❌            | ✅            | ✅            |
| WebSocket       | ✅ Native    | ✅ Plugin         | ⚠️ Basic      | ✅            | ✅            |
| gRPC            | ✅ Native    | ✅ Plugin         | ⚠️ Module     | ✅ Native     | ✅            |
| JWT Auth        | ✅ Built-in  | ✅ Plugin         | ❌            | ⚠️ EnvoyFilter | ✅            |
| Rate Limiting   | ✅ Built-in  | ✅ Plugin         | ⚠️ Module     | ✅            | ✅            |
| Caching         | ✅ Built-in  | ✅ Plugin         | ⚠️ Module     | ⚠️ Filter     | ❌            |
| Circuit Breaker | ✅ Built-in  | ✅ Plugin         | ❌            | ✅            | ✅            |
| Prometheus      | ✅ Native    | ✅ Plugin         | ⚠️ Module     | ✅            | ✅            |
| Edge Ready      | ✅           | ❌                | ✅            | ⚠️            | ✅            |
| License        | Apache 2.0    | Apache 2.0*       | BSD           | Apache 2.0    | MIT           |

*Note: Kong has enterprise features that are proprietary

## 🚀 Performance Comparison (req/s on same hardware)

**Test: Simple HTTP proxy, 100 concurrent connections**

| Gateway  | Requests per second | Memory Usage     |
|----------|---------------------|------------------|
| GONK     | 45,832 req/s        | 48MB RAM         |
| Nginx    | 51,245 req/s        | 64MB RAM         |
| Envoy    | 38,421 req/s        | 156MB RAM        |
| Kong     | 12,832 req/s        | 743MB RAM        |
| Traefik  | 22,109 req/s        | 89MB RAM         |

**Test: JWT validation + rate limiting**

| Gateway  | Requests per second | Memory Usage     |
|----------|---------------------|------------------|
| GONK     | 18,234 req/s        | 52MB RAM         |
| Nginx    | N/A (requires Nginx Plus) |                |
| Envoy    | 15,234 req/s        | 189MB RAM        |
| Kong     | 8,234 req/s         | 812MB RAM        |
| Traefik  | 11,432 req/s        | 95MB RAM         |

## 🌟 Use Cases

### 1. IoT Gateway on Edge

```yaml
# Lightweight config for Raspberry Pi
server:
  listen: ":8080"

routes:
  - name: "mqtt-bridge"
    path: "/mqtt/*"
    upstream: "http://mosquitto:1883"

  - name: "sensor-data"
    path: "/api/sensors/*"
    upstream: "http://influxdb:8086"
    auth:
      type: "api_key"
    rate_limit:
      requests_per_second: 10
```

### 2. Microservices Gateway

```yaml
routes:
  - name: "user-service"
    path: "/api/users/*"
    upstream: "http://user-service:3000"

  - name: "order-service"
    path: "/api/orders/*"
    upstream: "http://order-service:3001"

  - name: "inventory-service"
    path: "/api/inventory/*"
    upstream: "http://inventory-service:3002"
```

### 3. API Monetization

```yaml
auth:
  api_key:
    keys:
      - key: "free-tier-key"
        client_id: "free-user"
      - key: "premium-key"
        client_id: "premium-user"

routes:
  - name: "api"
    path: "/api/*"
    upstream: "http://api:3000"
    rate_limit:
      enabled: true
      by: "client_id"
      # Custom limits per client
      custom:
        - client_id: "free-user"
          requests_per_second: 10
        - client_id: "premium-user"
          requests_per_second: 1000
```

## 🤝 Contributing

We love contributions! Please see CONTRIBUTING.md for details.

### Development Setup

```bash
# Fork and clone
git clone https://github.com/JustVugg/gonk
cd gonk

# Create branch
git checkout -b feature/amazing-feature

# Make changes and test
make test

# Commit with conventional commits
git commit -m "feat: add amazing feature"

# Push and create PR
git push origin feature/amazing-feature
```

## 📜 License

GONK Community Edition is licensed under the Apache License 2.0. See LICENSE for details.

## 🙏 Acknowledgments

GONK is built on the shoulders of giants:

- Gorilla Mux - HTTP router  
- JWT-Go - JWT library  
- Prometheus Client - Metrics  
- FSNotify - File watching  
- Gorilla WebSocket - WebSocket support

<div align="center">
Built with ❤️ for the Edge

Simple • Fast • Reliable

</div>







