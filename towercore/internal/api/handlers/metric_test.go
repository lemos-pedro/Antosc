package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"towercore/internal/adapters/mock"
	"towercore/internal/core/domain"
	"towercore/internal/core/services"
)

func TestMetricHandler_Create_ServeHTTP(t *testing.T) {
	handler := NewMetricHandler(services.NewMetricService(mock.NewMetricRepository()))
	payload := []byte(`{"tower_id":"00000000-0000-0000-0000-000000000001","metrics":{"voltage":48.2,"temperature":34.1}}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/metrics", bytes.NewReader(payload))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, rec.Code)
	}

	var created domain.Metric
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}
	if created.ID == "" {
		t.Fatalf("expected generated metric_id")
	}
}

func TestMetricHandler_Create_BadRequest(t *testing.T) {
	handler := NewMetricHandler(services.NewMetricService(mock.NewMetricRepository()))
	req := httptest.NewRequest(http.MethodPost, "/api/v1/metrics", bytes.NewReader([]byte(`{"tower_id":"00000000-0000-0000-0000-000000000001","metrics":{}}`)))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestMetricHandler_List_ServeHTTP(t *testing.T) {
	handler := NewMetricHandler(services.NewMetricService(mock.NewMetricRepository()))

	first := httptest.NewRequest(http.MethodPost, "/api/v1/metrics", bytes.NewReader([]byte(`{"tower_id":"00000000-0000-0000-0000-000000000001","metrics":{"voltage":48.2}}`)))
	firstRec := httptest.NewRecorder()
	handler.ServeHTTP(firstRec, first)

	second := httptest.NewRequest(http.MethodPost, "/api/v1/metrics", bytes.NewReader([]byte(`{"tower_id":"00000000-0000-0000-0000-000000000002","metrics":{"voltage":47.7}}`)))
	secondRec := httptest.NewRecorder()
	handler.ServeHTTP(secondRec, second)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/metrics?tower_id=00000000-0000-0000-0000-000000000001&limit=10&offset=0", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	if got := rec.Header().Get("X-Total-Count"); got != "1" {
		t.Fatalf("expected X-Total-Count=1, got %q", got)
	}
}
