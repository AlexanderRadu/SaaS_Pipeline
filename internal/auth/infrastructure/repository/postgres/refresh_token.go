package postgres

import (
	"SaaS_Pipeline/internal/auth/entity"
	"context"
	"crypto/sha256"
	"encoding/hex"

	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type RefreshTokenRepository struct {
	db *pgxpool.Pool
}

func NewRefreshTokenRepository(db *pgxpool.Pool) *RefreshTokenRepository {
	return &RefreshTokenRepository{db: db}
}

func (r *RefreshTokenRepository) SaveRefreshToken(ctx context.Context, userID int64, tokenHash string, expiresAt time.Time, ipAddress string) error {
	query := `INSERT INTO refresh_tokens (user_id, token_hash, expires_at, ip_address) VALUES ($1, $2, $3, $4)`

	_, err := r.db.Exec(ctx, query, userID, tokenHash, expiresAt, ipAddress)
	if err != nil {
		return err
	}
	return nil
}

func (r *RefreshTokenRepository) GetRefreshToken(ctx context.Context, token string) (*entity.RefreshToken, error) {
	query := `SELECT user_id, expires_at, created_at, revoked, ip_address FROM refresh_tokens WHERE token_hash = $1`

	tokenHash := r.hashToken(token)

	var refreshToken entity.RefreshToken

	err := r.db.QueryRow(ctx, query, tokenHash).Scan(&refreshToken.UserID, &refreshToken.ExpiresAt, &refreshToken.CreatedAt, &refreshToken.Revoked, &refreshToken.IpAddress)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, entity.ErrRefreshTokenNotFound
		}
		return nil, err
	}
	refreshToken.Token = token
	return &refreshToken, nil

}

func (r *RefreshTokenRepository) RevokeRefreshToken(ctx context.Context, token string) error {
	query := `UPDATE refresh_tokens SET revoked = true WHERE token_hash = $1`
	tokenHash := r.hashToken(token)
	_, err := r.db.Exec(ctx, query, tokenHash)

	if err != nil {
		return err
	}

	return nil
}

func (r *RefreshTokenRepository) RevokeAllRefreshTokens(ctx context.Context, userID int64) error {
	query := `UPDATE refresh_tokens SET revoked = true WHERE user_id = $1`

	_, err := r.db.Exec(ctx, query, userID)
	if err != nil {
		return err
	}
	return nil
}

func (r *RefreshTokenRepository) hashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}

func (r *RefreshTokenRepository) RevokeRefreshTokensByIP(ctx context.Context, ipAddress string) error {
	query := `UPDATE refresh_tokens SET revoked = true WHERE ip_address = $1`
	_, err := r.db.Exec(ctx, query, ipAddress)
	return err
}

func (r *RefreshTokenRepository) DeleteExpiredRefreshTokens(ctx context.Context) error {
	query := `DELETE FROM refresh_tokens WHERE expires_at < NOW() OR (revoked = true AND created_at < NOW() - INTERVAL '3 days')`
	_, err := r.db.Exec(ctx, query)
	return err
}
