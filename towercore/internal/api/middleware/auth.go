package middleware

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strings"

	"towercore/internal/infrastructure/security"
	"towercore/pkg/apierror"
)

type ctxKeyUserID struct{}

// Auth aceita dois fluxos de autenticacao:
// 1) token de utilizador assinado (Authorization: Bearer <token>);
// 2) token tecnico legado (X-API-Key + Authorization: Bearer <AUTH_BEARER_TOKEN>).
func Auth(apiKeyHash, legacyBearerToken, userTokenSecret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			bearer := r.Header.Get("Authorization")
			if !strings.HasPrefix(bearer, "Bearer ") {
				apierror.Unauthorized(w, r)
				return
			}
			tokenStr := strings.TrimPrefix(bearer, "Bearer ")
			if userTokenSecret != "" {
				claims, err := security.VerifyAccessToken(userTokenSecret, tokenStr)
				if err == nil {
					ctx := context.WithValue(r.Context(), ctxKeyUserID{}, claims.Sub)
					next.ServeHTTP(w, r.WithContext(ctx))
					return
				}
				if strings.Contains(err.Error(), "token expired") {
					apierror.Unauthorized(w, r)
					return
				}
			}

			actor, ok := validateLegacyServiceToken(r, apiKeyHash, legacyBearerToken, tokenStr)
			if !ok {
				apierror.Unauthorized(w, r)
				return
			}
			ctx := context.WithValue(r.Context(), ctxKeyUserID{}, actor)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func validateLegacyServiceToken(r *http.Request, apiKeyHash, expectedBearer, bearerToken string) (string, bool) {
	if expectedBearer == "" || bearerToken != expectedBearer {
		return "", false
	}
	apiKey := r.Header.Get("X-API-Key")
	if apiKey == "" {
		return "", false
	}
	h := sha256.Sum256([]byte(apiKey))
	if hex.EncodeToString(h[:]) != apiKeyHash {
		return "", false
	}

	actor := strings.TrimSpace(r.Header.Get("X-User-Id"))
	if actor == "" {
		actor = "api:" + hex.EncodeToString(h[:])[:12]
	}
	return actor, true
}

func UserIDFromContext(ctx context.Context) string {
	v, _ := ctx.Value(ctxKeyUserID{}).(string)
	return v
}
