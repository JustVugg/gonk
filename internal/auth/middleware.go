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
		if routeAuth == nil || !requiresAuthentication(routeAuth) {
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

		if requiresAdditionalClientCert(routeAuth) {
			certCtx, certErr := ValidateMTLS(r, routeAuth)
			if certErr != nil || certCtx == nil || !certCtx.Authenticated {
				log.Printf("mTLS authentication failed: %v", certErr)
				respondUnauthorized(w, "client certificate required")
				return
			}
			mergeClientCertContext(authCtx, certCtx)
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
		if authConfig != nil && authConfig.JWT != nil && authConfig.JWT.Enabled {
			return ValidateJWT(r, authConfig.JWT)
		}

	case "api_key":
		if authConfig != nil && authConfig.APIKey != nil && authConfig.APIKey.Enabled {
			return ValidateAPIKey(r, authConfig.APIKey)
		}

	case "mtls", "":
		if routeAuth.RequireClientCert {
			return ValidateMTLS(r, routeAuth)
		}

	default:
		return nil, nil
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
			if authConfig != nil && authConfig.JWT != nil && authConfig.JWT.Enabled {
				authCtx, err = ValidateJWT(r, authConfig.JWT)
			}

		case "api_key":
			if authConfig != nil && authConfig.APIKey != nil && authConfig.APIKey.Enabled {
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

func requiresAuthentication(routeAuth *config.RouteAuth) bool {
	return routeAuth.Required || routeAuth.RequireClientCert || len(routeAuth.RequireEither) > 0
}

func requiresAdditionalClientCert(routeAuth *config.RouteAuth) bool {
	return routeAuth.RequireClientCert && routeAuth.Type != "mtls" && len(routeAuth.RequireEither) == 0
}

func mergeClientCertContext(authCtx, certCtx *AuthContext) {
	authCtx.CertCommonName = certCtx.CertCommonName
	if authCtx.ClientID == "" {
		authCtx.ClientID = certCtx.ClientID
	}
	authCtx.Roles = appendUnique(authCtx.Roles, certCtx.Roles...)
	authCtx.Scopes = appendUnique(authCtx.Scopes, certCtx.Scopes...)
}

func appendUnique(values []string, additions ...string) []string {
	seen := make(map[string]bool, len(values)+len(additions))
	for _, value := range values {
		seen[value] = true
	}
	for _, value := range additions {
		if value == "" || seen[value] {
			continue
		}
		values = append(values, value)
		seen[value] = true
	}
	return values
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
