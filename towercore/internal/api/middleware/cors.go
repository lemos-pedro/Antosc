package middleware

import "net/http"

func CORS(allowedOrigins []string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			origin := r.Header.Get("Origin")

			allowed := false

			for _, o := range allowedOrigins {
				if o == origin {
					allowed = true
					break
				}
			}

			if !allowed {
				if r.Method == http.MethodOptions {
					http.Error(w, "CORS origin not allowed", http.StatusForbidden)
					return
				}

				next.ServeHTTP(w, r)
				return
			}

			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")

			w.Header().Set(
				"Access-Control-Allow-Methods",
				"GET, POST, PUT, PATCH, DELETE, OPTIONS",
			)

			w.Header().Set(
				"Access-Control-Allow-Headers",
				"Content-Type, Authorization, X-Request-Id, X-API-Key, ngrok-skip-browser-warning",
			)

			w.Header().Set(
				"Access-Control-Allow-Credentials",
				"true",
			)

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}