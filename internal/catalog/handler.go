package catalog

import (
	api "SaaS_Pipeline/internal/catalog/api"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/redis/go-redis/v9"
)

const catalogCacheKey = "catalog:all_services"
const catalogCacheTTL = 5 * time.Minute

type CatalogService struct {
	repo  *Repository
	log   *slog.Logger
	redis *redis.Client
}

func NewCatalogService(repo *Repository, log *slog.Logger, rdb *redis.Client) *CatalogService {
	return &CatalogService{
		repo:  repo,
		log:   log,
		redis: rdb,
	}
}

func (c *CatalogService) GetCatalog(ctx context.Context, request api.GetCatalogRequestObject) (api.GetCatalogResponseObject, error) {

	cachedData, err := c.redis.Get(ctx, catalogCacheKey).Result()

	if err == nil {
		var apiServices []api.Service
		if err := json.Unmarshal([]byte(cachedData), &apiServices); err == nil {
			c.log.Info("catalog served from redis cache")
			return api.GetCatalog200JSONResponse{
				Services: &apiServices,
			}, nil
		}
		c.log.Error("failed to unmarshal cache data", slog.Any("error", err))
	} else if !errors.Is(err, redis.Nil) {
		c.log.Warn("redis connection error, falling back to DB", slog.Any("error", err))
	}

	c.log.Info("catalog cache miss, fetching from database")
	services, err := c.repo.GetAllServices(ctx)
	if err != nil {
		return nil, err
	}
	apiServices := make([]api.Service, len(services))
	for i, s := range services {
		apiServices[i] = api.Service{
			Id:          s.ID,
			Name:        s.Name,
			Description: s.Description,
			DockerImage: &s.DockerImage,
		}
	}

	go func() {
		bgCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
		defer cancel()
		cacheBytes, err := json.Marshal(apiServices)
		if err == nil {
			err = c.redis.Set(bgCtx, catalogCacheKey, cacheBytes, catalogCacheTTL).Err()
			if err != nil {
				c.log.Warn("failed to set cache in redis", slog.Any("error", err))
			} else {
				c.log.Info("catalog successfully cached in redis")
			}

		}
	}()
	return api.GetCatalog200JSONResponse{
		Services: &apiServices,
	}, nil
}

func (c *CatalogService) InternalGetServiceById(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		http.Error(w, "missing service id", http.StatusBadRequest)
		return
	}

	serviceTpl, err := c.repo.GetService(r.Context(), id)
	if err != nil {
		if errors.Is(err, ServiceNotFound) {
			http.Error(w, "service not found", http.StatusNotFound)
			return
		}
		c.log.Error("failed to get internal service", slog.String("id", id), slog.Any("error", err))
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(serviceTpl); err != nil {
		c.log.Error("failed to encode response", slog.Any("error", err))
	}
}
