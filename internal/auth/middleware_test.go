package auth

import (
	"testing"

	"github.com/JustVugg/gonk/internal/config"
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
