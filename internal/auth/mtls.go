package auth

import (
    "crypto/x509"
    "fmt"
    "net/http"
    "strings"
    
    "github.com/JustVugg/gonk/internal/config"
)

// ValidateMTLS validates client certificate and extracts identity
func ValidateMTLS(r *http.Request, routeAuth *config.RouteAuth) (*AuthContext, error) {
    if r.TLS == nil {
        return nil, fmt.Errorf("TLS connection required")
    }

    if len(r.TLS.PeerCertificates) == 0 {
        if routeAuth.RequireClientCert {
            return nil, fmt.Errorf("client certificate required")
        }
        return nil, nil
    }

    cert := r.TLS.PeerCertificates[0]
    
    // Extract identity from certificate
    authCtx := &AuthContext{
        Authenticated:  true,
        IdentityType:   "device", // Client certs are typically for devices/machines
        CertCommonName: cert.Subject.CommonName,
    }

    // Map certificate CN to role if configured
    if routeAuth.CertToRoleMapping != nil {
        role := mapCertToRole(cert, routeAuth.CertToRoleMapping)
        if role != "" {
            authCtx.Roles = []string{role}
        }
    }

    // Extract additional identity information from certificate
    if cert.Subject.CommonName != "" {
        authCtx.ClientID = cert.Subject.CommonName
    }

    // Extract organization as potential scope/role
    if len(cert.Subject.Organization) > 0 {
        // Could be used for additional authorization logic
        for _, org := range cert.Subject.Organization {
            if strings.HasPrefix(org, "role:") {
                role := strings.TrimPrefix(org, "role:")
                authCtx.Roles = append(authCtx.Roles, role)
            }
        }
    }

    return authCtx, nil
}

// mapCertToRole maps certificate attributes to roles using configured mapping
func mapCertToRole(cert *x509.Certificate, mapping map[string]string) string {
    cn := cert.Subject.CommonName

    // Exact match
    if role, ok := mapping[cn]; ok {
        return role
    }

    // Wildcard match (e.g., "CN=Device-*" -> "device")
    for pattern, role := range mapping {
        if strings.HasPrefix(pattern, "CN=") {
            pattern = strings.TrimPrefix(pattern, "CN=")
            if matchWildcard(cn, pattern) {
                return role
            }
        }
    }

    return ""
}

// matchWildcard performs simple wildcard matching
func matchWildcard(text, pattern string) bool {
    if !strings.Contains(pattern, "*") {
        return text == pattern
    }

    parts := strings.Split(pattern, "*")
    if len(parts) == 2 {
        // Simple prefix-suffix matching
        prefix := parts[0]
        suffix := parts[1]
        return strings.HasPrefix(text, prefix) && strings.HasSuffix(text, suffix)
    }

    return false
}

// ValidateCertChain validates the certificate chain
func ValidateCertChain(cert *x509.Certificate, caCert *x509.Certificate) error {
    // Verify certificate is signed by CA
    if err := cert.CheckSignatureFrom(caCert); err != nil {
        return fmt.Errorf("certificate not signed by CA: %w", err)
    }

    // Check if certificate is expired
    // Note: TLS library already does this, but we can add custom logic here
    
    return nil
}
