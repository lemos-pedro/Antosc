package database

import (
	"context"
	"database/sql"
	"errors"

	"towercore/internal/core/domain"
)

type UserRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) GetByUsername(ctx context.Context, username string) (*domain.User, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	const query = `
SELECT
	user_id::text,
	username,
	COALESCE(email, ''),
	password_hash,
	role,
	created_at,
	updated_at
FROM users
WHERE username = $1 OR email = $1`

	var u domain.User
	err := r.db.QueryRowContext(ctx, query, username).Scan(
		&u.ID,
		&u.Username,
		&u.Email,
		&u.PasswordHash,
		&u.Role,
		&u.CreatedAt,
		&u.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, sql.ErrNoRows
		}
		return nil, err
	}
	return &u, nil
}

func (r *UserRepository) Upsert(ctx context.Context, user *domain.User) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	const query = `
INSERT INTO users (
	user_id, username, email, password_hash, role, created_at, updated_at
) VALUES (
	$1::uuid, $2, $3, $4, $5, $6, $7
)
ON CONFLICT (username) DO UPDATE
SET
	email         = EXCLUDED.email,
	password_hash = EXCLUDED.password_hash,
	role          = EXCLUDED.role,
	updated_at    = EXCLUDED.updated_at`

	_, err := r.db.ExecContext(
		ctx, query,
		user.ID,
		user.Username,
		user.Email,
		user.PasswordHash,
		user.Role,
		user.CreatedAt,
		user.UpdatedAt,
	)
	return err
}

func (r *UserRepository) Create(ctx context.Context, user *domain.User) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	const query = `
INSERT INTO users (
	user_id, username, email, password_hash, role, created_at, updated_at
) VALUES (
	$1::uuid, $2, $3, $4, $5, $6, $7
)`

	_, err := r.db.ExecContext(
		ctx, query,
		user.ID,
		user.Username,
		user.Email,
		user.PasswordHash,
		user.Role,
		user.CreatedAt,
		user.UpdatedAt,
	)
	return err
}

// List devolve todos os users, sem password_hash (nunca deve sair via API).
func (r *UserRepository) List(ctx context.Context) ([]domain.User, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	const query = `
SELECT
	user_id::text,
	username,
	COALESCE(email, ''),
	role,
	created_at,
	updated_at
FROM users
ORDER BY username ASC`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []domain.User
	for rows.Next() {
		var u domain.User
		if err := rows.Scan(&u.ID, &u.Username, &u.Email, &u.Role, &u.CreatedAt, &u.UpdatedAt); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, rows.Err()
}