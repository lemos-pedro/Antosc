package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"towercore/internal/adapters/mock"
	"towercore/internal/core/domain"
	"towercore/internal/core/services"
)

func TestAuditHandler_List_ServeHTTP(t *testing.T) {
	repo := mock.NewAuditRepository()
	_ = repo.Create(nil, &domain.AuditLog{
		ID:         "00000000-0000-0000-0000-000000000901",
		Actor:      "test-user",
		Action:     "snmp.credentials.update",
		Resource:   "tower",
		ResourceID: "00000000-0000-0000-0000-000000000001",
		Details:    "updated",
		CreatedAt:  time.Now().UTC(),
	})

	handler := NewAuditHandler(services.NewAuditService(repo))
	req := httptest.NewRequest(http.MethodGet, "/api/v1/audit-logs?action=snmp.credentials.update&limit=10&offset=0", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	if got := rec.Header().Get("X-Total-Count"); got != "1" {
		t.Fatalf("expected X-Total-Count=1, got %q", got)
	}
}

func TestAuditHandler_GetByID_ServeHTTP(t *testing.T) {
	repo := mock.NewAuditRepository()
	_ = repo.Create(nil, &domain.AuditLog{
		ID:         "00000000-0000-0000-0000-000000000902",
		Actor:      "test-user",
		Action:     "snmp.credentials.update",
		Resource:   "tower",
		ResourceID: "00000000-0000-0000-0000-000000000001",
		Details:    "updated",
		CreatedAt:  time.Now().UTC(),
	})

	handler := NewAuditHandler(services.NewAuditService(repo))
	req := httptest.NewRequest(http.MethodGet, "/api/v1/audit-logs/00000000-0000-0000-0000-000000000902", nil)
	req.SetPathValue("id", "00000000-0000-0000-0000-000000000902")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	var got domain.AuditLog
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("failed to decode body: %v", err)
	}
	if got.ID != "00000000-0000-0000-0000-000000000902" {
		t.Fatalf("unexpected audit id %q", got.ID)
	}
}

func TestAuditHandler_ExportCSV_ServeHTTP(t *testing.T) {
	repo := mock.NewAuditRepository()
	_ = repo.Create(nil, &domain.AuditLog{
		ID:         "00000000-0000-0000-0000-000000000903",
		Actor:      "test-user",
		Action:     "snmp.credentials.update",
		Resource:   "tower",
		ResourceID: "00000000-0000-0000-0000-000000000001",
		Details:    "updated",
		CreatedAt:  time.Now().UTC(),
	})

	handler := NewAuditHandler(services.NewAuditService(repo))
	req := httptest.NewRequest(http.MethodGet, "/api/v1/audit-logs/export.csv", bytes.NewReader(nil))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct == "" || ct[:8] != "text/csv" {
		t.Fatalf("expected csv content type, got %q", ct)
	}
}
