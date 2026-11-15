package entity

import (
	"errors"
	"time"
)

var (
	ErrRefreshTokenNotFound = errors.New("refresh token not found")
	ErrRefreshTokenExpired  = errors.New("refresh token expired")
)

type RefreshToken struct {
	UserID    int64
	Token     string
	ExpiresAt time.Time
	CreatedAt time.Time
	Revoked   bool
	IpAddress string
}
