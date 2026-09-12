package storage

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/AlexeyKurlevsky/go-diploma/internal/models"
	"github.com/AlexeyKurlevsky/go-diploma/internal/service"
	"github.com/AlexeyKurlevsky/go-diploma/internal/storage"
)

type withdrawalRepo struct {
	pool *pgxpool.Pool
}

func NewWithdrawalRepository(pool *pgxpool.Pool) storage.WithdrawalRepository {
	return &withdrawalRepo{pool: pool}
}

func (r *withdrawalRepo) CreateWithBalanceCheck(ctx context.Context, w *models.Withdrawal) error {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// 1. Блокируем пользователя — сериализует все списания одного user_id.
	var lockedID uuid.UUID
	err = tx.QueryRow(ctx,
		`SELECT id FROM users WHERE id = $1 FOR UPDATE`,
		w.UserID,
	).Scan(&lockedID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return storage.ErrUserNotFound
		}
		return fmt.Errorf("lock user: %w", err)
	}

	// 2. Вычисляем баланс внутри транзакции (после блокировки).
	var balance float64
	err = tx.QueryRow(ctx, `
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
			COALESCE(oa.total_accrued, 0) - COALESCE(wa.total_withdrawn, 0)
		FROM users u
		LEFT JOIN order_agg oa ON u.id = oa.user_id
		LEFT JOIN withdrawal_agg wa ON u.id = wa.user_id
		WHERE u.id = $1
	`, w.UserID).Scan(&balance)
	if err != nil {
		return fmt.Errorf("calc balance: %w", err)
	}

	if balance < w.Amount {
		return service.ErrInsufficientFunds
	}

	// 3. Вставляем списание.
	if w.ID == uuid.Nil {
		w.ID = uuid.New()
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO withdrawals (id, user_id, order_number, amount, processed_at)
		VALUES ($1, $2, $3, $4, $5)
	`, w.ID, w.UserID, w.OrderNumber, w.Amount, w.ProcessedAt)
	if err != nil {
		return fmt.Errorf("insert withdrawal: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	return nil
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
