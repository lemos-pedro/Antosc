package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"towercore/internal/infrastructure/security"
)

func TestAuth_UserToken(t *testing.T) {
	tok, _, err := security.SignAccessToken("secret", "user-1", "admin", time.Hour)
	if err != nil {
		t.Fatalf("failed to sign token: %v", err)
	}

	mw := Auth("", "", "secret")
	h := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := UserIDFromContext(r.Context()); got != "user-1" {
			t.Fatalf("expected user-1, got %s", got)
		}
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/secure", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestAuth_LegacyToken(t *testing.T) {
	mw := Auth("62af8704764faf8ea82fc61ce9c4c3908b6cb97d463a634e9e587d7c885db0ef", "test-token", "")
	h := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/secure", nil)
	req.Header.Set("X-API-Key", "test-key")
	req.Header.Set("Authorization", "Bearer test-token")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
}
