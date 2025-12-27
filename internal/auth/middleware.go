package auth

import (
    "log"
    "net/http"
    
    "github.com/JustVugg/gonk/internal/config"
)

// Middleware handles authentication and authorization
func Middleware(authConfig *config.AuthConfig, routeAuth *config.RouteAuth, next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // If auth not required, skip
        if routeAuth == nil || !routeAuth.Required {
            next.ServeHTTP(w, r)
            return
        }

        var authCtx *AuthContext
        var authErr error

        // Handle dual authentication (require either)
        if len(routeAuth.RequireEither) > 0 {
            authCtx, authErr = handleDualAuth(r, authConfig, routeAuth)
        } else {
            // Single authentication type
            authCtx, authErr = handleSingleAuth(r, authConfig, routeAuth)
        }

        // Check authentication result
        if authErr != nil || authCtx == nil || !authCtx.Authenticated {
            log.Printf("Authentication failed: %v", authErr)
            respondUnauthorized(w, "authentication failed")
            return
        }

        // Store auth context in request
        r = StoreAuthContext(r, authCtx)

        // Perform authorization checks
        authorized, authzErr := ValidateAuthorization(r, routeAuth, authCtx)
        if authzErr != nil || !authorized {
            log.Printf("Authorization failed for user %s: %v", authCtx.UserID, authzErr)
            respondForbidden(w, authzErr)
            return
        }

        // Authentication and authorization successful
        next.ServeHTTP(w, r)
    })
}

// handleSingleAuth handles single authentication type
func handleSingleAuth(r *http.Request, authConfig *config.AuthConfig, routeAuth *config.RouteAuth) (*AuthContext, error) {
    switch routeAuth.Type {
    case "jwt":
        if authConfig.JWT != nil && authConfig.JWT.Enabled {
            return ValidateJWT(r, authConfig.JWT)
        }
        
    case "api_key":
        if authConfig.APIKey != nil && authConfig.APIKey.Enabled {
            return ValidateAPIKey(r, authConfig.APIKey)
        }
        
    case "mtls":
        if routeAuth.RequireClientCert {
            return ValidateMTLS(r, routeAuth)
        }
        
    default:
        // Unknown auth type, allow through
        return &AuthContext{Authenticated: true}, nil
    }

    return nil, nil
}

// handleDualAuth handles "require either" authentication
func handleDualAuth(r *http.Request, authConfig *config.AuthConfig, routeAuth *config.RouteAuth) (*AuthContext, error) {
    var lastErr error

    for _, authType := range routeAuth.RequireEither {
        var authCtx *AuthContext
        var err error

        switch authType {
        case "jwt":
            if authConfig.JWT != nil && authConfig.JWT.Enabled {
                authCtx, err = ValidateJWT(r, authConfig.JWT)
            }
            
        case "api_key":
            if authConfig.APIKey != nil && authConfig.APIKey.Enabled {
                authCtx, err = ValidateAPIKey(r, authConfig.APIKey)
            }
            
        case "client_cert", "mtls":
            if routeAuth.RequireClientCert || r.TLS != nil {
                authCtx, err = ValidateMTLS(r, routeAuth)
            }
        }

        // If authentication succeeded, return immediately
        if err == nil && authCtx != nil && authCtx.Authenticated {
            return authCtx, nil
        }

        lastErr = err
    }

    // None of the auth methods succeeded
    return nil, lastErr
}

// respondUnauthorized sends 401 Unauthorized response
func respondUnauthorized(w http.ResponseWriter, message string) {
    w.Header().Set("Content-Type", "application/json")
    w.Header().Set("WWW-Authenticate", "Bearer")
    w.WriteHeader(http.StatusUnauthorized)
    
    if message == "" {
        message = "authentication required"
    }
    
    w.Write([]byte(`{"error":"` + message + `"}`))
}

// respondForbidden sends 403 Forbidden response with detailed error
func respondForbidden(w http.ResponseWriter, err error) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusForbidden)
    
    message := "insufficient permissions"
    if err != nil {
        message = err.Error()
    }
    
    w.Write([]byte(`{"error":"` + message + `"}`))
}
