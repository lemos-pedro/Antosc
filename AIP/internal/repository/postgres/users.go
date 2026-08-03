package postgres

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

var (
	ErrUserNotFound      = errors.New("utilizador não encontrado")
	ErrEmailTaken        = errors.New("email já registado")
	ErrInvalidCredentials = errors.New("credenciais inválidas")
)

type User struct {
	ID               string
	Email            string
	FullName         string
	PasswordHash     string
	Role             string
	Active           bool
	CreatedAt        time.Time
	UpdatedAt        time.Time
	LastLoginAt      sql.NullTime
	FailedLoginCount int
	LockedUntil      sql.NullTime
}

type UserRepository interface {
	Create(ctx context.Context, u User) (User, error)
	GetByEmail(ctx context.Context, email string) (*User, error)
	GetByID(ctx context.Context, id string) (*User, error)
	List(ctx context.Context, limit int) ([]User, error)
	UpdateActive(ctx context.Context, id string, active bool) error
	UpdateRole(ctx context.Context, id string, role string) error
	UpdatePassword(ctx context.Context, id, passwordHash string) error
	TouchLastLogin(ctx context.Context, id string) error
	Count(ctx context.Context) (int, error)
	RegisterFailedLogin(ctx context.Context, id string, maxFails int, lockMinutes int) error
	ResetFailedLogins(ctx context.Context, id string) error
}

type userRepository struct {
	db *Database
}

func NewUserRepository(db *Database) UserRepository {
	return &userRepository{db: db}
}

func (r *userRepository) Create(ctx context.Context, u User) (User, error) {
	row := r.db.DB.QueryRowContext(ctx, `
		INSERT INTO users (email, full_name, password_hash, role, active)
		VALUES ($1, $2, $3, $4, TRUE)
		RETURNING id, email, full_name, password_hash, role, active, created_at, updated_at, last_login_at,
			COALESCE(failed_login_count, 0), locked_until
	`, u.Email, u.FullName, u.PasswordHash, u.Role)

	var out User
	err := scanUser(row, &out)
	if err != nil {
		// unique violation
		if isUniqueViolation(err) {
			return User{}, ErrEmailTaken
		}
		return User{}, err
	}
	return out, nil
}

func (r *userRepository) GetByEmail(ctx context.Context, email string) (*User, error) {
	row := r.db.DB.QueryRowContext(ctx, `
		SELECT id, email, full_name, password_hash, role, active, created_at, updated_at, last_login_at,
			COALESCE(failed_login_count, 0), locked_until
		FROM users WHERE email = $1
	`, email)
	var u User
	err := scanUser(row, &u)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *userRepository) GetByID(ctx context.Context, id string) (*User, error) {
	row := r.db.DB.QueryRowContext(ctx, `
		SELECT id, email, full_name, password_hash, role, active, created_at, updated_at, last_login_at,
			COALESCE(failed_login_count, 0), locked_until
		FROM users WHERE id = $1
	`, id)
	var u User
	err := scanUser(row, &u)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *userRepository) List(ctx context.Context, limit int) ([]User, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	rows, err := r.db.DB.QueryContext(ctx, `
		SELECT id, email, full_name, password_hash, role, active, created_at, updated_at, last_login_at,
			COALESCE(failed_login_count, 0), locked_until
		FROM users
		ORDER BY created_at DESC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []User
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.Email, &u.FullName, &u.PasswordHash, &u.Role, &u.Active, &u.CreatedAt, &u.UpdatedAt, &u.LastLoginAt, &u.FailedLoginCount, &u.LockedUntil); err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

func (r *userRepository) UpdateActive(ctx context.Context, id string, active bool) error {
	res, err := r.db.DB.ExecContext(ctx, `
		UPDATE users SET active = $2, updated_at = NOW() WHERE id = $1
	`, id, active)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrUserNotFound
	}
	return nil
}

func (r *userRepository) UpdateRole(ctx context.Context, id string, role string) error {
	res, err := r.db.DB.ExecContext(ctx, `
		UPDATE users SET role = $2, updated_at = NOW() WHERE id = $1
	`, id, role)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrUserNotFound
	}
	return nil
}

func (r *userRepository) UpdatePassword(ctx context.Context, id, passwordHash string) error {
	res, err := r.db.DB.ExecContext(ctx, `
		UPDATE users SET password_hash = $2, updated_at = NOW() WHERE id = $1
	`, id, passwordHash)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrUserNotFound
	}
	return nil
}

func (r *userRepository) TouchLastLogin(ctx context.Context, id string) error {
	_, err := r.db.DB.ExecContext(ctx, `
		UPDATE users SET last_login_at = NOW() WHERE id = $1
	`, id)
	return err
}

func (r *userRepository) Count(ctx context.Context) (int, error) {
	var n int
	err := r.db.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM users`).Scan(&n)
	return n, err
}


func (r *userRepository) RegisterFailedLogin(ctx context.Context, id string, maxFails int, lockMinutes int) error {
	if maxFails <= 0 {
		maxFails = 5
	}
	if lockMinutes <= 0 {
		lockMinutes = 15
	}
	_, err := r.db.DB.ExecContext(ctx, `
		UPDATE users SET
			failed_login_count = failed_login_count + 1,
			locked_until = CASE
				WHEN failed_login_count + 1 >= $2 THEN NOW() + make_interval(mins => $3)
				ELSE locked_until
			END,
			updated_at = NOW()
		WHERE id = $1
	`, id, maxFails, lockMinutes)
	return err
}

func (r *userRepository) ResetFailedLogins(ctx context.Context, id string) error {
	_, err := r.db.DB.ExecContext(ctx, `
		UPDATE users SET failed_login_count = 0, locked_until = NULL, updated_at = NOW()
		WHERE id = $1
	`, id)
	return err
}

func scanUser(row *sql.Row, u *User) error {
	return row.Scan(&u.ID, &u.Email, &u.FullName, &u.PasswordHash, &u.Role, &u.Active, &u.CreatedAt, &u.UpdatedAt, &u.LastLoginAt, &u.FailedLoginCount, &u.LockedUntil)
}

func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	// lib/pq: "duplicate key value violates unique constraint"
	return contains(err.Error(), "duplicate key") || contains(err.Error(), "unique constraint")
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 ||
		func() bool {
			for i := 0; i+len(sub) <= len(s); i++ {
				if s[i:i+len(sub)] == sub {
					return true
				}
			}
			return false
		}())
}
