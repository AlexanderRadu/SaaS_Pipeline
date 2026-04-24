package status

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
)

func TestRepository_GetInstancesByUserID(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	repo := NewRepository(db)
	rows := sqlmock.NewRows([]string{"id", "user_id", "service_name", "status", "connection_url"}).
		AddRow("inst-1", 123, "ubuntu", "running", "http://example.com")

	mock.ExpectQuery("^SELECT id, user_id, service_name, status, connection_url FROM instances WHERE user_id = \\$1$").WithArgs(123).WillReturnRows(rows)

	instances, err := repo.GetInstancesByUserID(context.Background(), 123)
	require.NoError(t, err)
	require.Len(t, instances, 1)
	require.Equal(t, "inst-1", instances[0].ID)
	require.Equal(t, int64(123), instances[0].UserID)
	require.Equal(t, "ubuntu", instances[0].ServiceName)
	require.Equal(t, "running", instances[0].Status)
	require.Equal(t, "http://example.com", *instances[0].ConnectionURL)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_UpsertInstance_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	repo := NewRepository(db)

	event := InstanceUpdateMessage{
		InstanceID:    "inst-1",
		UserID:        123,
		ServiceName:   "ubuntu",
		Status:        "running",
		ConnectionURL: "http://example.com",
	}
	expectedQuery := `INSERT INTO instances .* ON CONFLICT .* DO UPDATE SET .*`
	mock.ExpectExec(expectedQuery).
		WithArgs(
			event.InstanceID,
			event.UserID,
			event.ServiceName,
			event.Status,
			event.ConnectionURL,
		).WillReturnResult(sqlmock.NewResult(1, 1))
	err = repo.UpsertInstance(context.Background(), event)
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_GetInstancesByUserID_Empty(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	repo := NewRepository(db)

	rows := sqlmock.NewRows([]string{"id", "user_id", "service_name", "status", "connection_url"})
	mock.ExpectQuery("^SELECT id, user_id, service_name, status, connection_url FROM instances WHERE user_id = \\$1$").
		WithArgs(123).WillReturnRows(rows)

	instances, err := repo.GetInstancesByUserID(context.Background(), 123)
	require.NoError(t, err)
	require.Empty(t, instances)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_GetInstancesByUserID_QueryError(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	repo := NewRepository(db)

	mock.ExpectQuery("^SELECT id, user_id, service_name, status, connection_url FROM instances WHERE user_id = \\$1$").
		WithArgs(123).WillReturnError(context.Canceled)

	instances, err := repo.GetInstancesByUserID(context.Background(), 123)
	require.Error(t, err)
	require.Nil(t, instances)
	require.NoError(t, mock.ExpectationsWereMet())
}
