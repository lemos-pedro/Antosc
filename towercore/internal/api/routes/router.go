package routes

import (
	"net/http"

	"towercore/internal/api/handlers"
	"towercore/internal/infrastructure/config"
	"towercore/internal/infrastructure/logger"
)

func NewRouter(cfg config.Config, log *logger.Logger) http.Handler {
	mux := http.NewServeMux()

	healthHandler := handlers.NewHealthHandler(cfg)

	mux.Handle("GET /api/v1/health", healthHandler)
	mux.Handle("GET /health", healthHandler)

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		log.Errorf("route not found: method=%s path=%s", r.Method, r.URL.Path)
		http.NotFound(w, r)
	})

	return mux
}
