package deploy

import (
	"SaaS_Pipeline/pkg/auth"
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strconv"

	api "SaaS_Pipeline/internal/deploy/api"

	"github.com/containerd/errdefs"
	"github.com/google/uuid"
	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/network"
	"github.com/moby/moby/client"
)

const (
	StatusQueueName = "instance_updates"
	LocalhostIP     = "127.0.0.1"
)

type TaskType string

const (
	TaskProvision   TaskType = "provision"
	TaskDeprovision TaskType = "deprovision"
	DeployTaskQueue          = "deploy_tasks"
)

type DeployTaskEvent struct {
	OperationID string   `json:"operation_id"`
	UserID      string   `json:"user_id"`
	TaskType    TaskType `json:"task_type"`
	InstanceID  string   `json:"instance_id"`
	ServiceID   string   `json:"service_id,omitempty"`
}

type CatalogClient interface {
	GetServiceInfo(ctx context.Context, serviceID string) (*CatalogTemplate, error)
}

type EventPublisher interface {
	Publish(ctx context.Context, queue string, body []byte) error
}

type DeployService struct {
	log *slog.Logger
	rmq EventPublisher
}

type InstanceStatusEvent struct {
	OperationID   string `json:"operation_id"`
	InstanceID    string `json:"instance_id"`
	UserID        int64  `json:"user_id"`
	ServiceName   string `json:"service_name,omitempty"`
	Status        string `json:"status"`
	ConnectionURL string `json:"connection_url,omitempty"`
	Error         string `json:"error,omitempty"`
}

type CatalogTemplate struct {
	DockerImage string `json:"docker_image"`
	ExposedPort string `json:"exposed_port"`
	Scheme      string `json:"scheme"`
	DefaultUser string `json:"default_user"`
}

func NewDeployService(log *slog.Logger, rmq EventPublisher) *DeployService {
	return &DeployService{
		log: log,
		rmq: rmq,
	}
}

func (s *DeployService) ProvisionInstance(ctx context.Context, req api.ProvisionInstanceRequestObject) (api.ProvisionInstanceResponseObject, error) {
	userID, err := extractUserID(ctx)
	if err != nil {
		s.log.ErrorContext(ctx, "failed to extract user_id", slog.Any("error", err))
		return api.ProvisionInstance400JSONResponse{Error: stringPtr(err.Error())}, nil
	}
	serviceID := req.Body.ServiceId
	instanceID := req.InstanceId

	operationID := uuid.New()

	s.log.InfoContext(ctx, "received provision request, queuing task", "instance_id", instanceID, "operation_id", operationID)

	s.publishStatus(ctx, instanceID, userID, serviceID, "provisioning", "")

	task := DeployTaskEvent{
		OperationID: operationID,
		UserID:      userID,
		ServiceID:   serviceID,
		InstanceID:  instanceID,
		TaskType:    TaskProvision,
	}

	taskBytes, err := json.Marshal(task)
	if err != nil {
		s.log.ErrorContext(ctx, "failed to marshal task to json", slog.Any("error", err))
		return api.ProvisionInstance500JSONResponse{Error: stringPtr("internal error")}, nil
	}
}

func (s *DeployService) DeprovisionInstance(ctx context.Context, req api.DeprovisionInstanceRequestObject) (api.DeprovisionInstanceResponseObject, error) {
	userID, err := extractUserID(ctx)
	if err != nil {
		s.log.ErrorContext(ctx, "failed to extract user_id in deprovision", slog.Any("error", err))
		return api.DeprovisionInstance500Response{}, nil
	}

	containerName := buildContainerName(userID, req.InstanceId)
	s.log.InfoContext(ctx, "deprovisioning instance", "instance_id", req.InstanceId, "container", containerName)

	_, err = s.docker.ContainerRemove(ctx, containerName, client.ContainerRemoveOptions{Force: true, RemoveVolumes: true})
	if err != nil {
		if errdefs.IsNotFound(err) {
			s.log.WarnContext(ctx, "instance already deprovisioned", "instance_id", req.InstanceId)
			return api.DeprovisionInstance410Response{}, nil
		}
		s.log.ErrorContext(ctx, "failed to remove container", "error", err)
		return api.DeprovisionInstance500Response{}, nil
	}

	s.publishStatus(ctx, req.InstanceId, userID, "", "deleted", "")
	s.log.InfoContext(ctx, "instance deprovisioned successfully", "instance_id", req.InstanceId)

	return api.DeprovisionInstance204Response{}, nil
}

