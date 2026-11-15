package usecase

import (
	"SaaS_Pipeline/internal/auth/entity"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"
)

type AuthUseCase struct {
	userRepo        UserRepository
	tokenRepo       RefreshTokenRepository
	jwtManager      JWTManager
	passwordManager PasswordManager
	log             *slog.Logger
}

func NewAuthUseCase(ur UserRepository, tr RefreshTokenRepository, jm JWTManager, pm PasswordManager, log *slog.Logger) *AuthUseCase {
	return &AuthUseCase{
		userRepo:        ur,
		tokenRepo:       tr,
		jwtManager:      jm,
		passwordManager: pm,
		log:             log,
	}
}

func (a *AuthUseCase) Register(ctx context.Context, email, password string) error {
	existingUser, err := a.userRepo.GetUserByEmail(ctx, email)
	if err == nil && existingUser != nil {
		return entity.ErrUserAlreadyExists
	}

	passwordHash, err := a.passwordManager.Hash(password)
	if err != nil {
		return fmt.Errorf("passwordManager.Hash: %w", err)
	}
	err = a.userRepo.CreateUser(ctx, &entity.User{Email: email, PasswordHash: passwordHash})
	if err != nil {
		return fmt.Errorf("userRepo.CreateUser: %w", err)
	}
	a.log.InfoContext(ctx, "user registered successfully", slog.String("email", email))
	return nil
}

func (a *AuthUseCase) Login(ctx context.Context, email, password string, clientIP string) (string, string, error) {
	user, err := a.userRepo.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, entity.ErrUserNotFound) {
			a.log.WarnContext(ctx, "login failed: user not found",
				slog.String("email", email),
				slog.String("clientIP", clientIP),
			)
			return "", "", entity.ErrUserNotFound
		}
		return "", "", fmt.Errorf("userRepo.GetUserByEmail: %w", err)
	}
	if !a.passwordManager.ValidatePassword(password, user.PasswordHash) {
		a.log.WarnContext(ctx, "login failed: invalid password",
			slog.String("email", email),
			slog.String("clientIP", clientIP),
		)
		return "", "", entity.ErrWrongPassword
	}

	if err := a.tokenRepo.RevokeRefreshTokensByIP(ctx, clientIP); err != nil {
		return "", "", fmt.Errorf("tokenRepo.RevokeRefreshTokensByIP: %w", err)
	}

	accessToken, err := a.jwtManager.GenerateAccessToken(user.ID)
	if err != nil {
		return "", "", fmt.Errorf("jwtManager.GenerateAccessToken: %w", err)
	}

	refreshToken, err := a.jwtManager.GenerateRefreshToken()

	if err != nil {
		return "", "", fmt.Errorf("jwtManager.GenerateRefreshToken: %w", err)
	}
	expiresAt := time.Now().Add(a.jwtManager.GetRefreshTokenTTL())
	tokenHash := a.jwtManager.HashToken(refreshToken)
	if err := a.tokenRepo.SaveRefreshToken(ctx, user.ID, tokenHash, expiresAt, clientIP); err != nil {
		return "", "", fmt.Errorf("tokenRepo.SaveRefreshToken: %w", err)
	}
	a.log.InfoContext(ctx, "user logged in",
		slog.Int64("user_id", user.ID),
		slog.String("ip", clientIP),
	)
	return accessToken, refreshToken, nil
}

func (a *AuthUseCase) UpdateRefreshToken(ctx context.Context, refreshToken, clientIP string) (string, string, error) {
	refToken, err := a.tokenRepo.GetRefreshToken(ctx, refreshToken)

	if err != nil {
		if errors.Is(err, entity.ErrRefreshTokenNotFound) {
			return "", "", entity.ErrRefreshTokenNotFound
		}
		return "", "", fmt.Errorf("tokenRepo.GetRefreshToken: %w", err)
	}

	if refToken.Revoked == true {
		a.log.WarnContext(ctx, "security alert: attempt to use revoked refresh token",
			slog.Int64("user_id", refToken.UserID),
			slog.String("clientIP", clientIP),
		)
		return "", "", entity.ErrRefreshTokenExpired
	}

	if refToken.ExpiresAt.Before(time.Now()) {
		return "", "", entity.ErrRefreshTokenExpired
	}

	userID := refToken.UserID

	err = a.tokenRepo.RevokeRefreshTokensByIP(ctx, clientIP)
	if err != nil {
		if errors.Is(err, entity.ErrRefreshTokenNotFound) {
			return "", "", entity.ErrRefreshTokenNotFound
		}
		return "", "", fmt.Errorf("tokenRepo.RevokeRefreshTokensByIP: %w", err)
	}
	accessToken, err := a.jwtManager.GenerateAccessToken(userID)
	if err != nil {
		return "", "", fmt.Errorf("jwtManager.GenerateAccessToken: %w", err)
	}
	refreshToken, err = a.jwtManager.GenerateRefreshToken()
	if err != nil {
		return "", "", fmt.Errorf("jwtManager.GenerateRefreshToken: %w", err)
	}
	expiresAt := time.Now().Add(a.jwtManager.GetRefreshTokenTTL())
	tokenHash := a.jwtManager.HashToken(refreshToken)
	if err := a.tokenRepo.SaveRefreshToken(ctx, userID, tokenHash, expiresAt, clientIP); err != nil {
		return "", "", fmt.Errorf("tokenRepo.SaveRefreshToken: %w", err)
	}
	return accessToken, refreshToken, nil
}
