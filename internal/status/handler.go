package status

import (
	"context"
	"errors"
	"log/slog"

	api "SaaS_Pipeline/internal/status/api"
	"SaaS_Pipeline/pkg/auth"
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
	userID, ok := ctx.Value(auth.UserIDKey).(int64)
	if !ok {
		s.log.ErrorContext(ctx, "user_id not found in context or has invalid type")
		return nil, errors.New("unauthorized: missing or invalid user_id in context")
	}

	instances, err := s.repo.GetInstancesByUserID(ctx, userID)
	if err != nil {
		s.log.ErrorContext(ctx, "failed to get user instances",
			slog.Any("error", err),
			slog.Int64("user_id", userID),
		)
		return nil, err
	}

	var response api.ListUserInstances200JSONResponse
	for _, inst := range instances {
		url := inst.ConnectionURL
		response = append(response, api.InstanceInfo{
			Id:            new(inst.ID),
			ServiceName:   new(inst.ServiceName),
			Status:        new(api.InstanceInfoStatus(inst.Status)),
			ConnectionUrl: url,
		})
	}

	return response, nil
}
