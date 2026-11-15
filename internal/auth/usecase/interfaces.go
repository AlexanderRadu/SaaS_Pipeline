package usecase

import (
	"SaaS_Pipeline/internal/auth/entity"
	"context"
	"time"
)

type UserRepository interface {
	CreateUser(ctx context.Context, user *entity.User) error
	GetUserByEmail(ctx context.Context, email string) (*entity.User, error)
}

type RefreshTokenRepository interface {
	SaveRefreshToken(ctx context.Context, userID int64, tokenHash string, expiresAt time.Time, ipAddress string) error
	GetRefreshToken(ctx context.Context, token string) (*entity.RefreshToken, error)
	RevokeRefreshToken(ctx context.Context, token string) error
	RevokeAllRefreshTokens(ctx context.Context, userID int64) error
	RevokeRefreshTokensByIP(ctx context.Context, ipAddress string) error
	DeleteExpiredRefreshTokens(ctx context.Context) error
}

type PasswordManager interface {
	Hash(password string) (string, error)
	ValidatePassword(password string, hashedPassword string) bool
}

type JWTManager interface {
	GenerateAccessToken(userID int64) (string, error)
	GenerateRefreshToken() (string, error)
	ValidateAccessToken(token string) (int64, error)
	GetRefreshTokenTTL() time.Duration
	HashToken(token string) string
}
