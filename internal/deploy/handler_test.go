package deploy

import (
	api "SaaS_Pipeline/internal/deploy/api"
	"SaaS_Pipeline/pkg/auth"
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
)

type mockCatalogClient struct {
	res *CatalogTemplate
	err error
}

func (m *mockCatalogClient) GetServiceInfo(ctx context.Context, serviceID string) (*CatalogTemplate, error) {
	return m.res, m.err
}

type mockEventPublisher struct {
	published [][]byte
	err       error
}

func (m *mockEventPublisher) Publish(ctx context.Context, queue string, body []byte) error {
	m.published = append(m.published, body)
	return m.err
}

func TestBuildENV_Postgres(t *testing.T) {
	envs := buildENV("postgres", "pass", "user")
	assert.Contains(t, envs, "POSTGRES_USER=user")
	assert.Contains(t, envs, "POSTGRES_PASSWORD=pass")
}

func TestBuildENV_Unknown(t *testing.T) {
	envs := buildENV("unknown", "pass", "user")
	assert.Empty(t, envs)
}

func TestGenerateRandomPassword(t *testing.T) {
	pass1, err := generateRandomPassword(12)
	assert.NoError(t, err)
	assert.Len(t, pass1, 12)
}

func TestBuildContainerName(t *testing.T) {
	assert.Equal(t, "saas_instance_user42_inst-1", buildContainerName(42, "inst-1"))
}

func TestBuildConnectionURL(t *testing.T) {
	url1 := buildConnectionURL("postgres", "user", "pass", "127.0.0.1", "5432")
	assert.Equal(t, "postgres://user:pass@127.0.0.1:5432", url1)

	url2 := buildConnectionURL("http", "", "", "127.0.0.1", "8080")
	assert.Equal(t, "http://127.0.0.1:8080", url2)
}

func TestExtractUserID(t *testing.T) {
	_, err := extractUserID(context.Background())
	assert.Error(t, err)

	ctxVal := context.WithValue(context.Background(), auth.UserIDKey, "123")
	uid, err := extractUserID(ctxVal)
	assert.NoError(t, err)
	assert.Equal(t, int64(123), uid)
}

func TestDeployService_DeprovisionInstance_Unauthorized(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	service := NewDeployService(logger, nil, nil, nil)

	req := api.DeprovisionInstanceRequestObject{InstanceId: "123"}
	resp, err := service.DeprovisionInstance(context.Background(), req)

	assert.NoError(t, err)
	_, isErrStatus := resp.(api.DeprovisionInstance500Response)
	assert.True(t, isErrStatus)
}

func TestDeployService_ProvisionInstance_Unauthorized(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	service := NewDeployService(logger, nil, nil, nil)

	req := api.ProvisionInstanceRequestObject{}
	resp, err := service.ProvisionInstance(context.Background(), req)
	assert.NoError(t, err)

	res, isErrStatus := resp.(api.ProvisionInstance500JSONResponse)
	assert.True(t, isErrStatus)
	assert.Contains(t, *res.Error, "unauthorized")
}

func TestDeployService_ProvisionInstance_CatalogError(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	catMock := &mockCatalogClient{err: errors.New("catalog down")}
	service := NewDeployService(logger, nil, nil, catMock)

	ctx := context.WithValue(context.Background(), auth.UserIDKey, "123")
	req := api.ProvisionInstanceRequestObject{
		InstanceId: "inst",
		Body:       &api.ProvisionInstanceJSONRequestBody{ServiceId: "db"},
	}

	resp, err := service.ProvisionInstance(ctx, req)
	assert.NoError(t, err)

	res, ok := resp.(api.ProvisionInstance500JSONResponse)
	assert.True(t, ok)
	assert.Contains(t, *res.Error, "Catalog unavailable")
}

func TestDeployService_ProvisionInstance_DeployError_BadPort(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	catMock := &mockCatalogClient{
		res: &CatalogTemplate{
			DockerImage: "postgres",
			ExposedPort: "bad_port_format",
			Scheme:      "postgres",
			DefaultUser: "admin",
		},
	}
	service := NewDeployService(logger, nil, nil, catMock)

	ctx := context.WithValue(context.Background(), auth.UserIDKey, "123")
	req := api.ProvisionInstanceRequestObject{
		InstanceId: "1",
		Body:       &api.ProvisionInstanceJSONRequestBody{ServiceId: "db"},
	}

	resp, err := service.ProvisionInstance(ctx, req)
	assert.NoError(t, err)

	res, ok := resp.(api.ProvisionInstance500JSONResponse)
	assert.True(t, ok)
	assert.Contains(t, *res.Error, "Deployment failed")
}

func TestDeployService_PublishStatus(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	mockPub := &mockEventPublisher{}
	service := NewDeployService(logger, nil, mockPub, nil)

	service.publishStatus(context.Background(), "inst-1", 10, "svc", "running", "url")
	assert.Len(t, mockPub.published, 1)
	assert.Contains(t, string(mockPub.published[0]), "running")

	mockPub.err = errors.New("rabbitmq offline")
	service.publishStatus(context.Background(), "inst-2", 10, "svc", "stopped", "url")
	assert.Len(t, mockPub.published, 2)
}
