package auth

import (
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/JustVugg/gonk/internal/config"
	"github.com/golang-jwt/jwt/v5"
)

func TestRequiresAuthenticationForRequireEither(t *testing.T) {
	routeAuth := &config.RouteAuth{
		RequireEither: []string{"client_cert", "api_key"},
	}

	if !requiresAuthentication(routeAuth) {
		t.Fatal("require_either should require authentication even when required is omitted")
	}
}

func TestRequiresAdditionalClientCertForJWTAndMTLS(t *testing.T) {
	routeAuth := &config.RouteAuth{
		Type:              "jwt",
		Required:          true,
		RequireClientCert: true,
	}

	if !requiresAdditionalClientCert(routeAuth) {
		t.Fatal("jwt routes with require_client_cert should require both JWT and client certificate")
	}
}

func TestMiddlewareAllowsJWTWithRoleScopeAndPermission(t *testing.T) {
	secret := "test-secret"
	authConfig := &config.AuthConfig{
		JWT: &config.JWTConfig{
			Enabled:        true,
			SecretKey:      secret,
			Header:         "Authorization",
			Prefix:         "Bearer",
			ExpiryCheck:    true,
			ValidateRoles:  true,
			ValidateScopes: true,
		},
	}
	routeAuth := &config.RouteAuth{
		Type:           "jwt",
		Required:       true,
		AllowedRoles:   []string{"admin"},
		RequiredScopes: []string{"read:api"},
		Permissions: []config.Permission{
			{Role: "admin", Methods: []string{"GET"}, Scopes: []string{"read:api"}},
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/api/ping", nil)
	req.Header.Set("Authorization", "Bearer "+signedTestJWT(t, secret, []string{"admin"}, []string{"read:api"}))
	rr := httptest.NewRecorder()

	Middleware(authConfig, routeAuth, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authCtx := GetAuthContext(r)
		if !authCtx.Authenticated || authCtx.UserID != "test-user" {
			t.Fatalf("unexpected auth context: %#v", authCtx)
		}
		w.WriteHeader(http.StatusNoContent)
	})).ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d, body = %s", rr.Code, http.StatusNoContent, rr.Body.String())
	}
}

func TestMiddlewareRejectsMissingJWT(t *testing.T) {
	authConfig := &config.AuthConfig{
		JWT: &config.JWTConfig{
			Enabled:   true,
			SecretKey: "test-secret",
			Header:    "Authorization",
			Prefix:    "Bearer",
		},
	}
	routeAuth := &config.RouteAuth{Type: "jwt", Required: true}

	req := httptest.NewRequest(http.MethodGet, "/api/ping", nil)
	rr := httptest.NewRecorder()

	Middleware(authConfig, routeAuth, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("next handler should not be called")
	})).ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusUnauthorized)
	}
}

func TestMiddlewareRequiresClientCertWhenJWTRouteRequiresCert(t *testing.T) {
	secret := "test-secret"
	authConfig := &config.AuthConfig{
		JWT: &config.JWTConfig{
			Enabled:   true,
			SecretKey: secret,
			Header:    "Authorization",
			Prefix:    "Bearer",
		},
	}
	routeAuth := &config.RouteAuth{
		Type:              "jwt",
		Required:          true,
		RequireClientCert: true,
	}

	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	req.Header.Set("Authorization", "Bearer "+signedTestJWT(t, secret, []string{"admin"}, []string{"read:api"}))
	rr := httptest.NewRecorder()

	Middleware(authConfig, routeAuth, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("next handler should not be called without a client certificate")
	})).ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusUnauthorized)
	}
}

