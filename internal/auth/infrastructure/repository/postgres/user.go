package postgres

import (
	"SaaS_Pipeline/internal/auth/entity"
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UserRepository struct {
	db *pgxpool.Pool
}

func NewUserRepository(db *pgxpool.Pool) *UserRepository {
	return &UserRepository{db: db}
}

func (u *UserRepository) CreateUser(ctx context.Context, user *entity.User) error {
	query := `INSERT INTO users (email, password_hash) VALUES ($1, $2)`
	_, err := u.db.Exec(ctx, query, user.Email, user.PasswordHash)
	if err != nil {
		return err
	}
	return nil
}

func (u *UserRepository) GetUserByEmail(ctx context.Context, email string) (*entity.User, error) {
	query := `SELECT id, email, password_hash FROM users WHERE email = $1`

	var user entity.User

	err := u.db.QueryRow(ctx, query, email).Scan(&user.ID, &user.Email, &user.PasswordHash)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, entity.ErrUserNotFound
		}
		return nil, err
	}
	return &user, nil
}
