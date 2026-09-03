package storage

import (
	"context"
	"errors"
	"fmt"

	"github.com/AlexeyKurlevsky/go-diploma/internal/models"
	"github.com/AlexeyKurlevsky/go-diploma/internal/storage"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type userRepo struct {
	pool *pgxpool.Pool
}

func NewUserRepository(pool *pgxpool.Pool) storage.UserRepository {
	return &userRepo{pool: pool}
}

func (r *userRepo) Create(ctx context.Context, user *models.User) error {
	user.ID = uuid.New()
	query := `INSERT INTO users (id, login, password_hash) VALUES ($1, $2, $3)`
	_, err := r.pool.Exec(ctx, query, user.ID, user.Login, user.PasswordHash)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return storage.ErrUserExists
		}
		return fmt.Errorf("create user: %w", err)
	}
	return nil
}

func (r *userRepo) FindByLogin(ctx context.Context, login string) (*models.User, error) {
	query := `SELECT id, login, password_hash FROM users WHERE login = $1`
	var user models.User
	err := r.pool.QueryRow(ctx, query, login).Scan(&user.ID, &user.Login, &user.PasswordHash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, storage.ErrUserNotFound
		}
		return nil, fmt.Errorf("find by login: %w", err)
	}
	return &user, nil
}
