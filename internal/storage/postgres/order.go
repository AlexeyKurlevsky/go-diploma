package storage

import (
	"context"
	"errors"
	"fmt"

	"github.com/AlexeyKurlevsky/go-diploma/internal/models"
	"github.com/AlexeyKurlevsky/go-diploma/internal/storage"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type orderRepo struct {
	pool *pgxpool.Pool
}

func NewOrderRepository(pool *pgxpool.Pool) storage.OrderRepository {
	return &orderRepo{pool: pool}
}

func (r *orderRepo) Create(ctx context.Context, order *models.Order) error {
	query := `
		INSERT INTO orders (user_id, number, status, uploaded_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`
	err := r.pool.QueryRow(ctx, query,
		order.UserID,
		order.Number,
		order.Status,
		order.UploadedAt,
		order.UpdatedAt,
	).Scan(&order.ID)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return storage.ErrOrderAlreadyExists
		}
		return fmt.Errorf("create order: %w", err)
	}
	return nil
}

func (r *orderRepo) FindByNumber(ctx context.Context, number string) (*models.Order, error) {
	query := `
		SELECT id, user_id, number, status, accrual, uploaded_at, updated_at
		FROM orders
		WHERE number = $1
	`
	order := &models.Order{}
	var accrual *float64 // nullable
	err := r.pool.QueryRow(ctx, query, number).Scan(
		&order.ID,
		&order.UserID,
		&order.Number,
		&order.Status,
		&accrual,
		&order.UploadedAt,
		&order.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, storage.ErrOrderNotFound
		}
		return nil, fmt.Errorf("find by number: %w", err)
	}
	order.Accrual = accrual // может быть nil
	return order, nil
}

func (r *orderRepo) FindByUserID(ctx context.Context, userID int64) ([]*models.Order, error) {
	query := `
		SELECT id, user_id, number, status, accrual, uploaded_at, updated_at
		FROM orders
		WHERE user_id = $1
		ORDER BY uploaded_at DESC
	`
	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("find by user: %w", err)
	}
	defer rows.Close()

	var orders []*models.Order
	for rows.Next() {
		order := &models.Order{}
		var accrual *float64
		err := rows.Scan(
			&order.ID,
			&order.UserID,
			&order.Number,
			&order.Status,
			&accrual,
			&order.UploadedAt,
			&order.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan order: %w", err)
		}
		order.Accrual = accrual
		orders = append(orders, order)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration: %w", err)
	}
	return orders, nil
}

func (r *orderRepo) UpdateStatusAndAccrual(ctx context.Context, orderID int64, status models.OrderStatus, accrual *float64) error {
	query := `
		UPDATE orders
		SET status = $1, accrual = $2, updated_at = NOW()
		WHERE id = $3
	`
	_, err := r.pool.Exec(ctx, query, status, accrual, orderID)
	if err != nil {
		return fmt.Errorf("update order: %w", err)
	}
	return nil
}

func (r *orderRepo) FindPendingOrders(ctx context.Context, limit int) ([]*models.Order, error) {
	query := `
		SELECT id, user_id, number, status, accrual, uploaded_at, updated_at
		FROM orders
		WHERE status IN ($1, $2)
		ORDER BY uploaded_at ASC
		LIMIT $3
	`
	rows, err := r.pool.Query(ctx, query, models.StatusNew, models.StatusProcessing, limit)
	if err != nil {
		return nil, fmt.Errorf("find pending: %w", err)
	}
	defer rows.Close()

	var orders []*models.Order
	for rows.Next() {
		order := &models.Order{}
		var accrual *float64
		err := rows.Scan(
			&order.ID,
			&order.UserID,
			&order.Number,
			&order.Status,
			&accrual,
			&order.UploadedAt,
			&order.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan pending: %w", err)
		}
		order.Accrual = accrual
		orders = append(orders, order)
	}
	return orders, nil
}
