package routes

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"towercore/internal/infrastructure/config"
	"towercore/internal/infrastructure/logger"
)

func TestRouter_HealthRoutes(t *testing.T) {
	cfg := config.Config{AppName: "towercore-api"}
	log := logger.New("info")
	router := NewRouter(cfg, log)

	tests := []struct {
		name string
		path string
	}{
		{name: "v1 health", path: "/api/v1/health"},
		{name: "legacy health", path: "/health"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			rec := httptest.NewRecorder()

			router.ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Fatalf("expected status %d for %s, got %d", http.StatusOK, tt.path, rec.Code)
			}
		})
	}
}

func TestRouter_NotFound(t *testing.T) {
	cfg := config.Config{AppName: "towercore-api"}
	log := logger.New("info")
	router := NewRouter(cfg, log)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/unknown", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, rec.Code)
	}
}
