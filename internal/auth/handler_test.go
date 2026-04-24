package auth

import (
	"context"
	"log/slog"
	"os"
	"testing"

	api "SaaS_Pipeline/internal/auth/api"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

func setupTestService(t *testing.T) (*AuthService, sqlmock.Sqlmock) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)

	repo := NewRepository(db)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	service := NewAuthService(repo, []byte("super-secret-key"), logger)
	return service, mock
}

func TestAuthService_RegisterUser_Success(t *testing.T) {
	service, mock := setupTestService(t)

	mock.ExpectExec("INSERT INTO users").
		WithArgs("test@example.com", sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	req := api.RegisterUserRequestObject{
		Body: &api.RegisterUserJSONRequestBody{
			Email:    "test@example.com",
			Password: "password123",
		},
	}

	resp, err := service.RegisterUser(context.Background(), req)

	assert.NoError(t, err)
	assert.IsType(t, api.RegisterUser201Response{}, resp)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestAuthService_LoginUser_Success(t *testing.T) {
	service, mock := setupTestService(t)

	hash, err := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	require.NoError(t, err)

	rows := sqlmock.NewRows([]string{"id", "email", "password_hash"}).
		AddRow(1, "test@example.com", string(hash))

	mock.ExpectQuery("SELECT id, email, password_hash FROM users").
		WithArgs("test@example.com").
		WillReturnRows(rows)

	req := api.LoginUserRequestObject{
		Body: &api.LoginUserJSONRequestBody{
			Email:    "test@example.com",
			Password: "password123",
		},
	}

	resp, err := service.LoginUser(context.Background(), req)

	assert.NoError(t, err)

	successResp, ok := resp.(api.LoginUser200JSONResponse)
	assert.True(t, ok)
	assert.NotNil(t, successResp.Token)
	assert.NotEmpty(t, *successResp.Token)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestAuthService_LoginUser_WrongPassword(t *testing.T) {
	service, mock := setupTestService(t)

	hash, err := bcrypt.GenerateFromPassword([]byte("correct-password"), bcrypt.DefaultCost)
	require.NoError(t, err)

	rows := sqlmock.NewRows([]string{"id", "email", "password_hash"}).
		AddRow(1, "test@example.com", string(hash))

	mock.ExpectQuery("SELECT id, email, password_hash FROM users").
		WithArgs("test@example.com").
		WillReturnRows(rows)

	req := api.LoginUserRequestObject{
		Body: &api.LoginUserJSONRequestBody{
			Email:    "test@example.com",
			Password: "wrong-password",
		},
	}

	resp, err := service.LoginUser(context.Background(), req)

	assert.NoError(t, err)
	assert.IsType(t, api.LoginUser401Response{}, resp)
}
