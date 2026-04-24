package deploy

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/moby/moby/client"

	api "SaaS_Pipeline/internal/deploy/api"
	"SaaS_Pipeline/pkg/config"
	"SaaS_Pipeline/pkg/logger"
	"SaaS_Pipeline/pkg/rabbitmq"
)

type contextKey string

const UserIDKey contextKey = "user_id"

type App struct {
	log       *slog.Logger
	srv       *http.Server
	publisher *rabbitmq.Publisher
	docker    *client.Client
}

type httpCatalogClient struct {
	baseURL string
	client  *http.Client
}

func NewHTTPCatalogClient(baseURL string) *httpCatalogClient {
	return &httpCatalogClient{
		baseURL: baseURL,
		client: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

func (c *httpCatalogClient) GetServiceInfo(ctx context.Context, serviceID string) (*CatalogTemplate, error) {
	catalogURL := fmt.Sprintf("%s/internal/catalog/%s", c.baseURL, serviceID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, catalogURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get template from catalog: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("catalog returned bad status code: %d", resp.StatusCode)
	}

	var template CatalogTemplate
	if err := json.NewDecoder(resp.Body).Decode(&template); err != nil {
		return nil, fmt.Errorf("failed to decode catalog response: %w", err)
	}
	return &template, nil
}

func New(ctx context.Context) (*App, error) {
	if err := config.LoadDotEnv(".env"); err != nil {
		return nil, fmt.Errorf("app.New: %w", err)
	}

	env := config.Get("DEPLOY_APP_ENV", "local")
	port := config.Get("DEPLOY_APP_PORT", "3030")
	amqpURL := config.MustGet("RABBITMQ_URL")

	catalogURL := config.Get("CATALOG_SERVICE_URL", "http://catalog-service:8080")

	logger.Setup(env)
	log := logger.With("service", "deploy")
	log.Info("starting app deploy", "env", env)

	dockerClient, err := client.New(client.FromEnv)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize docker client: %w", err)
	}

	rmqPublisher, err := rabbitmq.NewPublisher(amqpURL)
	if err != nil {
		_ = dockerClient.Close()
		return nil, fmt.Errorf("failed to initialize rabbitmq publisher: %w", err)
	}

	catalogClient := NewHTTPCatalogClient(catalogURL)

	userIDMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userIDStr := r.Header.Get("X-User-Id")
			ctxWithValue := context.WithValue(r.Context(), UserIDKey, userIDStr)
			next.ServeHTTP(w, r.WithContext(ctxWithValue))
		})
	}

	deployService := NewDeployService(log, dockerClient, rmqPublisher, catalogClient)

	strictHandler := api.NewStrictHandler(deployService, nil)
	mux := http.NewServeMux()
	api.HandlerFromMux(strictHandler, mux)

	handlerWithMiddleware := userIDMiddleware(mux)

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: handlerWithMiddleware,
	}

	return &App{
		log:       log,
		srv:       srv,
		publisher: rmqPublisher,
		docker:    dockerClient,
	}, nil
}

func (a *App) Run(ctx context.Context) error {
	errCh := make(chan error, 1)

	go func() {
		a.log.Info("HTTP server is running", slog.String("addr", a.srv.Addr))
		if err := a.srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	select {
	case err := <-errCh:
		return fmt.Errorf("server failed: %w", err)
	case <-ctx.Done():
		a.log.Info("shutting down gracefully...")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := a.srv.Shutdown(shutdownCtx); err != nil {
		a.log.Error("server forced to shutdown", slog.Any("error", err))
	}

	a.log.Info("closing RabbitMQ connection...")
	if err := a.publisher.Close(); err != nil {
		a.log.Error("failed to close RabbitMQ connection", slog.Any("error", err))
	}

	a.log.Info("closing Docker client...")
	if err := a.docker.Close(); err != nil {
		a.log.Error("failed to close Docker client", slog.Any("error", err))
	}

	a.log.Info("app stopped successfully")
	return nil
}
