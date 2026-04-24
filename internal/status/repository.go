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
	query := `SELECT id, user_id, service_name, status, connection_url FROM instances WHERE user_id = $1`
	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var instances []Instance
	for rows.Next() {
		var inst Instance
		if err := rows.Scan(&inst.ID, &inst.UserID, &inst.ServiceName, &inst.Status, &inst.ConnectionURL); err != nil {
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

func (r *Repository) UpsertInstance(ctx context.Context, event InstanceUpdateMessage) error {
	query := `
        INSERT INTO instances (id, user_id, service_name, status, connection_url)
        VALUES ($1, $2, $3, $4, $5)
        ON CONFLICT (id, user_id)     
        DO UPDATE SET 
            status = EXCLUDED.status,
            connection_url = EXCLUDED.connection_url;`

	_, err := r.db.ExecContext(ctx, query,
		event.InstanceID,
		event.UserID,
		event.ServiceName,
		event.Status,
		event.ConnectionURL,
	)

	return err
}
