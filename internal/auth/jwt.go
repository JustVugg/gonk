package auth

import (
    "fmt"
    "net/http"
    "strings"
    "time"
    
    "github.com/golang-jwt/jwt/v5"
    "github.com/zrufy/gonk/internal/config"
)

func ValidateJWT(r *http.Request, cfg *config.JWTConfig) (bool, error) {
    tokenString := extractToken(r, cfg)
    if tokenString == "" {
        return false, fmt.Errorf("no token provided")
    }

    // Usa RegisteredClaims per gestione automatica di exp
    token, err := jwt.ParseWithClaims(tokenString, &jwt.RegisteredClaims{}, func(token *jwt.Token) (interface{}, error) {
        if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
            return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
        }
        return []byte(cfg.SecretKey), nil
    })

    if err != nil {
        return false, err
    }

    if !token.Valid {
        return false, fmt.Errorf("invalid token")
    }

    // Con RegisteredClaims, la validazione di exp è automatica se ExpiryCheck è true
    if cfg.ExpiryCheck && token.Claims != nil {
        if claims, ok := token.Claims.(*jwt.RegisteredClaims); ok {
            if claims.ExpiresAt != nil && claims.ExpiresAt.Before(time.Now()) {
                return false, fmt.Errorf("token expired")
            }
        }
    }

    return true, nil
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
