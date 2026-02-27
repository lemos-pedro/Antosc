package middleware

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strings"

	"towercore/pkg/apierror"
)

type ctxKeyUserID struct{}

// Auth valida API Key + bearer token em dois passos:
// 1. Header X-API-Key deve corresponder ao hash configurado.
// 2. Header Authorization: Bearer <token> deve corresponder ao segredo esperado.
// Esta estratégia é simples para Fase 1 e pode evoluir para JWT assinado.
func Auth(apiKeyHash, jwtSecret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Passo 1: validar API Key
			apiKey := r.Header.Get("X-API-Key")
			if apiKey == "" {
				apierror.Unauthorized(w, r)
				return
			}
			h := sha256.Sum256([]byte(apiKey))
			if hex.EncodeToString(h[:]) != apiKeyHash {
				apierror.Unauthorized(w, r)
				return
			}

			// Passo 2: validar token bearer
			bearer := r.Header.Get("Authorization")
			if !strings.HasPrefix(bearer, "Bearer ") {
				apierror.Unauthorized(w, r)
				return
			}
			tokenStr := strings.TrimPrefix(bearer, "Bearer ")
			if jwtSecret == "" || tokenStr != jwtSecret {
				apierror.Unauthorized(w, r)
				return
			}

			// Na Fase 1 usamos um user fixo ao validar token.
			ctx := context.WithValue(r.Context(), ctxKeyUserID{}, "system")
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func UserIDFromContext(ctx context.Context) string {
	v, _ := ctx.Value(ctxKeyUserID{}).(string)
	return v
}
