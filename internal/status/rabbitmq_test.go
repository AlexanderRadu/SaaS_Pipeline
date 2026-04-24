package status

import (
	"context"
	"io"
	"testing"

	"log/slog"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConsumer_ProccessMessage_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewRepository(db)
	log := slog.New(slog.NewTextHandler(io.Discard, nil))

	consumer := &Consumer{
		repo: repo,
		log:  log,
	}

	validJSON := []byte(`{
		"instance_id": "rbt-1",
		"user_id": 999,
		"service_name": "postgres",
		"status": "online",
		"connection_url": "postgres://user:pass@localhost:5432"
	}`)

	mock.ExpectExec(`INSERT INTO instances .* ON CONFLICT .* DO UPDATE SET .*`).
		WithArgs(
			"rbt-1",
			int64(999),
			"postgres",
			"online",
			"postgres://user:pass@localhost:5432",
		).WillReturnResult(sqlmock.NewResult(1, 1))

	err = consumer.ProcessMessage(context.Background(), validJSON)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestConsumer_ProcessMessage_InvalidJSON(t *testing.T) {
	repo := NewRepository(nil)
	log := slog.New(slog.NewTextHandler(io.Discard, nil))

	consumer := &Consumer{
		repo: repo,
		log:  log,
	}

	invalidJSON := []byte(`{"instance_id": "rbt-1", "user...}`)
	err := consumer.ProcessMessage(context.Background(), invalidJSON)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected end of JSON input")
}
