package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/AlexeyKurlevsky/go-diploma/internal/models"
	"github.com/AlexeyKurlevsky/go-diploma/internal/storage"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type balanceRepo struct {
	pool *pgxpool.Pool
}

func NewBalanceRepository(pool *pgxpool.Pool) storage.BalanceRepository {
	return &balanceRepo{pool: pool}
}

func (r *balanceRepo) GetByUserID(ctx context.Context, userID uuid.UUID) (*models.UserBalance, error) {
	query := `
	SELECT
    $1 AS user_id,
    COALESCE(
        (SELECT SUM(accrual) FROM orders WHERE user_id = $1 AND status = 'PROCESSED'), 0
    ) - COALESCE(
        (SELECT SUM(amount) FROM withdrawals WHERE user_id = $1), 0
    ) AS balance,
    COALESCE(
        (SELECT SUM(accrual) FROM orders WHERE user_id = $1 AND status = 'PROCESSED'), 0
    ) AS total_accrued,
    COALESCE(
        (SELECT SUM(amount) FROM withdrawals WHERE user_id = $1), 0
    ) AS total_withdrawn;
    `
	var balance models.UserBalance
	err := r.pool.QueryRow(ctx, query, userID).Scan(
		&balance.UserID,
		&balance.Balance,
		&balance.TotalAccrued,
		&balance.TotalWithdrawn,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return &models.UserBalance{
				UserID:         userID,
				Balance:        0,
				TotalAccrued:   0,
				TotalWithdrawn: 0,
			}, nil
		}
		return nil, fmt.Errorf("get balance: %w", err)
	}
	return &balance, nil
}
