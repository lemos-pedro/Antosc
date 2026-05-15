package services

import (
	"context"
	"testing"
	"time"

	"towercore/internal/adapters/mock"
)

func TestAuthService_Login(t *testing.T) {
	repo := mock.NewUserRepository()
	svc := NewAuthService(repo, "secret", time.Hour)
	if err := svc.EnsureBootstrapUser(context.Background(), "admin", "admin123", "admin"); err != nil {
		t.Fatalf("bootstrap failed: %v", err)
	}

	token, user, _, err := svc.Login(context.Background(), "admin", "admin123")
	if err != nil {
		t.Fatalf("login failed: %v", err)
	}
	if token == "" {
		t.Fatal("expected token")
	}
	if user == nil || user.Username != "admin" {
		t.Fatal("expected user admin")
	}
}

func TestAuthService_LoginInvalid(t *testing.T) {
	repo := mock.NewUserRepository()
	svc := NewAuthService(repo, "secret", time.Hour)
	if err := svc.EnsureBootstrapUser(context.Background(), "admin", "admin123", "admin"); err != nil {
		t.Fatalf("bootstrap failed: %v", err)
	}

	_, _, _, err := svc.Login(context.Background(), "admin", "bad")
	if err == nil {
		t.Fatal("expected error")
	}
}
