package postgres

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

var ErrRefreshNotFound = errors.New("refresh token inválido ou revogado")

type RefreshToken struct {
	ID        string
	UserID    string
	TokenHash string
	ExpiresAt time.Time
	RevokedAt sql.NullTime
	CreatedAt time.Time
}

type RefreshTokenRepository interface {
	Create(ctx context.Context, userID, tokenHash string, expiresAt time.Time, ip, ua string) error
	GetValid(ctx context.Context, tokenHash string) (*RefreshToken, error)
	Revoke(ctx context.Context, tokenHash string) error
	RevokeAllForUser(ctx context.Context, userID string) error
}

type refreshTokenRepository struct {
	db *Database
}

func NewRefreshTokenRepository(db *Database) RefreshTokenRepository {
	return &refreshTokenRepository{db: db}
}

func (r *refreshTokenRepository) Create(ctx context.Context, userID, tokenHash string, expiresAt time.Time, ip, ua string) error {
	_, err := r.db.DB.ExecContext(ctx, `
		INSERT INTO refresh_tokens (user_id, token_hash, expires_at, ip, user_agent)
		VALUES ($1, $2, $3, $4, $5)
	`, userID, tokenHash, expiresAt, ip, ua)
	return err
}

func (r *refreshTokenRepository) GetValid(ctx context.Context, tokenHash string) (*RefreshToken, error) {
	row := r.db.DB.QueryRowContext(ctx, `
		SELECT id, user_id, token_hash, expires_at, revoked_at, created_at
		FROM refresh_tokens
		WHERE token_hash = $1 AND revoked_at IS NULL AND expires_at > NOW()
	`, tokenHash)
	var t RefreshToken
	err := row.Scan(&t.ID, &t.UserID, &t.TokenHash, &t.ExpiresAt, &t.RevokedAt, &t.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrRefreshNotFound
	}
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *refreshTokenRepository) Revoke(ctx context.Context, tokenHash string) error {
	_, err := r.db.DB.ExecContext(ctx, `
		UPDATE refresh_tokens SET revoked_at = NOW() WHERE token_hash = $1 AND revoked_at IS NULL
	`, tokenHash)
	return err
}

func (r *refreshTokenRepository) RevokeAllForUser(ctx context.Context, userID string) error {
	_, err := r.db.DB.ExecContext(ctx, `
		UPDATE refresh_tokens SET revoked_at = NOW()
		WHERE user_id = $1 AND revoked_at IS NULL
	`, userID)
	return err
}
