package handlers

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"towercore/internal/adapters/mock"
	"towercore/internal/core/services"
)

func TestAuthHandler_LoginOK(t *testing.T) {
	repo := mock.NewUserRepository()
	svc := services.NewAuthService(repo, "secret-123", 30*time.Minute)
	if err := svc.EnsureBootstrapUser(context.Background(), "admin", "admin123", "admin"); err != nil {
		t.Fatalf("bootstrap user failed: %v", err)
	}
	h := NewAuthHandler(svc)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader([]byte(`{"username":"admin","password":"admin123"}`)))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestAuthHandler_LoginUnauthorized(t *testing.T) {
	repo := mock.NewUserRepository()
	svc := services.NewAuthService(repo, "secret-123", 30*time.Minute)
	if err := svc.EnsureBootstrapUser(context.Background(), "admin", "admin123", "admin"); err != nil {
		t.Fatalf("bootstrap user failed: %v", err)
	}
	h := NewAuthHandler(svc)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader([]byte(`{"username":"admin","password":"wrong"}`)))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
	}
}
