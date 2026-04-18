package catalog

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

var ErrEmptyCatalog = errors.New("catalog is empty")

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
