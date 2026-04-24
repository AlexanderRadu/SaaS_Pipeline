package api_gateway

import (
	"SaaS_Pipeline/pkg/config"
	"SaaS_Pipeline/pkg/logger"
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"
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

	mux.Handle("/api/v1/auth/", createProxy("AuthService", authServiceURL))

	mux.Handle("/v1/catalog", AuthMiddleware(jwtSecret, createProxy("CatalogService", catalogServiceURL)))
	mux.Handle("/v1/service_instances/", AuthMiddleware(jwtSecret, createProxy("DeployService", deployServiceURL)))
	mux.Handle("/api/v1/instances", AuthMiddleware(jwtSecret, createProxy("StatusService", statusServiceURL)))

	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
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
