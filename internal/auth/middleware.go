package auth

import (
    "net/http"
    
    "github.com/JustVugg/gonk/internal/config"
)

func Middleware(authConfig *config.AuthConfig, routeAuth *config.RouteAuth, next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if routeAuth == nil || !routeAuth.Required {
            next.ServeHTTP(w, r)
            return
        }

        var authenticated bool
        var err error

        switch routeAuth.Type {
        case "jwt":
            if authConfig.JWT != nil && authConfig.JWT.Enabled {
                authenticated, err = ValidateJWT(r, authConfig.JWT)
            }
        case "api_key":
            if authConfig.APIKey != nil && authConfig.APIKey.Enabled {
                authenticated, err = ValidateAPIKey(r, authConfig.APIKey)
            }
        default:
            next.ServeHTTP(w, r)
            return
        }

        if err != nil || !authenticated {
            w.Header().Set("Content-Type", "application/json")
            w.WriteHeader(http.StatusUnauthorized)
            w.Write([]byte(`{"error":"unauthorized"}`))
            return
        }

        next.ServeHTTP(w, r)
    })

}

