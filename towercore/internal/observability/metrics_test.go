package observability

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMetricsHandler_ExposesPrometheusMetrics(t *testing.T) {
	metrics, err := New("towercore-api", nil)
	if err != nil {
		t.Fatalf("failed to initialize metrics: %v", err)
	}

	handler := metrics.Instrument("/api/v1/towers", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	}))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/towers", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	metricsReq := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	metricsRec := httptest.NewRecorder()
	metrics.Handler().ServeHTTP(metricsRec, metricsReq)

	if metricsRec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, metricsRec.Code)
	}
	body := metricsRec.Body.String()
	if !strings.Contains(body, "towercore_http_requests_total") {
		t.Fatalf("expected metrics body to contain requests_total, got %q", body)
	}
	if !strings.Contains(body, `route="/api/v1/towers"`) {
		t.Fatalf("expected metrics body to contain route label, got %q", body)
	}
}
