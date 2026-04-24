package auth

import (
	api "SaaS_Pipeline/internal/auth/api"
	"context"
	"errors"
	"log"
	"log/slog"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"golang.org/x/crypto/bcrypt"
)

type AuthService struct {
	repo      *Repository
	jwtSecret []byte
	log       *slog.Logger
}

func NewAuthService(repo *Repository, secret []byte, log *slog.Logger) *AuthService {
	return &AuthService{repo: repo, jwtSecret: secret, log: log}
}

func (s *AuthService) RegisterUser(ctx context.Context, request api.RegisterUserRequestObject) (api.RegisterUserResponseObject, error) {
	email := string(request.Body.Email)
	password := string(request.Body.Password)

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("Bcrypt error: %v", err)
		return api.RegisterUser400Response{}, nil
	}
	err = s.repo.CreateUser(ctx, email, string(hash))
	if err != nil {
		if errors.Is(err, ErrUserExists) {
			return api.RegisterUser409Response{}, nil
		}
		log.Printf("DB error: %v", err)
		return nil, err
	}
	return api.RegisterUser201Response{}, nil
}

func (s *AuthService) LoginUser(ctx context.Context, request api.LoginUserRequestObject) (api.LoginUserResponseObject, error) {
	email := string(request.Body.Email)
	password := string(request.Body.Password)

	user, err := s.repo.GetUserByEmail(ctx, email)

	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return api.LoginUser401Response{}, nil
		}
		log.Printf("DB error: %v", err)
		return nil, err
	}
	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password))
	if err != nil {
		return api.LoginUser401Response{}, nil
	}

	tokenString, err := s.generateJWT(user.ID, user.Email)
	if err != nil {
		log.Printf("JWT generation error: %v", err)
		return nil, err
	}

	return api.LoginUser200JSONResponse{
		Token: &tokenString,
	}, nil
}

func (s *AuthService) generateJWT(userID int64, email string) (string, error) {
	claims := jwt.MapClaims{
		"user_id": userID,
		"email":   email,
		"exp":     time.Now().Add(time.Hour * 72).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.jwtSecret)
}
