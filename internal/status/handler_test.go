package status

import (
	api "SaaS_Pipeline/internal/status/api"
	"context"
	"io"
	"log/slog"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStatusService_ListUserInstances_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewRepository(db)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	service := NewStatusService(repo, logger)

	userID := int64(123)
	testURL := "http://example.com"

	rows := sqlmock.NewRows([]string{"id", "user_id", "service_name", "status", "connection_url"}).
		AddRow("inst-1", userID, "ubuntu", "running", &testURL)

	mock.ExpectQuery(`SELECT id, user_id, service_name, status, connection_url FROM instances WHERE user_id = \$1`).
		WithArgs(userID).
		WillReturnRows(rows)

	ctx := context.WithValue(context.Background(), "user_id", userID)
	req := api.ListUserInstancesRequestObject{}

	respObj, err := service.ListUserInstances(ctx, req)

	require.NoError(t, err)

	resp, ok := respObj.(api.ListUserInstances200JSONResponse)
	require.True(t, ok, "Expected response of type ListUserInstances200JSONResponse")
	require.Len(t, resp, 1)

	assert.Equal(t, "inst-1", *resp[0].Id)
	assert.Equal(t, "ubuntu", *resp[0].ServiceName)
	assert.Equal(t, api.InstanceInfoStatus("running"), *resp[0].Status)
	assert.Equal(t, testURL, *resp[0].ConnectionUrl)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestStatusService_ListUserInstances_NoContextUserID(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewRepository(db)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	service := NewStatusService(repo, logger)

	ctx := context.Background()
	req := api.ListUserInstancesRequestObject{}

	respObj, err := service.ListUserInstances(ctx, req)

	assert.NoError(t, err)
	assert.Nil(t, respObj)

	assert.NoError(t, mock.ExpectationsWereMet())
}
