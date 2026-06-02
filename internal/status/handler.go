package status

import (
	"context"
	"errors"
	"log/slog"

	api "SaaS_Pipeline/internal/status/api"
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
	userID, ok := ctx.Value(UserIDKey).(int64)
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
