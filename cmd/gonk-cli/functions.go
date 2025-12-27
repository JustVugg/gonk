package main

import (
    "crypto/rand"
    "crypto/rsa"
    "crypto/x509"
    "crypto/x509/pkix"
    "encoding/base64"
    "encoding/json"
    "encoding/pem"
    "fmt"
    "io/ioutil"
    "math/big"
    "net/http"
    "os"
    "os/exec"
    "strings"
    "time"
    
    "github.com/golang-jwt/jwt/v5"
    "gopkg.in/yaml.v3"
    
    "github.com/JustVugg/gonk/internal/config"
)

const (
    defaultGonkURL = "http://localhost:8080"
)

// Server management functions
func startServer(configPath string) {
    cmd := exec.Command("gonk", "-config", configPath)
    cmd.Stdout = os.Stdout
    cmd.Stderr = os.Stderr
    
    if err := cmd.Run(); err != nil {
        fmt.Printf("Failed to start server: %v\n", err)
        os.Exit(1)
    }
}

func checkStatus() {
    resp, err := http.Get(defaultGonkURL + "/_gonk/health")
    if err != nil {
        fmt.Println("❌ GONK server is not running")
        return
    }
    defer resp.Body.Close()
    
    if resp.StatusCode == 200 {
        var health map[string]interface{}
        json.NewDecoder(resp.Body).Decode(&health)
        fmt.Println("✅ GONK server is running")
        fmt.Printf("   Uptime: %v\n", health["uptime"])
        fmt.Printf("   Upstreams: %v\n", health["upstreams"])
    } else {
        fmt.Println("❌ GONK server is unhealthy")
    }
}

func reloadConfig() {
    // Send SIGHUP to reload
    fmt.Println("🔄 Reloading configuration...")
    cmd := exec.Command("pkill", "-HUP", "gonk")
    if err := cmd.Run(); err != nil {
        fmt.Printf("Failed to reload: %v\n", err)
        return
    }
    fmt.Println("✅ Configuration reloaded")
}

// Config management
func validateConfig(configPath string) error {
    _, err := config.Load(configPath)
    return err
}

func initializeConfig(template, output string) {
    var configContent string
    
    switch template {
    case "industrial":
        configContent = industrialTemplate
    case "microservices":
        configContent = microservicesTemplate
    default:
        configContent = basicTemplate
    }
    
    if err := ioutil.WriteFile(output, []byte(configContent), 0644); err != nil {
        fmt.Printf("Failed to write config: %v\n", err)
        return
    }
    
    fmt.Printf("✅ Created %s configuration: %s\n", template, output)
    fmt.Println("\nNext steps:")
    fmt.Println("1. Edit the configuration file")
    fmt.Println("2. Run: gonk validate -c", output)
    fmt.Println("3. Run: gonk start -c", output)
}

func showConfig(configPath string) {
    cfg, err := config.Load(configPath)
    if err != nil {
        fmt.Printf("Failed to load config: %v\n", err)
        return
    }
    
    data, _ := yaml.Marshal(cfg)
    fmt.Println(string(data))
}

// Route management
func listRoutes() {
    resp, err := http.Get(defaultGonkURL + "/_gonk/info")
    if err != nil {
        fmt.Println("Failed to fetch routes:", err)
        return
    }
    defer resp.Body.Close()
    
    var info map[string]interface{}
    json.NewDecoder(resp.Body).Decode(&info)
    
    fmt.Printf("Total routes: %v\n\n", info["routes"])
    
    // In a real implementation, we'd fetch actual route list
    fmt.Println("Use 'gonk config show' to see all routes")
}

func addRoute() {
    fmt.Println("Interactive route creation not yet implemented")
    fmt.Println("Please edit your configuration file manually")
}

func describeRoute(routeName string) {
    fmt.Printf("Describing route: %s\n", routeName)
    fmt.Println("Route details not yet implemented")
}

// JWT management
func generateJWT(role string, scopes []string, userID string, expiryStr string) {
    // Parse expiry duration
    expiry, err := time.ParseDuration(expiryStr)
    if err != nil {
        fmt.Printf("Invalid expiry duration: %v\n", err)
        return
    }
    
    // Create claims
    now := time.Now()
    claims := jwt.MapClaims{
        "iss":    "gonk-cli",
        "sub":    userID,
        "iat":    now.Unix(),
        "exp":    now.Add(expiry).Unix(),
        "roles":  []string{role},
        "scopes": scopes,
        "user_id": userID,
    }
    
    // Create token
    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    
    // Sign token (use secret from config or env)
    secret := os.Getenv("JWT_SECRET")
    if secret == "" {
        secret = "change-me-in-production"
        fmt.Println("⚠️  Warning: Using default secret. Set JWT_SECRET environment variable.")
    }
    
    tokenString, err := token.SignedString([]byte(secret))
    if err != nil {
        fmt.Printf("Failed to generate token: %v\n", err)
        return
    }
    
    fmt.Println("✅ JWT Token generated:")
    fmt.Println()
    fmt.Println(tokenString)
    fmt.Println()
    fmt.Println("Token details:")
    fmt.Printf("  User ID: %s\n", userID)
    fmt.Printf("  Role: %s\n", role)
    fmt.Printf("  Scopes: %v\n", scopes)
    fmt.Printf("  Expires: %s\n", now.Add(expiry).Format(time.RFC3339))
    fmt.Println()
    fmt.Println("Usage:")
    fmt.Printf("  curl -H 'Authorization: Bearer %s' http://localhost:8080/api/endpoint\n", tokenString)
}

