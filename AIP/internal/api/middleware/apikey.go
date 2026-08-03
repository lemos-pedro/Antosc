package middleware

import (
	"net/http"
	"os"
	"strings"
)

// APIKey protege rotas sensíveis (Power BI, exports) com uma chave simples.
//
// Configuração:
//
//	AIP_API_KEY=segredo-longo
//
// O cliente envia um de:
//
//	Authorization: Bearer <chave>
//	X-API-Key: <chave>
//
// Se AIP_API_KEY estiver vazio, o middleware é transparent (dev local).
// Em produção define sempre a variável.
func APIKey(next http.HandlerFunc) http.HandlerFunc {
	expected := strings.TrimSpace(os.Getenv("AIP_API_KEY"))
	return func(w http.ResponseWriter, r *http.Request) {
		if expected == "" {
			next(w, r)
			return
		}

		got := r.Header.Get("X-API-Key")
		if got == "" {
			auth := r.Header.Get("Authorization")
			if strings.HasPrefix(strings.ToLower(auth), "bearer ") {
				got = strings.TrimSpace(auth[7:])
			}
		}

		if got == "" || got != expected {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"error":{"code":"UNAUTHORIZED","message":"API key em falta ou inválida"}}`))
			return
		}
		next(w, r)
	}
}
