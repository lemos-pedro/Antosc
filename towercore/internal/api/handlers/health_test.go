package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"towercore/internal/infrastructure/config"
)

func TestHealthHandler_ServeHTTP(t *testing.T) {
	handler := NewHealthHandler(config.Config{AppName: "towercore-api"})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	req.Header.Set("X-Request-Id", "req-123")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	if got := rec.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("expected content type application/json, got %q", got)
	}
	if got := rec.Header().Get("X-Request-Id"); got != "req-123" {
		t.Fatalf("expected request id header to be echoed, got %q", got)
	}

	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}
	if body["status"] != "ok" {
		t.Fatalf("expected status=ok, got %q", body["status"])
	}
	if body["service"] != "towercore-api" {
		t.Fatalf("expected service=towercore-api, got %q", body["service"])
	}
	if body["time"] == "" {
		t.Fatalf("expected non-empty time field")
	}
}
