package deploy

import (
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
	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/network"
	"github.com/moby/moby/client"
)

const (
	StatusQueueName = "instance_updates"
	LocalhostIP     = "127.0.0.1"
)

type CatalogClient interface {
	GetServiceInfo(ctx context.Context, serviceID string) (*CatalogTemplate, error)
}

type EventPublisher interface {
	Publish(ctx context.Context, queue string, body []byte) error
}

type DeployService struct {
	log     *slog.Logger
	docker  client.APIClient
	rmq     EventPublisher
	catalog CatalogClient
}

type InstanceStatusEvent struct {
	InstanceID    string `json:"instance_id"`
	UserID        int64  `json:"user_id"`
	ServiceName   string `json:"service_name,omitempty"`
	Status        string `json:"status"`
	ConnectionURL string `json:"connection_url,omitempty"`
}

type CatalogTemplate struct {
	DockerImage string `json:"docker_image"`
	ExposedPort string `json:"exposed_port"`
	Scheme      string `json:"scheme"`
	DefaultUser string `json:"default_user"`
}

func NewDeployService(log *slog.Logger, dockerClient client.APIClient, rmq EventPublisher, catalog CatalogClient) *DeployService {
	return &DeployService{
		log:     log,
		docker:  dockerClient,
		rmq:     rmq,
		catalog: catalog,
	}
}

func (s *DeployService) ProvisionInstance(ctx context.Context, req api.ProvisionInstanceRequestObject) (api.ProvisionInstanceResponseObject, error) {
	userID, err := extractUserID(ctx)
	if err != nil {
		s.log.Error("failed to extract user_id", slog.Any("error", err))
		return api.ProvisionInstance500JSONResponse{Error: stringPtr(err.Error())}, nil
	}

	serviceName := req.Body.ServiceId
	s.log.Info("starting provisioning", "instance_id", req.InstanceId, "service_id", serviceName)

	template, err := s.catalog.GetServiceInfo(ctx, serviceName)
	if err != nil {
		s.log.Error("error getting service info", slog.Any("error", err))
		return api.ProvisionInstance500JSONResponse{Error: stringPtr("Catalog unavailable")}, nil
	}

	password, err := generateRandomPassword(12)
	if err != nil {
		s.log.Error("failed to generate password", slog.Any("error", err))
		return api.ProvisionInstance500JSONResponse{Error: stringPtr("Internal server error")}, nil
	}

	hostPort, err := s.deployDockerContainer(ctx, userID, req.InstanceId, template, password)
	if err != nil {
		s.log.Error("container deployment failed", slog.Any("error", err))
		return api.ProvisionInstance500JSONResponse{Error: stringPtr("Deployment failed")}, nil
	}

	connectionURL := buildConnectionURL(template.Scheme, template.DefaultUser, password, LocalhostIP, hostPort)

	s.publishStatus(ctx, req.InstanceId, userID, serviceName, "running", connectionURL)

	return api.ProvisionInstance201JSONResponse{
		ConnectionDetails: &struct {
			Password *string `json:"password,omitempty"`
			Url      *string `json:"url,omitempty"`
			Username *string `json:"username,omitempty"`
		}{
			Url:      stringPtr(connectionURL),
			Username: stringPtr(template.DefaultUser),
			Password: stringPtr(password),
		},
	}, nil
}

func (s *DeployService) DeprovisionInstance(ctx context.Context, req api.DeprovisionInstanceRequestObject) (api.DeprovisionInstanceResponseObject, error) {
	userID, err := extractUserID(ctx)
	if err != nil {
		s.log.Error("failed to extract user_id in deprovision", slog.Any("error", err))
		return api.DeprovisionInstance500Response{}, nil
	}

	containerName := buildContainerName(userID, req.InstanceId)
	s.log.Info("deprovisioning instance", "instance_id", req.InstanceId, "container", containerName)

	_, err = s.docker.ContainerRemove(ctx, containerName, client.ContainerRemoveOptions{Force: true, RemoveVolumes: true})
	if err != nil {
		if errdefs.IsNotFound(err) {
			s.log.Warn("instance already deprovisioned", "instance_id", req.InstanceId)
			return api.DeprovisionInstance410Response{}, nil
		}
		s.log.Error("failed to remove container", "error", err)
		return api.DeprovisionInstance500Response{}, nil
	}

	s.publishStatus(ctx, req.InstanceId, userID, "", "deleted", "")
	s.log.Info("instance deprovisioned successfully", "instance_id", req.InstanceId)

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
		s.log.Warn("failed to pull image (might use local)", slog.Any("error", err))
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
		s.log.Error("failed to marshal status event", "error", err, "instance_id", instanceID)
		return
	}

	if err := s.rmq.Publish(ctx, StatusQueueName, eventBytes); err != nil {
		s.log.Error("failed to publish status to rabbitmq", "error", err, "instance_id", instanceID)
	} else {
		s.log.Info("status update published", "instance_id", instanceID, "status", status)
	}
}

func extractUserID(ctx context.Context) (int64, error) {
	userIDStr, ok := ctx.Value(UserIDKey).(string)
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
