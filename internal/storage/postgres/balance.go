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
	WITH order_agg AS (
		SELECT user_id, SUM(accrual) AS total_accrued
		FROM orders
		WHERE status = 'PROCESSED' AND user_id = $1
		GROUP BY user_id
	),
	withdrawal_agg AS (
		SELECT user_id, SUM(amount) AS total_withdrawn
		FROM withdrawals
		WHERE user_id = $1
		GROUP BY user_id
	)
	SELECT
		u.id AS user_id,
		COALESCE(oa.total_accrued, 0) - COALESCE(wa.total_withdrawn, 0) AS balance,
		COALESCE(oa.total_accrued, 0) AS total_accrued,
		COALESCE(wa.total_withdrawn, 0) AS total_withdrawn
	FROM users u
	LEFT JOIN order_agg oa ON u.id = oa.user_id
	LEFT JOIN withdrawal_agg wa ON u.id = wa.user_id
	WHERE u.id = $1;
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
