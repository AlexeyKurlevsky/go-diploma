package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/AlexeyKurlevsky/go-diploma/internal/models"
	"github.com/AlexeyKurlevsky/go-diploma/internal/storage"

	"github.com/jackc/pgx/v5/pgxpool"
)

type balanceRepo struct {
	pool *pgxpool.Pool
}

func NewBalanceRepository(pool *pgxpool.Pool) storage.BalanceRepository {
	return &balanceRepo{pool: pool}
}

func (r *balanceRepo) GetByUserID(ctx context.Context, userID int64) (*models.UserBalance, error) {
	query := `
		SELECT user_id, balance, total_accrued, total_withdrawn
		FROM user_balance
		WHERE user_id = $1
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
			// Если пользователь ещё не имеет записей, возвращаем нулевой баланс
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

func (r *balanceRepo) RefreshMaterializedView(ctx context.Context) error {
	query := `REFRESH MATERIALIZED VIEW CONCURRENTLY user_balance`
	_, err := r.pool.Exec(ctx, query)
	if err != nil {
		return fmt.Errorf("refresh mv: %w", err)
	}
	return nil
}