func TestMiddlewareAllowsAPIKeyWithIdentityPermission(t *testing.T) {
	authConfig := &config.AuthConfig{
		APIKey: &config.APIKeyConfig{
			Enabled: true,
			Header:  "X-API-Key",
			Keys: []config.APIKey{
				{
					Key:      "device-secret",
					ClientID: "device-001",
					Scopes:   []string{"write:sensors"},
				},
			},
		},
	}
	routeAuth := &config.RouteAuth{
		Type:     "api_key",
		Required: true,
		Permissions: []config.Permission{
			{IdentityType: "service", Methods: []string{"POST"}, Scopes: []string{"write:sensors"}},
		},
	}

	req := httptest.NewRequest(http.MethodPost, "/api/sensors", nil)
	req.Header.Set("X-API-Key", "device-secret")
	rr := httptest.NewRecorder()

	Middleware(authConfig, routeAuth, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("X-Client-ID"); got != "device-001" {
			t.Fatalf("X-Client-ID = %q, want device-001", got)
		}
		w.WriteHeader(http.StatusAccepted)
	})).ServeHTTP(rr, req)

	if rr.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want %d, body = %s", rr.Code, http.StatusAccepted, rr.Body.String())
	}
}

func TestMiddlewareAllowsRequireEitherAPIKey(t *testing.T) {
	authConfig := &config.AuthConfig{
		APIKey: &config.APIKeyConfig{
			Enabled: true,
			Header:  "X-API-Key",
			Keys: []config.APIKey{
				{Key: "service-secret", ClientID: "svc-001", Scopes: []string{"write:api"}},
			},
		},
	}
	routeAuth := &config.RouteAuth{
		RequireEither: []string{"client_cert", "api_key"},
		Permissions: []config.Permission{
			{IdentityType: "service", Methods: []string{"POST"}, Scopes: []string{"write:api"}},
		},
	}

	req := httptest.NewRequest(http.MethodPost, "/api", nil)
	req.Header.Set("X-API-Key", "service-secret")
	rr := httptest.NewRecorder()

	Middleware(authConfig, routeAuth, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusAccepted)
	})).ServeHTTP(rr, req)

	if rr.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want %d, body = %s", rr.Code, http.StatusAccepted, rr.Body.String())
	}
}

func TestValidateMTLSMapsCertificateIdentity(t *testing.T) {
	cert := &x509.Certificate{
		Subject: pkix.Name{
			CommonName:   "Device-001",
			Organization: []string{"role:field-device"},
		},
	}
	req := httptest.NewRequest(http.MethodGet, "/device", nil)
	req.TLS = &tls.ConnectionState{PeerCertificates: []*x509.Certificate{cert}}

	authCtx, err := ValidateMTLS(req, &config.RouteAuth{
		RequireClientCert: true,
		CertToRoleMapping: map[string]string{
			"CN=Device-*": "device",
		},
	})
	if err != nil {
		t.Fatalf("ValidateMTLS() returned error: %v", err)
	}
	if !authCtx.Authenticated || authCtx.ClientID != "Device-001" || authCtx.CertCommonName != "Device-001" {
		t.Fatalf("unexpected auth context: %#v", authCtx)
	}
	if !hasRole(authCtx.Roles, "device") || !hasRole(authCtx.Roles, "field-device") {
		t.Fatalf("roles = %v, want device and field-device", authCtx.Roles)
	}
}

func TestValidateMTLSRequiresTLSConnection(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/device", nil)
	_, err := ValidateMTLS(req, &config.RouteAuth{RequireClientCert: true})
	if err == nil {
		t.Fatal("ValidateMTLS() should fail without TLS")
	}
}

func TestValidateAuthorizationSupportsWildcardScopes(t *testing.T) {
	authorized, err := ValidateAuthorization(
		httptest.NewRequest(http.MethodGet, "/api/users", nil),
		&config.RouteAuth{
			Required:       true,
			RequiredScopes: []string{"read:users"},
		},
		&AuthContext{
			Authenticated: true,
			Scopes:        []string{"read:*"},
		},
	)
	if err != nil {
		t.Fatalf("ValidateAuthorization() returned error: %v", err)
	}
	if !authorized {
		t.Fatal("wildcard scope should authorize read:users")
	}
}

func signedTestJWT(t *testing.T, secret string, roles, scopes []string) string {
	t.Helper()

	claims := CustomClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "test-user",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
		UserID: "test-user",
		Roles:  roles,
		Scopes: scopes,
	}

	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("failed to sign test JWT: %v", err)
	}
	return token
}
