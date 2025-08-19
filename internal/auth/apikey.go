package auth

import (
    "net/http"
    
    "github.com/zrufy/gonk/internal/config"
)

func ValidateAPIKey(r *http.Request, cfg *config.APIKeyConfig) (bool, error) {
    key := r.Header.Get(cfg.Header)
    if key == "" {
        return false, nil
    }

    for _, apiKey := range cfg.Keys {
        if apiKey.Key == key {
            // Set client ID in context for rate limiting
            r.Header.Set("X-Client-ID", apiKey.ClientID)
            return true, nil
        }
    }

    return false, nil
}
