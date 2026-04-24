package catalog

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRepository_GetAllServices_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewRepository(db)

	rows := sqlmock.NewRows([]string{"id", "name", "description", "docker_image"}).
		AddRow(1, "Ubuntu CLI", "Терминал Ubuntu 22.04", "ubuntu:22.04")

	mock.ExpectQuery("SELECT id, name, description, docker_image FROM services").WithoutArgs().WillReturnRows(rows)
	services, err := repo.GetAllServices(context.Background())
	require.NoError(t, err)
	require.NotNil(t, services)
	require.Equal(t, 1, len(services))
	assert.Equal(t, services[0].Name, "Ubuntu CLI")
	assert.Equal(t, services[0].Description, "Терминал Ubuntu 22.04")
	assert.Equal(t, services[0].DockerImage, "ubuntu:22.04")

	err = mock.ExpectationsWereMet()
	require.NoError(t, err)
}

func TestRepository_GetService_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewRepository(db)

	rows := sqlmock.NewRows([]string{"docker_image", "exposed_port", "scheme", "default_user"}).
		AddRow("ubuntu:22.04", "22/tcp", "ssh", "root")

	mock.ExpectQuery("^SELECT docker_image, exposed_port, scheme, default_user FROM services WHERE name = \\$1$").WithArgs("Ubuntu CLI").WillReturnRows(rows)

	services, err := repo.GetService(context.Background(), "Ubuntu CLI")
	require.NoError(t, err)
	require.NotNil(t, services)
	assert.Equal(t, services.DockerImage, "ubuntu:22.04")
	assert.Equal(t, services.ExposedPort, "22/tcp")
	assert.Equal(t, services.Scheme, "ssh")
	assert.Equal(t, services.DefaultUser, "root")

	err = mock.ExpectationsWereMet()
	require.NoError(t, err)
}
