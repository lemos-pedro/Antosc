package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"towercore/internal/infrastructure/config"
)

type HealthHandler struct {
	cfg config.Config
}

func NewHealthHandler(cfg config.Config) *HealthHandler {
	return &HealthHandler{cfg: cfg}
}

func (h *HealthHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if requestID := r.Header.Get("X-Request-Id"); requestID != "" {
		w.Header().Set("X-Request-Id", requestID)
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"status":  "ok",
		"service": h.cfg.AppName,
		"time":    time.Now().UTC().Format(time.RFC3339),
	})
}
