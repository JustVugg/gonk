package auth

import (
    "context"
    "fmt"
    "net/http"
    "strings"
    
    "github.com/JustVugg/gonk/internal/config"
)

// authContextKey is the key for auth context in request context
type contextKey string

const authContextKey contextKey = "auth_context"

// AuthContext holds authentication and authorization information
type AuthContext struct {
    Authenticated  bool
    IdentityType   string   // "user", "device", "service"
    UserID         string
    ClientID       string
    Roles          []string
    Scopes         []string
    CertCommonName string
}

// ValidateAuthorization performs RBAC and scope validation
func ValidateAuthorization(r *http.Request, routeAuth *config.RouteAuth, authCtx *AuthContext) (bool, error) {
    if routeAuth == nil || !routeAuth.Required {
        return true, nil
    }

    if !authCtx.Authenticated {
        return false, fmt.Errorf("not authenticated")
    }

    // Validate roles if specified
    if len(routeAuth.AllowedRoles) > 0 {
        if !hasAnyRole(authCtx.Roles, routeAuth.AllowedRoles) {
            return false, fmt.Errorf("insufficient role privileges: requires one of %v, has %v", 
                routeAuth.AllowedRoles, authCtx.Roles)
        }
    }

    // Validate scopes if specified
    if len(routeAuth.RequiredScopes) > 0 {
        if !hasAllScopes(authCtx.Scopes, routeAuth.RequiredScopes) {
            return false, fmt.Errorf("insufficient scopes: requires %v, has %v", 
                routeAuth.RequiredScopes, authCtx.Scopes)
        }
    }

    // Validate permissions matrix
    if len(routeAuth.Permissions) > 0 {
        allowed, err := checkPermissions(r.Method, authCtx, routeAuth.Permissions)
        if err != nil {
            return false, err
        }
        if !allowed {
            return false, fmt.Errorf("method %s not allowed for role %v or identity type %s", 
                r.Method, authCtx.Roles, authCtx.IdentityType)
        }
    }

    return true, nil
}

// checkPermissions validates against the permission matrix
func checkPermissions(method string, authCtx *AuthContext, permissions []config.Permission) (bool, error) {
    for _, perm := range permissions {
        // Check if permission matches by role
        if perm.Role != "" && hasRole(authCtx.Roles, perm.Role) {
            if hasMethod(perm.Methods, method) {
                // If scopes are specified in permission, validate them
                if len(perm.Scopes) > 0 {
                    if !hasAllScopes(authCtx.Scopes, perm.Scopes) {
                        continue
                    }
                }
                return true, nil
            }
        }

        // Check if permission matches by identity type
        if perm.IdentityType != "" && perm.IdentityType == authCtx.IdentityType {
            if hasMethod(perm.Methods, method) {
                // If scopes are specified in permission, validate them
                if len(perm.Scopes) > 0 {
                    if !hasAllScopes(authCtx.Scopes, perm.Scopes) {
                        continue
                    }
                }
                return true, nil
            }
        }
    }

    return false, nil
}

// hasAnyRole checks if user has any of the required roles
func hasAnyRole(userRoles, requiredRoles []string) bool {
    for _, required := range requiredRoles {
        for _, userRole := range userRoles {
            if strings.EqualFold(userRole, required) {
                return true
            }
        }
    }
    return false
}

// hasRole checks if user has a specific role
func hasRole(userRoles []string, role string) bool {
    for _, userRole := range userRoles {
        if strings.EqualFold(userRole, role) {
            return true
        }
    }
    return false
}

// hasAllScopes checks if user has all required scopes
func hasAllScopes(userScopes, requiredScopes []string) bool {
    for _, required := range requiredScopes {
        found := false
        for _, userScope := range userScopes {
            if matchScope(userScope, required) {
                found = true
                break
            }
        }
        if !found {
            return false
        }
    }
    return true
}

// matchScope supports wildcard scope matching
// e.g., "read:*" matches "read:users", "read:sensors", etc.
func matchScope(userScope, requiredScope string) bool {
    if userScope == requiredScope {
        return true
    }

    // Support wildcard matching
    if strings.HasSuffix(userScope, ":*") {
        prefix := strings.TrimSuffix(userScope, ":*")
        return strings.HasPrefix(requiredScope, prefix+":")
    }

    if strings.HasSuffix(requiredScope, ":*") {
        prefix := strings.TrimSuffix(requiredScope, ":*")
        return strings.HasPrefix(userScope, prefix+":")
    }

    return false
}

// hasMethod checks if method is in allowed methods
func hasMethod(allowedMethods []string, method string) bool {
    for _, allowed := range allowedMethods {
        if strings.EqualFold(allowed, method) {
            return true
        }
    }
    return false
}

// ExtractAuthContext extracts authentication context from request
func ExtractAuthContext(r *http.Request) *AuthContext {
    if authCtxVal := r.Context().Value(authContextKey); authCtxVal != nil {
        if authCtx, ok := authCtxVal.(*AuthContext); ok {
            return authCtx
        }
    }

    return &AuthContext{
        Authenticated: false,
        IdentityType:  "unknown",
    }
}

// SetAuthContext stores auth context in request context
func SetAuthContext(r *http.Request, authCtx *AuthContext) *http.Request {
    ctx := context.WithValue(r.Context(), authContextKey, authCtx)
    return r.WithContext(ctx)
}