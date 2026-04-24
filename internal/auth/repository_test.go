package auth

import (
	"context"
	"database/sql"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRepository_GetUserByEmail_NotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewRepository(db)

	mock.ExpectQuery("SELECT id, email, password_hash FROM users").
		WithArgs("notfound@example.com").
		WillReturnError(sql.ErrNoRows)

	user, err := repo.GetUserByEmail(context.Background(), "notfound@example.com")

	assert.Error(t, err)
	assert.Nil(t, user)
	assert.ErrorIs(t, err, ErrUserNotFound)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_GetUserByEmail_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewRepository(db)

	rows := sqlmock.NewRows([]string{"id", "email", "password_hash"}).
		AddRow(42, "user@example.com", "hashed_password")

	mock.ExpectQuery("SELECT id, email, password_hash FROM users").
		WithArgs("user@example.com").
		WillReturnRows(rows)

	user, err := repo.GetUserByEmail(context.Background(), "user@example.com")

	assert.NoError(t, err)
	assert.NotNil(t, user)
	assert.Equal(t, int64(42), user.ID)
	assert.Equal(t, "user@example.com", user.Email)
	assert.Equal(t, "hashed_password", user.PasswordHash)
}
