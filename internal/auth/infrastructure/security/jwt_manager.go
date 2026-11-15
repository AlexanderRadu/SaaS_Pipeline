package security

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type JWTManager struct {
	secretKey       string
	refreshTokenTTL time.Duration
	accessTokenTTL  time.Duration
}

func NewJWTManager(secretKey string, refreshTokenTTL time.Duration, accessTokenTTL time.Duration) *JWTManager {
	return &JWTManager{secretKey: secretKey, refreshTokenTTL: refreshTokenTTL, accessTokenTTL: accessTokenTTL}
}

func (j *JWTManager) GenerateAccessToken(userID int64) (string, error) {
	claims := jwt.MapClaims{
		"user_id": userID,
		"exp":     time.Now().Add(j.accessTokenTTL).Unix(),
		"iat":     time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(j.secretKey))
	if err != nil {
		return "", err
	}
	return tokenString, nil
}

func (j *JWTManager) ValidateAccessToken(token string) (int64, error) {
	tokenStr, err := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(j.secretKey), nil
	})
	if err != nil || !tokenStr.Valid {
		return 0, fmt.Errorf("invalid token: %w", err)
	}
	claims, ok := tokenStr.Claims.(jwt.MapClaims)
	if !ok {
		return 0, fmt.Errorf("invalid token claims")
	}
	userIDFloat, ok := claims["user_id"].(float64)
	if !ok {
		return 0, fmt.Errorf("invalid user_id in claims")
	}
	return int64(userIDFloat), nil
}

func (j *JWTManager) GenerateRefreshToken() (string, error) {
	b := make([]byte, 32)
	_, err := rand.Read(b)
	if err != nil {
		return "", fmt.Errorf("failed to generate refresh token: %w", err)
	}
	return hex.EncodeToString(b), nil
}

func (j *JWTManager) GetRefreshTokenTTL() time.Duration {
	return j.refreshTokenTTL
}

func (j *JWTManager) HashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}
