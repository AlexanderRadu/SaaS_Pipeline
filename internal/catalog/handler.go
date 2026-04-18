package catalog

import (
	api "SaaS_Pipeline/internal/catalog/api"
	"context"
	"log/slog"
)

type CatalogService struct {
	repo Repository
	log  *slog.Logger
}

func NewCatalogService(repo Repository) *CatalogService {
	return &CatalogService{
		repo: repo,
	}
}

func (c *CatalogService) GetCatalog(ctx context.Context, request api.GetCatalogRequestObject) (api.GetCatalogResponseObject, error) {
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
	return api.GetCatalog200JSONResponse{
		Services: &apiServices,
	}, nil
}
