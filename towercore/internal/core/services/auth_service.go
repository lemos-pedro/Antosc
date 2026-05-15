package services

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"towercore/internal/core/domain"
	"towercore/internal/core/interfaces"
	"towercore/internal/infrastructure/security"
)

type AuthService struct {
	repo     interfaces.UserRepository
	secret   string
	tokenTTL time.Duration
}

func NewAuthService(repo interfaces.UserRepository, secret string, tokenTTL time.Duration) *AuthService {
	return &AuthService{
		repo:     repo,
		secret:   strings.TrimSpace(secret),
		tokenTTL: tokenTTL,
	}
}

func (s *AuthService) EnsureBootstrapUser(ctx context.Context, username, password, role string) error {
	username = normalizeUsername(username)
	password = strings.TrimSpace(password)
	role = normalizeRole(role)
	if username == "" || password == "" {
		return nil
	}
	if _, err := s.repo.GetByUsername(ctx, username); err == nil {
		return nil
	} else if !errors.Is(err, sql.ErrNoRows) {
		return err
	}

	hash, err := security.HashPassword(password)
	if err != nil {
		return err
	}

	now := time.Now().UTC()
	id, err := newUUIDv4()
	if err != nil {
		return err
	}
	return s.repo.Upsert(ctx, &domain.User{
		ID:           id,
		Username:     username,
		PasswordHash: hash,
		Role:         role,
		CreatedAt:    now,
		UpdatedAt:    now,
	})
}

func (s *AuthService) Login(ctx context.Context, username, password string) (string, *domain.User, time.Time, error) {
	if strings.TrimSpace(s.secret) == "" {
		return "", nil, time.Time{}, errors.New("auth token secret is not configured")
	}
	if s.tokenTTL <= 0 {
		return "", nil, time.Time{}, errors.New("auth token ttl is not configured")
	}

	username = normalizeUsername(username)
	password = strings.TrimSpace(password)
	if username == "" || password == "" {
		return "", nil, time.Time{}, errors.New("username and password are required")
	}

	u, err := s.repo.GetByUsername(ctx, username)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", nil, time.Time{}, errors.New("invalid credentials")
		}
		return "", nil, time.Time{}, err
	}

	if !security.VerifyPassword(u.PasswordHash, password) {
		return "", nil, time.Time{}, errors.New("invalid credentials")
	}

	token, expiresAt, err := security.SignAccessToken(s.secret, u.ID, u.Role, s.tokenTTL)
	if err != nil {
		return "", nil, time.Time{}, err
	}

	return token, u, expiresAt, nil
}

func normalizeUsername(v string) string {
	return strings.ToLower(strings.TrimSpace(v))
}

func normalizeRole(v string) string {
	r := strings.ToLower(strings.TrimSpace(v))
	if r == "" {
		return "viewer"
	}
	return r
}
