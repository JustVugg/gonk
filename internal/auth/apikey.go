package auth

import (
    "fmt"
    "net/http"
    
    "github.com/JustVugg/gonk/internal/config"
)

// ValidateAPIKey validates API key and returns auth context
func ValidateAPIKey(r *http.Request, cfg *config.APIKeyConfig) (*AuthContext, error) {
    key := r.Header.Get(cfg.Header)
    if key == "" {
        return nil, fmt.Errorf("no API key provided")
    }

    for _, apiKey := range cfg.Keys {
        if apiKey.Key == key {
            // Build auth context
            authCtx := &AuthContext{
                Authenticated: true,
                IdentityType:  "service", // API keys are typically for services
                ClientID:      apiKey.ClientID,
                Roles:         apiKey.Roles,
                Scopes:        apiKey.Scopes,
            }

            // Set client ID in header for rate limiting
            r.Header.Set("X-Client-ID", apiKey.ClientID)
            
            return authCtx, nil
        }
    }

    return nil, fmt.Errorf("invalid API key")
}
