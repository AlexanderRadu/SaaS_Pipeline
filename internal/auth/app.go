package auth

import (
	api "SaaS_Pipeline/internal/auth/api"
	"SaaS_Pipeline/pkg/config"
	"SaaS_Pipeline/pkg/logger"
	"SaaS_Pipeline/pkg/postgres"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
)

type App struct {
	log  *slog.Logger
	pool *pgxpool.Pool
	db   *sql.DB
	srv  *http.Server
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

	pool, err := postgres.NewPool(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("app.New: %w", err)
	}

	sqlDB := stdlib.OpenDBFromPool(pool)

	repo := NewRepository(sqlDB)
	service := NewAuthService(repo, []byte(jwtSecret), log)

	strictHandler := api.NewStrictHandler(service, nil)
	mux := http.NewServeMux()
	api.HandlerFromMux(strictHandler, mux)

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

	return &App{log: log, pool: pool, db: sqlDB, srv: srv}, nil
}

func (a *App) Run(ctx context.Context) error {
	errCh := make(chan error, 1)

	go func() {
		a.log.Info("HTTP server is running", slog.String("addr", a.srv.Addr))
		if err := a.srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
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

	if err := a.srv.Shutdown(shutdownCtx); err != nil {
		a.log.Error("server forced to shutdown", slog.Any("error", err))
	}

	a.log.Info("closing database connections...")
	if err := a.db.Close(); err != nil {
		a.log.Error("failed to close sql.DB", slog.Any("error", err))
	}
	a.pool.Close()

	a.log.Info("app stopped successfully")
	return nil

}
