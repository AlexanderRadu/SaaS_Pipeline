package status

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"sync"
	"time"

	api "SaaS_Pipeline/internal/status/api"
	"SaaS_Pipeline/pkg/config"
	"SaaS_Pipeline/pkg/logger"
	"SaaS_Pipeline/pkg/postgres"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
)

type contextKey string

const (
	UserIDKey contextKey = "user_id"
)

type App struct {
	log      *slog.Logger
	pool     *pgxpool.Pool
	db       *sql.DB
	srv      *http.Server
	consumer *Consumer
}

func New(ctx context.Context) (*App, error) {
	if err := config.LoadDotEnv(".env"); err != nil {
		return nil, fmt.Errorf("app.New: load env: %w", err)
	}

	env := config.Get("STATUS_APP_ENV", "local")
	dsn := config.MustGet("DATABASE_URL")
	port := config.Get("STATUS_APP_PORT", "5050")
	amqpURL := config.MustGet("RABBITMQ_URL")

	logger.Setup(env)
	log := logger.With("service", "status")
	log.Info("starting app status", "env", env)

	pool, err := postgres.NewPool(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("app.New: init postgres pool: %w", err)
	}

	sqlDB := stdlib.OpenDBFromPool(pool)

	repo := NewRepository(sqlDB)
	service := NewStatusService(repo, log)

	consumer, err := NewConsumer(amqpURL, repo, log)
	if err != nil {
		return nil, fmt.Errorf("app.New: failed to init rabbitmq: %w", err)
	}

	strictHandler := api.NewStrictHandler(service, nil)
	mux := http.NewServeMux()
	api.HandlerFromMux(strictHandler, mux)

	httpHandler := authMiddleware(mux)

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: httpHandler,
	}

	return &App{
		log:      log,
		pool:     pool,
		db:       sqlDB,
		srv:      srv,
		consumer: consumer,
	}, nil
}

func authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userIDStr := r.Header.Get("X-User-Id")
		if userIDStr == "" {
			http.Error(w, "Unauthorized: missing X-User-Id", http.StatusUnauthorized)
			return
		}

		userID, err := strconv.ParseInt(userIDStr, 10, 64)
		if err != nil {
			http.Error(w, "Bad Request: invalid X-User-Id", http.StatusBadRequest)
			return
		}

		ctxWithValue := context.WithValue(r.Context(), UserIDKey, userID)
		next.ServeHTTP(w, r.WithContext(ctxWithValue))
	})
}

func (a *App) Run(ctx context.Context) error {
	var wg sync.WaitGroup
	errCh := make(chan error, 1)

	workerCtx, cancelWorker := context.WithCancel(ctx)
	defer cancelWorker()

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := a.consumer.Start(workerCtx); err != nil {
			a.log.Error("consumer stopped with error", slog.Any("error", err))
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		a.log.Info("HTTP server is running", slog.String("addr", a.srv.Addr))
		if err := a.srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- fmt.Errorf("server failed: %w", err)
		}
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		a.log.Info("shutting down gracefully...")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := a.srv.Shutdown(shutdownCtx); err != nil {
		a.log.Error("server forced to shutdown", slog.Any("error", err))
	}

	cancelWorker()
	a.consumer.Close(shutdownCtx)

	wg.Wait()

	a.log.Info("closing database connections...")
	if err := a.db.Close(); err != nil {
		a.log.Error("failed to close sql.DB", slog.Any("error", err))
	}
	a.pool.Close()

	a.log.Info("app stopped successfully")
	return nil
}
