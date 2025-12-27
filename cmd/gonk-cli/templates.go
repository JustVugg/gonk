package main

const basicTemplate = `# GONK v1.1 - Basic Configuration
server:
  listen: ":8080"
  hot_reload: true

logging:
  level: info
  format: text
  output: stdout

metrics:
  enabled: true
  path: /metrics

auth:
  jwt:
    enabled: true
    secret_key: "change-me"
    header: "Authorization"
    prefix: "Bearer"
    expiry_check: true
    validate_roles: true
    validate_scopes: true

routes:
  - name: "api"
    path: "/api/*"
    methods: ["GET", "POST", "PUT", "DELETE"]
    upstreams:
      - url: "http://localhost:3000"
        weight: 100
    
    auth:
      type: "jwt"
      required: true
      allowed_roles: ["user", "admin"]
`

const industrialTemplate = `# GONK v1.1 - Industrial IoT Configuration
server:
  listen: ":8443"
  http2: true
  hot_reload: true
  
  # mTLS Configuration for device authentication
  tls:
    enabled: true
    cert_file: "/certs/server.crt"
    key_file: "/certs/server.key"
    client_ca: "/certs/ca.crt"
    client_auth: "request"  # request or require

logging:
  level: info
  format: json
  output: /var/log/gonk/gonk.log

metrics:
  enabled: true
  path: /metrics

auth:
  jwt:
    enabled: true
    secret_key: "change-me-in-production"
    header: "Authorization"
    prefix: "Bearer"
    expiry_check: true
    validate_roles: true
    validate_scopes: true
  
  api_key:
    enabled: true
    header: "X-API-Key"
    keys:
      - key: "your-device-api-key-here"
        client_id: "plc-001"
        roles: ["device"]
        scopes: ["write:sensors"]

# Global rate limiting
rate_limit:
  enabled: true
  requests_per_second: 1000
  burst: 2000
  by: "ip"

routes:
  # Sensor Data Ingestion - Devices write via mTLS or API Key
  - name: "sensor-data"
    path: "/api/sensors/*"
    methods: ["POST"]
    upstreams:
      - url: "http://timeseries-db:8086"
        weight: 100
    
    auth:
      require_either: ["client_cert", "api_key"]
      permissions:
        - identity_type: "device"
          methods: ["POST"]
          scopes: ["write:sensors"]
    
    rate_limit:
      enabled: true
      requests_per_second: 100
      by: "client_id"

  # Sensor Data Read - Users read via JWT
  - name: "sensor-read"
    path: "/api/sensors/*"
    methods: ["GET"]
    upstreams:
      - url: "http://timeseries-db:8086"
        weight: 100
    
    auth:
      type: "jwt"
      required: true
      permissions:
        - role: "technician"
          methods: ["GET"]
          scopes: ["read:sensors"]
        - role: "engineer"
          methods: ["GET"]
          scopes: ["read:sensors"]
    
    cache:
      enabled: true
      ttl: 30s
      methods: ["GET"]

  # Actuator Control - Engineers and Admins only
  - name: "actuators"
    path: "/api/actuators/*"
    methods: ["POST", "PUT"]
    upstreams:
      - url: "http://plc-gateway:502"
        weight: 100
    
    auth:
      type: "jwt"
      required: true
      allowed_roles: ["engineer", "admin"]
      required_scopes: ["write:actuators"]
    
    rate_limit:
      enabled: true
      requests_per_second: 10
      burst: 20

  # Admin Panel - Admin only with mTLS
  - name: "admin"
    path: "/admin/*"
    methods: ["GET", "POST", "PUT", "DELETE"]
    upstreams:
      - url: "http://admin-panel:3000"
        weight: 100
    
    auth:
      type: "jwt"
      required: true
      require_client_cert: true
      allowed_roles: ["admin"]
`

const microservicesTemplate = `# GONK v1.1 - Microservices Configuration
server:
  listen: ":8080"
  http2: true
  hot_reload: true
  
  cors:
    enabled: true
    allowed_origins: ["https://app.example.com"]
    allowed_methods: ["GET", "POST", "PUT", "DELETE", "OPTIONS"]
    allowed_headers: ["*"]
    max_age: 3600

logging:
  level: info
  format: json
  output: stdout

metrics:
  enabled: true
  path: /metrics

auth:
  jwt:
    enabled: true
    secret_key: "change-me-in-production"
    header: "Authorization"
    prefix: "Bearer"
    expiry_check: true
    validate_roles: true
    validate_scopes: true

rate_limit:
  enabled: true
  requests_per_second: 1000
  burst: 2000
  by: "ip"

routes:
  # User Service with Load Balancing
  - name: "user-service"
    path: "/api/users/*"
    methods: ["GET", "POST", "PUT", "DELETE"]
    upstreams:
      - url: "http://user-service-1:3000"
        weight: 70
        health_check: "/health"
      - url: "http://user-service-2:3000"
        weight: 30
        health_check: "/health"
    
    load_balancing:
      strategy: "weighted"
      health_check_interval: 10s
      health_check_timeout: 5s
    
    auth:
      type: "jwt"
      required: true
      permissions:
        - role: "user"
          methods: ["GET"]
        - role: "admin"
          methods: ["GET", "POST", "PUT", "DELETE"]
    
    circuit_breaker:
      enabled: true
      max_failures: 5
      reset_timeout: 60s
    
    cache:
      enabled: true
      ttl: 300s
      methods: ["GET"]

  # Order Service
  - name: "order-service"
    path: "/api/orders/*"
    methods: ["GET", "POST", "PUT", "DELETE"]
    upstreams:
      - url: "http://order-service:3001"
        weight: 100
        health_check: "/health"
    
    auth:
      type: "jwt"
      required: true
      required_scopes: ["read:orders", "write:orders"]
    
    circuit_breaker:
      enabled: true
      max_failures: 3
      reset_timeout: 30s

  # Payment Service (sensitive)
  - name: "payment-service"
    path: "/api/payments/*"
    methods: ["POST"]
    upstreams:
      - url: "http://payment-service:3002"
        weight: 100
    
    auth:
      type: "jwt"
      required: true
      allowed_roles: ["admin", "payment-processor"]
      required_scopes: ["write:payments"]
    
    rate_limit:
      enabled: true
      requests_per_second: 10
      burst: 20
      by: "client_id"
    
    timeout:
      connect: 5s
      read: 30s
      write: 30s

  # gRPC Service
  - name: "grpc-inventory"
    path: "/grpc/inventory/*"
    methods: ["POST"]
    upstreams:
      - url: "grpc-service:50051"
        weight: 100
    protocol: "grpc"
    
    auth:
      type: "jwt"
      required: true

  # WebSocket Notifications
  - name: "notifications"
    path: "/ws/notifications"
    methods: ["GET"]
    upstreams:
      - url: "ws://notification-service:8080"
        weight: 100
    protocol: "ws"
    
    auth:
      type: "jwt"
      required: true

  # Public Health Check (no auth)
  - name: "health"
    path: "/health"
    methods: ["GET"]
    upstreams:
      - url: "http://user-service:3000/health"
        weight: 100
    
    cache:
      enabled: false
`