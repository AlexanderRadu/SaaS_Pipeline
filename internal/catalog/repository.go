package catalog

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

var ErrEmptyCatalog = errors.New("catalog is empty")
var ServiceNotFound = errors.New("service not found")

type ServiceTemplate struct {
	DockerImage string `json:"docker_image"`
	ExposedPort string `json:"exposed_port"`
	Scheme      string `json:"scheme"`
	DefaultUser string `json:"default_user"`
}

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{
		db: db,
	}
}

func (r *Repository) GetAllServices(ctx context.Context) ([]Service, error) {
	query := `SELECT id, name, description, docker_image FROM services`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query error: %w", err)
	}
	defer rows.Close()

	var services []Service

	for rows.Next() {
		var s Service
		err := rows.Scan(&s.ID, &s.Name, &s.Description, &s.DockerImage)
		if err != nil {
			return nil, fmt.Errorf("rows scan error: %w", err)
		}
		services = append(services, s)

	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}
	if len(services) == 0 {
		return nil, ErrEmptyCatalog
	}
	return services, nil
}

func (r *Repository) GetService(ctx context.Context, serviceName string) (*ServiceTemplate, error) {
	query := `SELECT docker_image, exposed_port, scheme, default_user FROM services WHERE name = $1`
	row := r.db.QueryRowContext(ctx, query, serviceName)
	var service ServiceTemplate

	err := row.Scan(&service.DockerImage, &service.ExposedPort, &service.Scheme, &service.DefaultUser)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ServiceNotFound
		}
		return nil, err
	}
	return &service, nil
}
