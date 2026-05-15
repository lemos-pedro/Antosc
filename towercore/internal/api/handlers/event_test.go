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

func TestEventHandler_Create_ServeHTTP(t *testing.T) {
	handler := NewEventHandler(services.NewEventService(mock.NewEventRepository()))
	payload := []byte(`{"tower_id":"00000000-0000-0000-0000-000000000001","type":"alarm","severity":"warning","message":"power fluctuation"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/events", bytes.NewReader(payload))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, rec.Code)
	}

	var created domain.Event
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}
	if created.ID == "" {
		t.Fatalf("expected generated event_id")
	}
}

func TestEventHandler_Create_BadRequest(t *testing.T) {
	handler := NewEventHandler(services.NewEventService(mock.NewEventRepository()))
	req := httptest.NewRequest(http.MethodPost, "/api/v1/events", bytes.NewReader([]byte(`{"tower_id":"","type":"alarm","severity":"warning","message":"x"}`)))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestEventHandler_List_ServeHTTP(t *testing.T) {
	handler := NewEventHandler(services.NewEventService(mock.NewEventRepository()))

	first := httptest.NewRequest(http.MethodPost, "/api/v1/events", bytes.NewReader([]byte(`{"tower_id":"00000000-0000-0000-0000-000000000001","type":"alarm","severity":"warning","message":"a"}`)))
	firstRec := httptest.NewRecorder()
	handler.ServeHTTP(firstRec, first)

	second := httptest.NewRequest(http.MethodPost, "/api/v1/events", bytes.NewReader([]byte(`{"tower_id":"00000000-0000-0000-0000-000000000001","type":"info","severity":"info","message":"b"}`)))
	secondRec := httptest.NewRecorder()
	handler.ServeHTTP(secondRec, second)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/events?tower_id=00000000-0000-0000-0000-000000000001&severity=warning&limit=10&offset=0", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	if got := rec.Header().Get("X-Total-Count"); got != "1" {
		t.Fatalf("expected X-Total-Count=1, got %q", got)
	}
}
