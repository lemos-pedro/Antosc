package mock

import (
	"context"
	"database/sql"
	"strings"
	"sync"
	"time"

	"towercore/internal/core/domain"
)

type UserRepository struct {
	mu    sync.RWMutex
	users map[string]domain.User
}

func NewUserRepository() *UserRepository {
	return &UserRepository{users: make(map[string]domain.User)}
}

func (r *UserRepository) GetByUsername(_ context.Context, username string) (*domain.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	u, ok := r.users[strings.ToLower(strings.TrimSpace(username))]
	if !ok {
		return nil, sql.ErrNoRows
	}
	c := u
	return &c, nil
}

func (r *UserRepository) Upsert(_ context.Context, user *domain.User) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now().UTC()
	key := strings.ToLower(strings.TrimSpace(user.Username))
	u := *user
	if existing, ok := r.users[key]; ok {
		u.CreatedAt = existing.CreatedAt
		if u.CreatedAt.IsZero() {
			u.CreatedAt = now
		}
	} else if u.CreatedAt.IsZero() {
		u.CreatedAt = now
	}
	if u.UpdatedAt.IsZero() {
		u.UpdatedAt = now
	}
	r.users[key] = u
	return nil
}
