package v2

import (
	api "SaaS_Pipeline/internal/auth/api"
	"SaaS_Pipeline/internal/auth/entity"
	"SaaS_Pipeline/internal/auth/usecase"
	"context"
	"errors"
	"log/slog"
)

type AuthHandler struct {
	authUseCase *usecase.AuthUseCase
	log         *slog.Logger
}

func NewAuthHandler(authUseCase *usecase.AuthUseCase, log *slog.Logger) *AuthHandler {
	return &AuthHandler{authUseCase: authUseCase, log: log}
}
func (a *AuthHandler) LoginUser(ctx context.Context, req api.LoginUserRequestObject) (api.LoginUserResponseObject, error) {
	clientIP := ctx.Value("X-Real-IP").(string)

	accessToken, refreshToken, err := a.authUseCase.Login(ctx, string(req.Body.Email), req.Body.Password, clientIP)
	if err != nil {
		if errors.Is(err, entity.ErrUserNotFound) || errors.Is(err, entity.ErrWrongPassword) {
			return api.LoginUser401Response{}, nil
		}
		a.log.ErrorContext(ctx, "failed to login user",
			slog.String("email", string(req.Body.Email)),
			slog.String("error", err.Error()),
		)
		return api.LoginUser500Response{}, nil
	}
	return api.LoginUser200JSONResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil

}

func (a *AuthHandler) RegisterUser(ctx context.Context, req api.RegisterUserRequestObject) (api.RegisterUserResponseObject, error) {
	email := string(req.Body.Email)
	password := req.Body.Password

	if email == "" || password == "" {
		return api.RegisterUser400Response{}, nil
	}

	err := a.authUseCase.Register(ctx, email, password)
	if err != nil {
		if errors.Is(err, entity.ErrUserAlreadyExists) {
			return api.RegisterUser409Response{}, nil
		}
		a.log.ErrorContext(ctx, "failed to register user",
			slog.String("email", string(req.Body.Email)),
			slog.String("error", err.Error()),
		)
		return api.RegisterUser500Response{}, nil
	}

	return api.RegisterUser201Response{}, nil
}

func (a *AuthHandler) UpdateRefreshToken(ctx context.Context, req api.UpdateRefreshTokenRequestObject) (api.UpdateRefreshTokenResponseObject, error) {
	token := req.Body.RefreshToken
	clientIP := ctx.Value("X-Real-IP").(string)

	accessToken, refreshToken, err := a.authUseCase.UpdateRefreshToken(ctx, token, clientIP)
	if err != nil {
		if errors.Is(err, entity.ErrRefreshTokenNotFound) || errors.Is(err, entity.ErrRefreshTokenExpired) {
			return api.UpdateRefreshToken401Response{}, nil
		}
		a.log.ErrorContext(ctx, "failed to refresh token",
			slog.String("error", err.Error()),
		)
		return api.UpdateRefreshToken500Response{}, nil
	}
	return api.UpdateRefreshToken200JSONResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}