func (s *DeployService) deployDockerContainer(ctx context.Context, userID int64, instanceID string, template *CatalogTemplate, password string) (string, error) {
	containerPort, err := network.ParsePort(template.ExposedPort)
	if err != nil {
		return "", fmt.Errorf("invalid port in catalog: %w", err)
	}

	s.log.Info("pulling image", "image", template.DockerImage)
	pullResp, err := s.docker.ImagePull(ctx, template.DockerImage, client.ImagePullOptions{})
	if err != nil {
		s.log.WarnContext(ctx, "failed to pull image (might use local)", slog.Any("error", err))
	} else {
		_, _ = io.Copy(io.Discard, pullResp)
		pullResp.Close()
	}

	containerName := buildContainerName(userID, instanceID)
	envVars := buildENV(template.Scheme, password, template.DefaultUser)

	createResp, err := s.docker.ContainerCreate(ctx, client.ContainerCreateOptions{
		Image: template.DockerImage,
		Config: &container.Config{
			Env: envVars,
			Tty: true,
			ExposedPorts: network.PortSet{
				containerPort: struct{}{},
			},
		},
		HostConfig: &container.HostConfig{
			PublishAllPorts: true,
		},
		Platform:         nil,
		NetworkingConfig: nil,
		Name:             containerName,
	})

	if err != nil {
		return "", fmt.Errorf("failed to create container: %w", err)
	}

	_, err = s.docker.ContainerStart(ctx, createResp.ID, client.ContainerStartOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to start container: %w", err)
	}

	inspect, err := s.docker.ContainerInspect(ctx, createResp.ID, client.ContainerInspectOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to inspect container: %w", err)
	}

	if inspect.Container.NetworkSettings != nil && inspect.Container.NetworkSettings.Ports != nil {
		for p, bindings := range inspect.Container.NetworkSettings.Ports {
			if p.String() == containerPort.String() && len(bindings) > 0 {
				return bindings[0].HostPort, nil
			}
		}
	}

	return "", errors.New("failed to find exposed port mapping")
}

func (s *DeployService) publishStatus(ctx context.Context, instanceID string, userID int64, serviceName, status, url string) {
	event := InstanceStatusEvent{
		InstanceID:    instanceID,
		UserID:        userID,
		ServiceName:   serviceName,
		Status:        status,
		ConnectionURL: url,
	}

	eventBytes, err := json.Marshal(event)
	if err != nil {
		s.log.ErrorContext(ctx, "failed to marshal status event", "error", err, "instance_id", instanceID)
		return
	}

	if err := s.rmq.Publish(ctx, StatusQueueName, eventBytes); err != nil {
		s.log.ErrorContext(ctx, "failed to publish status to rabbitmq", "error", err, "instance_id", instanceID)
	} else {
		s.log.InfoContext(ctx, "status update published", "instance_id", instanceID, "status", status)
	}
}

func extractUserID(ctx context.Context) (int64, error) {
	userIDStr, ok := ctx.Value(auth.UserIDKey).(string)
	if !ok || userIDStr == "" {
		return 0, errors.New("unauthorized or missing X-User-Id header")
	}
	return strconv.ParseInt(userIDStr, 10, 64)
}

func buildContainerName(userID int64, instanceID string) string {
	return fmt.Sprintf("saas_instance_user%d_%s", userID, instanceID)
}

func buildConnectionURL(scheme, username, password, host, port string) string {
	if password != "" && username != "" {
		return fmt.Sprintf("%s://%s:%s@%s:%s", scheme, username, password, host, port)
	}
	return fmt.Sprintf("%s://%s:%s", scheme, host, port)
}

func generateRandomPassword(length int) (string, error) {
	const chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to read random bytes: %w", err) // НИКОГДА не возвращаем хардкод пароль
	}
	for i := range b {
		b[i] = chars[int(b[i])%len(chars)]
	}
	return string(b), nil
}

func buildENV(scheme string, password string, username string) []string {
	switch scheme {
	case "postgres":
		return []string{
			fmt.Sprintf("POSTGRES_USER=%s", username),
			fmt.Sprintf("POSTGRES_PASSWORD=%s", password),
		}
	}
	return []string{}
}

func stringPtr(str string) *string { return &str }
