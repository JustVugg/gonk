package auth

import (
    "context"
    "fmt"
    "net/http"
    "strings"
    "time"
    
    "github.com/golang-jwt/jwt/v5"
    "github.com/JustVugg/gonk/internal/config"
)

// CustomClaims extends JWT claims with roles and scopes
type CustomClaims struct {
    jwt.RegisteredClaims
    Roles  []string `json:"roles,omitempty"`
    Scopes []string `json:"scopes,omitempty"`
    UserID string   `json:"user_id,omitempty"`
}

// ValidateJWT validates JWT token and extracts authentication context
func ValidateJWT(r *http.Request, cfg *config.JWTConfig) (*AuthContext, error) {
    tokenString := extractToken(r, cfg)
    if tokenString == "" {
        return nil, fmt.Errorf("no token provided")
    }

    // Parse token with custom claims
    token, err := jwt.ParseWithClaims(tokenString, &CustomClaims{}, func(token *jwt.Token) (interface{}, error) {
        if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
            return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
        }
        return []byte(cfg.SecretKey), nil
    })

    if err != nil {
        return nil, fmt.Errorf("failed to parse token: %w", err)
    }

    if !token.Valid {
        return nil, fmt.Errorf("invalid token")
    }

    claims, ok := token.Claims.(*CustomClaims)
    if !ok {
        return nil, fmt.Errorf("invalid token claims")
    }

    // Validate expiry if enabled
    if cfg.ExpiryCheck && claims.ExpiresAt != nil {
        if claims.ExpiresAt.Before(time.Now()) {
            return nil, fmt.Errorf("token expired")
        }
    }

    // Validate roles if configured
    if cfg.ValidateRoles && len(claims.Roles) == 0 {
        return nil, fmt.Errorf("token missing required roles")
    }

    // Validate scopes if configured
    if cfg.ValidateScopes && len(claims.Scopes) == 0 {
        return nil, fmt.Errorf("token missing required scopes")
    }

    // Build auth context
    authCtx := &AuthContext{
        Authenticated: true,
        IdentityType:  "user",
        UserID:        claims.UserID,
        Roles:         claims.Roles,
        Scopes:        claims.Scopes,
    }

    // If subject is available, use it as UserID if UserID is not set
    if authCtx.UserID == "" && claims.Subject != "" {
        authCtx.UserID = claims.Subject
    }

    return authCtx, nil
}

func extractToken(r *http.Request, cfg *config.JWTConfig) string {
    header := r.Header.Get(cfg.Header)
    if header == "" {
        return ""
    }

    if cfg.Prefix != "" {
        parts := strings.Split(header, " ")
        if len(parts) != 2 || parts[0] != strings.TrimSpace(cfg.Prefix) {
            return ""
        }
        return parts[1]
    }

    return header
}

// StoreAuthContext stores auth context in request context
func StoreAuthContext(r *http.Request, authCtx *AuthContext) *http.Request {
    ctx := context.WithValue(r.Context(), "auth_context", authCtx)
    return r.WithContext(ctx)
}

// GetAuthContext retrieves auth context from request context
func GetAuthContext(r *http.Request) *AuthContext {
    if authCtxVal := r.Context().Value("auth_context"); authCtxVal != nil {
        if authCtx, ok := authCtxVal.(*AuthContext); ok {
            return authCtx
        }
    }
    return &AuthContext{
        Authenticated: false,
        IdentityType:  "unknown",
    }
}
