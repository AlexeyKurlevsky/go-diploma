package storage

import (
	"context"
	"fmt"

	"github.com/AlexeyKurlevsky/go-diploma/internal/models"

	"github.com/AlexeyKurlevsky/go-diploma/internal/storage"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type withdrawalRepo struct {
	pool *pgxpool.Pool
}

func NewWithdrawalRepository(pool *pgxpool.Pool) storage.WithdrawalRepository {
	return &withdrawalRepo{pool: pool}
}

func (r *withdrawalRepo) Create(ctx context.Context, withdrawal *models.Withdrawal) error {
	withdrawal.ID = uuid.New()
	query := `
        INSERT INTO withdrawals (id, user_id, order_number, amount, processed_at)
        VALUES ($1, $2, $3, $4, $5)
    `
	_, err := r.pool.Exec(ctx, query,
		withdrawal.ID,
		withdrawal.UserID,
		withdrawal.OrderNumber,
		withdrawal.Amount,
		withdrawal.ProcessedAt,
	)
	if err != nil {
		return fmt.Errorf("create withdrawal: %w", err)
	}
	return nil
}

func (r *withdrawalRepo) FindByUserID(ctx context.Context, userID uuid.UUID) ([]*models.Withdrawal, error) {
	query := `
        SELECT id, user_id, order_number, amount, processed_at
        FROM withdrawals
        WHERE user_id = $1
        ORDER BY processed_at DESC
    `
	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("find by user: %w", err)
	}
	defer rows.Close()

	var withdrawals []*models.Withdrawal
	for rows.Next() {
		w := &models.Withdrawal{}
		err := rows.Scan(&w.ID, &w.UserID, &w.OrderNumber, &w.Amount, &w.ProcessedAt)
		if err != nil {
			return nil, fmt.Errorf("scan withdrawal: %w", err)
		}
		withdrawals = append(withdrawals, w)
	}
	return withdrawals, rows.Err()
}

func (r *withdrawalRepo) SumByUser(ctx context.Context, userID uuid.UUID) (float64, error) {
	query := `SELECT COALESCE(SUM(amount), 0) FROM withdrawals WHERE user_id = $1`
	var sum float64
	err := r.pool.QueryRow(ctx, query, userID).Scan(&sum)
	if err != nil {
		return 0, fmt.Errorf("sum withdrawals: %w", err)
	}
	return sum, nil
}
