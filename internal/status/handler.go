package status

import (
	api "SaaS_Pipeline/internal/status/api"
	"context"
	"log/slog"
)

type StatusService struct {
	repo *Repository
	log  *slog.Logger
}

func NewStatusService(repo *Repository, log *slog.Logger) *StatusService {
	return &StatusService{
		repo: repo,
		log:  log,
	}
}

func (s *StatusService) ListUserInstances(ctx context.Context, request api.ListUserInstancesRequestObject) (api.ListUserInstancesResponseObject, error) {
	userID, ok := ctx.Value("user_id").(int64)
	if !ok {
		return nil, nil
	}
	instances, err := s.repo.GetInstancesByUserID(ctx, userID)
	if err != nil {
		s.log.Error("failed to get user instances", slog.Any("error", err))
		return nil, err
	}
	var response api.ListUserInstances200JSONResponse
	for _, inst := range instances {
		id := inst.ID
		name := inst.ServiceName
		url := inst.ConnectionURL
		statusEnum := api.InstanceInfoStatus(inst.Status)

		response = append(response, api.InstanceInfo{
			Id:            &id,
			ServiceName:   &name,
			Status:        &statusEnum,
			ConnectionUrl: url,
		})
	}
	return response, nil
}