func validateJWT(tokenString string) {
    secret := os.Getenv("JWT_SECRET")
    if secret == "" {
        secret = "change-me-in-production"
    }
    
    token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
        if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
            return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
        }
        return []byte(secret), nil
    })
    
    if err != nil {
        fmt.Printf("❌ Token invalid: %v\n", err)
        return
    }
    
    if token.Valid {
        fmt.Println("✅ Token is valid")
        if claims, ok := token.Claims.(jwt.MapClaims); ok {
            fmt.Println("\nClaims:")
            for k, v := range claims {
                fmt.Printf("  %s: %v\n", k, v)
            }
        }
    } else {
        fmt.Println("❌ Token is invalid")
    }
}

func decodeJWT(tokenString string) {
    parts := strings.Split(tokenString, ".")
    if len(parts) != 3 {
        fmt.Println("Invalid JWT format")
        return
    }
    
    // Decode header
    headerBytes, _ := base64.RawURLEncoding.DecodeString(parts[0])
    var header map[string]interface{}
    json.Unmarshal(headerBytes, &header)
    
    // Decode payload
    payloadBytes, _ := base64.RawURLEncoding.DecodeString(parts[1])
    var payload map[string]interface{}
    json.Unmarshal(payloadBytes, &payload)
    
    fmt.Println("JWT Token Decoded:")
    fmt.Println("\nHeader:")
    printJSON(header)
    fmt.Println("\nPayload:")
    printJSON(payload)
    fmt.Println("\n⚠️  Note: This only decodes the token, it does NOT validate the signature")
}

// API Key management
func generateAPIKey(clientID string, roles, scopes []string) {
    // Generate random API key
    key := make([]byte, 32)
    rand.Read(key)
    apiKey := base64.URLEncoding.EncodeToString(key)
    
    fmt.Println("✅ API Key generated:")
    fmt.Println()
    fmt.Println(apiKey)
    fmt.Println()
    fmt.Println("Add to your gonk.yaml:")
    fmt.Println()
    fmt.Println("auth:")
    fmt.Println("  api_key:")
    fmt.Println("    enabled: true")
    fmt.Println("    header: X-API-Key")
    fmt.Println("    keys:")
    fmt.Printf("      - key: %s\n", apiKey)
    fmt.Printf("        client_id: %s\n", clientID)
    if len(roles) > 0 {
        fmt.Printf("        roles: %v\n", roles)
    }
    if len(scopes) > 0 {
        fmt.Printf("        scopes: %v\n", scopes)
    }
    fmt.Println()
    fmt.Println("Usage:")
    fmt.Printf("  curl -H 'X-API-Key: %s' http://localhost:8080/api/endpoint\n", apiKey)
}

func listAPIKeys() {
    fmt.Println("API Keys (from configuration):")
    fmt.Println("Not yet implemented - please check your gonk.yaml")
}

// Certificate management
func generateCertificate(cn, certType, output string) {
    fmt.Printf("Generating %s certificate for CN=%s...\n", certType, cn)
    
    // Generate private key
    privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
    if err != nil {
        fmt.Printf("Failed to generate private key: %v\n", err)
        return
    }
    
    // Create certificate template
    serialNumber, _ := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
    
    template := x509.Certificate{
        SerialNumber: serialNumber,
        Subject: pkix.Name{
            CommonName:   cn,
            Organization: []string{"GONK"},
        },
        NotBefore:             time.Now(),
        NotAfter:              time.Now().Add(365 * 24 * time.Hour),
        KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
        ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
        BasicConstraintsValid: true,
    }
    
    if certType == "ca" {
        template.IsCA = true
        template.KeyUsage |= x509.KeyUsageCertSign
    }
    
    // Create certificate
    certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
    if err != nil {
        fmt.Printf("Failed to create certificate: %v\n", err)
        return
    }
    
    // Write certificate
    certFile := fmt.Sprintf("%s/%s.crt", output, certType)
    certOut, _ := os.Create(certFile)
    pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: certDER})
    certOut.Close()
    
    // Write private key
    keyFile := fmt.Sprintf("%s/%s.key", output, certType)
    keyOut, _ := os.Create(keyFile)
    pem.Encode(keyOut, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(privateKey)})
    keyOut.Close()
    
    fmt.Printf("✅ Certificate generated:\n")
    fmt.Printf("   Certificate: %s\n", certFile)
    fmt.Printf("   Private Key: %s\n", keyFile)
}

