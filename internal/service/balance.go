package service

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/AlexeyKurlevsky/go-diploma/internal/models"
	"github.com/AlexeyKurlevsky/go-diploma/internal/storage"
	"github.com/theplant/luhn"

	"github.com/google/uuid"
)

type BalanceService interface {
	GetBalance(ctx context.Context, userID uuid.UUID) (current float64, withdrawn float64, err error)
	Withdraw(ctx context.Context, userID uuid.UUID, orderNumber string, amount float64) error
	GetWithdrawals(ctx context.Context, userID uuid.UUID) ([]*models.Withdrawal, error)
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

func (s *balanceService) GetBalance(ctx context.Context, userID uuid.UUID) (float64, float64, error) {
	balance, err := s.balanceRepo.GetByUserID(ctx, userID)
	if err != nil {
		return 0, 0, fmt.Errorf("get balance: %w", err)
	}
	return balance.Balance, balance.TotalWithdrawn, nil
}

func (s *balanceService) Withdraw(ctx context.Context, userID uuid.UUID, orderNumber string, amount float64) error {
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
	balance, err := s.balanceRepo.GetByUserID(ctx, userID)
	if err != nil {
		return fmt.Errorf("get balance: %w", err)
	}
	if balance.Balance < amount {
		return ErrInsufficientFunds
	}
	withdrawal := &models.Withdrawal{
		UserID:      userID,
		OrderNumber: orderNumber,
		Amount:      amount,
		ProcessedAt: time.Now(),
	}
	if err := s.withdrawalRepo.Create(ctx, withdrawal); err != nil {
		return fmt.Errorf("create withdrawal: %w", err)
	}
	return nil
}

func (s *balanceService) GetWithdrawals(ctx context.Context, userID uuid.UUID) ([]*models.Withdrawal, error) {
	return s.withdrawalRepo.FindByUserID(ctx, userID)
}
