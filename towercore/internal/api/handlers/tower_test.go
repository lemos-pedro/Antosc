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

func TestTowerHandler_ServeHTTP(t *testing.T) {
	handler := NewTowerHandler(services.NewTowerService(mock.NewTowerRepository()))
	req := httptest.NewRequest(http.MethodGet, "/api/v1/towers?status=online&limit=10&offset=0", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	if got := rec.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("expected content type application/json, got %q", got)
	}
	if got := rec.Header().Get("X-Total-Count"); got != "1" {
		t.Fatalf("expected X-Total-Count=1, got %q", got)
	}

	var body []domain.Tower
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}
	if len(body) != 1 {
		t.Fatalf("expected 1 tower, got %d", len(body))
	}
	if body[0].Status != domain.TowerStatusOnline {
		t.Fatalf("expected online tower, got %q", body[0].Status)
	}
}

func TestTowerHandler_GetByID_ServeHTTP(t *testing.T) {
	handler := NewTowerHandler(services.NewTowerService(mock.NewTowerRepository()))
	req := httptest.NewRequest(http.MethodGet, "/api/v1/towers/00000000-0000-0000-0000-000000000001", nil)
	req.SetPathValue("id", "00000000-0000-0000-0000-000000000001")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	var body domain.Tower
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}
	if body.ID != "00000000-0000-0000-0000-000000000001" {
		t.Fatalf("expected tower_id 00000000-0000-0000-0000-000000000001, got %q", body.ID)
	}
}

func TestTowerHandler_GetByID_NotFound(t *testing.T) {
	handler := NewTowerHandler(services.NewTowerService(mock.NewTowerRepository()))
	req := httptest.NewRequest(http.MethodGet, "/api/v1/towers/00000000-0000-0000-0000-000000000999", nil)
	req.SetPathValue("id", "00000000-0000-0000-0000-000000000999")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, rec.Code)
	}
}

func TestTowerHandler_Create_ServeHTTP(t *testing.T) {
	handler := NewTowerHandler(services.NewTowerService(mock.NewTowerRepository()))
	payload := []byte(`{"name":"Tower-900","status":"online","operator_id":"00000000-0000-0000-0000-000000000101","region_id":"00000000-0000-0000-0000-000000000201","availability_30d":97.5}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/towers", bytes.NewReader(payload))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, rec.Code)
	}
	if got := rec.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("expected content type application/json, got %q", got)
	}

	var created domain.Tower
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}
	if created.ID == "" {
		t.Fatalf("expected generated tower_id")
	}
	if created.Name != "Tower-900" {
		t.Fatalf("expected name Tower-900, got %q", created.Name)
	}
}

func TestTowerHandler_Create_BadRequest(t *testing.T) {
	handler := NewTowerHandler(services.NewTowerService(mock.NewTowerRepository()))
	req := httptest.NewRequest(http.MethodPost, "/api/v1/towers", bytes.NewReader([]byte(`{"status":"online"}`)))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestTowerHandler_ConfigureSNMP_ServeHTTP(t *testing.T) {
	handler := NewTowerHandler(services.NewTowerService(mock.NewTowerRepository()))
	body := []byte(`{"vendor":"eltek","snmp_enabled":true,"snmp_target":"10.10.0.200","snmp_community":"private"}`)
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/towers/00000000-0000-0000-0000-000000000001/snmp", bytes.NewReader(body))
	req.SetPathValue("id", "00000000-0000-0000-0000-000000000001")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	var updated domain.Tower
	if err := json.Unmarshal(rec.Body.Bytes(), &updated); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}
	if !updated.SNMPEnabled {
		t.Fatalf("expected snmp_enabled=true")
	}
	if updated.SNMPCommunity != "" {
		t.Fatalf("expected snmp_community to be hidden in response")
	}
}
