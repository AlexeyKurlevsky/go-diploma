package service

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/AlexeyKurlevsky/go-diploma/internal/models"
	"github.com/AlexeyKurlevsky/go-diploma/internal/storage"
	"github.com/theplant/luhn"
)

type BalanceService interface {
	GetBalance(ctx context.Context, userID int64) (current float64, withdrawn float64, err error)
	Withdraw(ctx context.Context, userID int64, orderNumber string, amount float64) error
	GetWithdrawals(ctx context.Context, userID int64) ([]*models.Withdrawal, error)
	RefreshBalanceView(ctx context.Context) error
}

type balanceService struct {
	balanceRepo    storage.BalanceRepository
	withdrawalRepo storage.WithdrawalRepository
}

func NewBalanceService(balanceRepo storage.BalanceRepository, withdrawalRepo storage.WithdrawalRepository) BalanceService {
	return &balanceService{
		balanceRepo:    balanceRepo,
		withdrawalRepo: withdrawalRepo,
	}
}

func (s *balanceService) GetBalance(ctx context.Context, userID int64) (float64, float64, error) {
	balance, err := s.balanceRepo.GetByUserID(ctx, userID)
	if err != nil {
		return 0, 0, fmt.Errorf("get balance: %w", err)
	}
	return balance.Balance, balance.TotalWithdrawn, nil
}

func (s *balanceService) Withdraw(ctx context.Context, userID int64, orderNumber string, amount float64) error {
	// 1. Валидация номера заказа (алгоритм Луна)
	num, err := strconv.Atoi(orderNumber)
	if err != nil {
		fmt.Println("Error during conversion:", err)
		return ErrInvalidOrderNumber
	}
	if !luhn.Valid(num) {
		return ErrInvalidOrderNumber
	}
	if amount <= 0 {
		return ErrInvalidAmount
	}

	// 2. Проверяем, достаточно ли средств (из MV)
	balance, err := s.balanceRepo.GetByUserID(ctx, userID)
	if err != nil {
		return fmt.Errorf("get balance: %w", err)
	}
	if balance.Balance < amount {
		return ErrInsufficientFunds
	}

	// 3. Создаём запись списания
	withdrawal := &models.Withdrawal{
		UserID:      userID,
		OrderNumber: orderNumber,
		Amount:      amount,
		ProcessedAt: time.Now(),
	}
	if err := s.withdrawalRepo.Create(ctx, withdrawal); err != nil {
		return fmt.Errorf("create withdrawal: %w", err)
	}

	// 4. Обновляем материализованное представление (асинхронно или синхронно)
	// Рекомендуется обновлять асинхронно, чтобы не блокировать ответ клиента
	// Можно либо использовать горутину, либо полагаться на фоновый воркер
	go func() {
		ctxBg := context.Background()
		if err := s.balanceRepo.RefreshMaterializedView(ctxBg); err != nil {
			// Логируем ошибку, но не возвращаем клиенту
			// log.Printf("failed to refresh balance view: %v", err)
		}
	}()

	return nil
}

func (s *balanceService) GetWithdrawals(ctx context.Context, userID int64) ([]*models.Withdrawal, error) {
	return s.withdrawalRepo.FindByUserID(ctx, userID)
}

func (s *balanceService) RefreshBalanceView(ctx context.Context) error {
	return s.balanceRepo.RefreshMaterializedView(ctx)
}
