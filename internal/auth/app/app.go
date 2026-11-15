package app

import (
	api "SaaS_Pipeline/internal/auth/api"
	"SaaS_Pipeline/internal/auth/handler/http/middleware"
	"SaaS_Pipeline/internal/auth/handler/http/v2"
	"SaaS_Pipeline/internal/auth/infrastructure/repository/postgres"
	"SaaS_Pipeline/internal/auth/infrastructure/security"
	"SaaS_Pipeline/internal/auth/usecase"
	"SaaS_Pipeline/pkg/config"
	"SaaS_Pipeline/pkg/logger"
	"SaaS_Pipeline/pkg/middlewares/logging"
	pgx "SaaS_Pipeline/pkg/postgres"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	RefreshTokenTTL = time.Hour * 24 * 15
	AccessTokenTTL  = time.Minute * 15
)

type App struct {
	log    *slog.Logger
	server *http.Server
	db     *pgxpool.Pool
}

func New(ctx context.Context) (*App, error) {
	if err := config.LoadDotEnv(".env"); err != nil {
		return nil, fmt.Errorf("app.New: %w", err)
	}
	env := config.Get("AUTH_APP_ENV", "local")
	dsn := config.MustGet("DATABASE_URL")
	port := config.Get("AUTH_APP_PORT", "8080")
	jwtSecret := config.MustGet("JWT_SECRET")
	logger.Setup(env)
	log := logger.With("service", "auth")
	log.Info("starting app auth", "env", env)
	pool, err := pgx.NewPool(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("app.New: %w", err)
	}
	userRepo := postgres.NewUserRepository(pool)
	tokenRepo := postgres.NewRefreshTokenRepository(pool)
	jwtManager := security.NewJWTManager(jwtSecret, RefreshTokenTTL, AccessTokenTTL)
	passwordManager := security.NewPasswordManager()

	authUseCase := usecase.NewAuthUseCase(userRepo, tokenRepo, jwtManager, passwordManager, log)

	authHandler := v2.NewAuthHandler(authUseCase, log)

	strictHandler := api.NewStrictHandler(authHandler, nil)
	defaultMux := http.NewServeMux()
	api.HandlerFromMux(strictHandler, defaultMux)

	mux := logging.LoggingMiddleware(log)(defaultMux)
	//mux = requestId.RequestIDMiddleware(mux)
	mux = middleware.XRealIP(mux)
	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 20,
		BaseContext:       func(_ net.Listener) context.Context { return ctx },
	}

	go func() {
		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				cleanCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				if err := tokenRepo.DeleteExpiredRefreshTokens(cleanCtx); err != nil {
					log.Error("failed to delete expired tokens", slog.Any("error", err))
				} else {
					log.Info("expired tokens deleted successfully")
				}
				cancel()
			case <-ctx.Done():
				log.Info("stopping background token cleaner")
				return
			}
		}
	}()

	return &App{
		log:    log,
		server: srv,
		db:     pool,
	}, nil
}

func (a *App) Run(ctx context.Context) error {
	errCh := make(chan error, 1)

	go func() {
		a.log.Info("starting http server")
		a.log.Info("HTTP server is running", slog.String("addr", a.server.Addr))
		if err := a.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	select {
	case err := <-errCh:
		return fmt.Errorf("server failed: %w", err)
	case <-ctx.Done():
		a.log.Info("shutting down gracefully...")
	}
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := a.server.Shutdown(shutdownCtx); err != nil {
		a.log.Error("server forced to shutdown", slog.Any("error", err))
	}

	a.log.Info("closing database connections...")

	defer a.db.Close()

	a.log.Info("app stopped successfully")
	return nil
}
