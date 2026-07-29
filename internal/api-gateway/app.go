package api_gateway

import (
	"SaaS_Pipeline/pkg/config"
	"SaaS_Pipeline/pkg/logger"
	"SaaS_Pipeline/pkg/middlewares/bodylimit"
	"SaaS_Pipeline/pkg/middlewares/ratelimit"
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"time"

	"golang.org/x/time/rate"
)

type App struct {
	log *slog.Logger
	srv *http.Server
}

func New(ctx context.Context) (*App, error) {
	env := config.Get("APP_ENV", "local")
	port := config.Get("GATEWAY_PORT", "8080")
	jwtSecret := config.MustGet("JWT_SECRET")
	authServiceURL := config.Get("AUTH_SERVICE_URL", "http://localhost:8081")
	deployServiceURL := config.Get("DEPLOY_SERVICE_URL", "http://localhost:8082")
	statusServiceURL := config.Get("STATUS_SERVICE_URL", "http://localhost:8083")
	catalogServiceURL := config.Get("CATALOG_SERVICE_URL", "http://localhost:8084")

	logger.Setup(env)
	log := logger.With("service", "api-gateway")

	mux := http.NewServeMux()

	//registerLimiter := ratelimit.RateLimit(rate.Limit(5.0/60.0), 5, 20*time.Minute, 5*time.Minute, ratelimit.IPKeyFn)
	//loginLimiter := ratelimit.RateLimit(rate.Limit(10.0/60.0), 5, 20*time.Minute, 5*time.Minute, ratelimit.IPKeyFn)

	authProxy, err := createProxy("AuthService", authServiceURL)

	if err != nil {
		return nil, err
	}

	mux.Handle("/api/v2/auth/register", authProxy)
	mux.Handle("/api/v2/auth/login", authProxy)
	mux.Handle("/api/v2/auth", authProxy)

	authLimiter := ratelimit.RateLimit(rate.Limit(1000.0/60.0), 50, 20*time.Minute, 5*time.Minute, ratelimit.UserKeyFn)

	secureRoute := func(next http.Handler) http.Handler {
		return AuthMiddleware(jwtSecret, authLimiter(next))
	}

	catalogProxy, err := createProxy("CatalogService", catalogServiceURL)
	if err != nil {
		return nil, err
	}
	mux.Handle("/api/v2/catalog", secureRoute(catalogProxy))

	deployProxy, err := createProxy("DeployService", deployServiceURL)
	if err != nil {
		return nil, err
	}
	mux.Handle("/api/v2/service_instances", secureRoute(deployProxy))

	statusProxy, err := createProxy("StatusService", statusServiceURL)
	if err != nil {
		return nil, err
	}
	mux.Handle("/api/v2/instances", secureRoute(statusProxy))

	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))

	})

	mux.HandleFunc("GET /readyz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ready"}`))
	})

	var limit int64 = 1024 * 1024 * 5
	handler := bodylimit.LimitBody(limit)(RequestIDMiddleware(mux))

	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 20,
		BaseContext:       func(_ net.Listener) context.Context { return ctx },
	}
	return &App{
		log: log,
		srv: srv,
	}, nil
}

func (a *App) Run(ctx context.Context) error {
	errCh := make(chan error, 1)

	go func() {
		a.log.Info("API Gateway is running", slog.String("addr", a.srv.Addr))
		if err := a.srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	select {
	case <-ctx.Done():
		a.log.Info("shutting down gracefully...")
	case err := <-errCh:
		return err
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return a.srv.Shutdown(shutdownCtx)
}
