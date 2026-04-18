package status

import (
	"context"
	"database/sql"
	"fmt"
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) GetInstancesByUserID(ctx context.Context, userID int64) ([]Instance, error) {
	query := `SELECT id, user_id, service_name, status, connection_url, created_at FROM instances WHERE user_id = $1`
	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var instances []Instance
	for rows.Next() {
		var inst Instance
		if err := rows.Scan(&inst.ID, &inst.UserID, &inst.ServiceName, &inst.Status, &inst.ConnectionURL, &inst.CreatedAt); err != nil {
			return nil, err
		}
		instances = append(instances, inst)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}
	if len(instances) == 0 {
		return []Instance{}, nil
	}

	return instances, nil
}

func (r *Repository) UpdateInstanceStatus(instanceID string, status string, connectionURL string) error {
	query := `UPDATE instances SET status = $1, connection_url = $2 WHERE id = $3`
	_, err := r.db.Exec(query, status, connectionURL, instanceID)
	return err
}