func validateCertificate(certFile, caFile string) {
    fmt.Printf("Validating certificate: %s\n", certFile)
    
    certPEM, err := ioutil.ReadFile(certFile)
    if err != nil {
        fmt.Printf("Failed to read certificate: %v\n", err)
        return
    }
    
    block, _ := pem.Decode(certPEM)
    cert, err := x509.ParseCertificate(block.Bytes)
    if err != nil {
        fmt.Printf("Failed to parse certificate: %v\n", err)
        return
    }
    
    now := time.Now()
    if now.Before(cert.NotBefore) {
        fmt.Println("❌ Certificate not yet valid")
        return
    }
    if now.After(cert.NotAfter) {
        fmt.Println("❌ Certificate expired")
        return
    }
    
    fmt.Println("✅ Certificate is valid")
    fmt.Printf("   Subject: %s\n", cert.Subject.CommonName)
    fmt.Printf("   Valid from: %s\n", cert.NotBefore.Format(time.RFC3339))
    fmt.Printf("   Valid until: %s\n", cert.NotAfter.Format(time.RFC3339))
}

func showCertInfo(certFile string) {
    certPEM, err := ioutil.ReadFile(certFile)
    if err != nil {
        fmt.Printf("Failed to read certificate: %v\n", err)
        return
    }
    
    block, _ := pem.Decode(certPEM)
    cert, err := x509.ParseCertificate(block.Bytes)
    if err != nil {
        fmt.Printf("Failed to parse certificate: %v\n", err)
        return
    }
    
    fmt.Println("Certificate Information:")
    fmt.Printf("  Subject: %s\n", cert.Subject.CommonName)
    fmt.Printf("  Issuer: %s\n", cert.Issuer.CommonName)
    fmt.Printf("  Serial Number: %s\n", cert.SerialNumber)
    fmt.Printf("  Valid From: %s\n", cert.NotBefore.Format(time.RFC3339))
    fmt.Printf("  Valid Until: %s\n", cert.NotAfter.Format(time.RFC3339))
    fmt.Printf("  Is CA: %v\n", cert.IsCA)
}

// Monitoring functions
func showMetrics(route string) {
    url := defaultGonkURL + "/metrics"
    resp, err := http.Get(url)
    if err != nil {
        fmt.Printf("Failed to fetch metrics: %v\n", err)
        return
    }
    defer resp.Body.Close()
    
    body, _ := ioutil.ReadAll(resp.Body)
    
    if route != "" {
        // Filter metrics for specific route
        lines := strings.Split(string(body), "\n")
        fmt.Printf("Metrics for route: %s\n\n", route)
        for _, line := range lines {
            if strings.Contains(line, route) {
                fmt.Println(line)
            }
        }
    } else {
        fmt.Println(string(body))
    }
}

func tailLogs(route string, follow bool) {
    fmt.Println("Log tailing not yet implemented")
    fmt.Println("Use: journalctl -u gonk -f")
}

func checkHealth() {
    resp, err := http.Get(defaultGonkURL + "/_gonk/health")
    if err != nil {
        fmt.Printf("Health check failed: %v\n", err)
        return
    }
    defer resp.Body.Close()
    
    var health map[string]interface{}
    json.NewDecoder(resp.Body).Decode(&health)
    
    fmt.Println("Health Status:")
    printJSON(health)
}

func showCacheStats() {
    resp, err := http.Get(defaultGonkURL + "/_gonk/cache/stats")
    if err != nil {
        fmt.Printf("Failed to fetch cache stats: %v\n", err)
        return
    }
    defer resp.Body.Close()
    
    var stats map[string]interface{}
    json.NewDecoder(resp.Body).Decode(&stats)
    
    fmt.Println("Cache Statistics:")
    printJSON(stats)
}

func clearCache() {
    resp, err := http.Post(defaultGonkURL+"/_gonk/cache/clear", "application/json", nil)
    if err != nil {
        fmt.Printf("Failed to clear cache: %v\n", err)
        return
    }
    defer resp.Body.Close()
    
    fmt.Println("✅ Cache cleared")
}

// Utility functions
func printJSON(data interface{}) {
    output, _ := json.MarshalIndent(data, "", "  ")
    fmt.Println(string(output))
}
