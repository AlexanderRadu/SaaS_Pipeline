package api_gateway

import (
	"SaaS_Pipeline/pkg/bodylimit"
	"SaaS_Pipeline/pkg/config"
	"SaaS_Pipeline/pkg/logger"
	"SaaS_Pipeline/pkg/ratelimit"
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
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

	registerLimiter := ratelimit.RateLimit(rate.Limit(5.0/60.0), 5, ratelimit.IPKeyFn)

	loginLimiter := ratelimit.RateLimit(rate.Limit(10.0/60.0), 5, ratelimit.IPKeyFn)

	authProxy := createProxy("AuthService", authServiceURL)

	mux.Handle("/api/v1/auth/register", registerLimiter(authProxy))
	mux.Handle("/api/v1/auth/login", loginLimiter(authProxy))
	mux.Handle("/api/v1/auth", authProxy)

	authLimiter := ratelimit.RateLimit(rate.Limit(1000.0/60.0), 50, ratelimit.UserKeyFn)

	secureRoute := func(next http.Handler) http.Handler {
		return AuthMiddleware(jwtSecret, authLimiter(next))
	}

	catalogProxy := createProxy("CatalogService", catalogServiceURL)
	mux.Handle("/v1/catalog", secureRoute(catalogProxy))

	deployProxy := createProxy("DeployService", deployServiceURL)
	mux.Handle("/v1/service_instances/", secureRoute(deployProxy))

	statusProxy := createProxy("StatusService", statusServiceURL)
	mux.Handle("/api/v1/instances", secureRoute(statusProxy))


	var limit16kB int64 = 1024 * 16
	limitedHandler := bodylimit.LimitBody(limit16kB)(mux)

	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           limitedHandler,
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

func createProxy(name string, target string) *httputil.ReverseProxy {
	targetURL, err := url.Parse(target)
	if err != nil {
		panic("invalid target url: " + target)
	}

	proxy := httputil.NewSingleHostReverseProxy(targetURL)

	proxy.Transport = &ResilientTransport{
		Next:       http.DefaultTransport,
		CB:         NewCircuitBreaker(name),
		MaxRetries: 2,
	}

	return proxy
}
