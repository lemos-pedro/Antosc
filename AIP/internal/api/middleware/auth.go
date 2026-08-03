package middleware

import (
	"net/http"
	"strings"

	"github.com/antosc/aip/internal/auth"
)

// Authenticate exige JWT Bearer válido e injeta Principal no context.
func Authenticate(tokens *auth.TokenService) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			raw := bearerToken(r)
			if raw == "" {
				writeAuthError(w, http.StatusUnauthorized, "token em falta")
				return
			}
			claims, err := tokens.Parse(raw)
			if err != nil {
				writeAuthError(w, http.StatusUnauthorized, "token inválido ou expirado")
				return
			}
			p := auth.Principal{
				ID:    claims.Sub,
				Email: claims.Email,
				Role:  claims.Role,
				Name:  claims.Name,
			}
			next(w, r.WithContext(auth.WithPrincipal(r.Context(), p)))
		}
	}
}

// RequireAdmin exige role=admin (depois de Authenticate).
func RequireAdmin(next http.HandlerFunc) http.HandlerFunc {
	return RequireRoles("admin")(next)
}

// RequireRoles exige que o Principal tenha uma das roles indicadas.
// admin passa sempre. Deve ser usado depois de Authenticate.
func RequireRoles(roles ...string) func(http.HandlerFunc) http.HandlerFunc {
	allowed := make(map[string]struct{}, len(roles)+1)
	for _, r := range roles {
		allowed[r] = struct{}{}
	}
	allowed["admin"] = struct{}{}

	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			p, ok := auth.PrincipalFrom(r.Context())
			if !ok {
				writeAuthError(w, http.StatusUnauthorized, "não autenticado")
				return
			}
			if _, ok := allowed[p.Role]; !ok {
				writeAuthError(w, http.StatusForbidden, "sem permissão para esta operação")
				return
			}
			next(w, r)
		}
	}
}

// AuthenticateOrAPIKey aceita JWT Bearer OU X-API-Key.
func AuthenticateOrAPIKey(tokens *auth.TokenService, apiKey string) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			if raw := bearerToken(r); raw != "" && tokens != nil {
				if claims, err := tokens.Parse(raw); err == nil {
					p := auth.Principal{
						ID: claims.Sub, Email: claims.Email,
						Role: claims.Role, Name: claims.Name,
					}
					next(w, r.WithContext(auth.WithPrincipal(r.Context(), p)))
					return
				}
			}

			if apiKey != "" {
				got := r.Header.Get("X-API-Key")
				if got == "" {
					if b := bearerToken(r); b != "" && b == apiKey {
						got = b
					}
				}
				if got == apiKey {
					next(w, r)
					return
				}
			}

			writeAuthError(w, http.StatusUnauthorized, "autenticação necessária (Bearer JWT ou X-API-Key)")
		}
	}
}

func bearerToken(r *http.Request) string {
	h := r.Header.Get("Authorization")
	if !strings.HasPrefix(strings.ToLower(h), "bearer ") {
		return ""
	}
	return strings.TrimSpace(h[7:])
}

func writeAuthError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(`{"error":{"code":"AUTH","message":"` + msg + `"}}`))
}
