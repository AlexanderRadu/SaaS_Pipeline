package catalog

import (
	api "SaaS_Pipeline/internal/catalog/api"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-redis/redismock/v9"
	"github.com/stretchr/testify/require"
)

func TestCatalogService_GetCatalog_CacheMiss(t *testing.T) {
	db, sqlMock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	repo := NewRepository(db)

	rdb, redisMock := redismock.NewClientMock()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	service := NewCatalogService(repo, logger, rdb)

	ctx := context.Background()

	dockerImg := "ubuntu:latest"
	expectedServices := []api.Service{
		{
			Id:          "1",
			Name:        "Ubuntu",
			Description: "Test OS",
			DockerImage: &dockerImg,
		},
	}
	expectedJSON, _ := json.Marshal(expectedServices)

	redisMock.ExpectSet("catalog:all_services", expectedJSON, 5*time.Minute).SetVal("OK")
	rows := sqlmock.NewRows([]string{"id", "name", "description", "docker_image"}).
		AddRow(1, "Ubuntu", "Test OS", "ubuntu:latest")
	sqlMock.ExpectQuery("^SELECT (.+) FROM services").WillReturnRows(rows)

	resp, err := service.GetCatalog(ctx, api.GetCatalogRequestObject{})

	require.NoError(t, err)

	jsonResp, ok := resp.(api.GetCatalog200JSONResponse)
	require.True(t, ok)
	require.NotNil(t, jsonResp.Services)
	require.Len(t, *jsonResp.Services, 1)
	require.Equal(t, "Ubuntu", (*jsonResp.Services)[0].Name)

	time.Sleep(50 * time.Millisecond)

	require.NoError(t, sqlMock.ExpectationsWereMet())
	require.NoError(t, redisMock.ExpectationsWereMet())
}
