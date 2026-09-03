package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/AlexeyKurlevsky/go-diploma/internal/models"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	mock_storage "github.com/AlexeyKurlevsky/go-diploma/internal/storage/mocks"
)

func TestBalanceService_GetBalance(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockBalanceRepo := mock_storage.NewMockBalanceRepository(ctrl)
	mockWithdrawalRepo := mock_storage.NewMockWithdrawalRepository(ctrl)
	balanceService := NewBalanceService(mockBalanceRepo, mockWithdrawalRepo)
	ctx := context.Background()
	userID := uuid.New()

	t.Run("успешное получение баланса", func(t *testing.T) {
		expectedBalance := &models.UserBalance{
			UserID:         userID,
			Balance:        150.75,
			TotalAccrued:   200.00,
			TotalWithdrawn: 49.25,
		}
		mockBalanceRepo.EXPECT().
			GetByUserID(ctx, userID).
			Return(expectedBalance, nil)

		current, withdrawn, err := balanceService.GetBalance(ctx, userID)
		assert.NoError(t, err)
		assert.Equal(t, expectedBalance.Balance, current)
		assert.Equal(t, expectedBalance.TotalWithdrawn, withdrawn)
	})

	t.Run("пользователь без записей – возвращаем нулевой баланс", func(t *testing.T) {
		zeroBalance := &models.UserBalance{
			UserID:         userID,
			Balance:        0,
			TotalAccrued:   0,
			TotalWithdrawn: 0,
		}
		mockBalanceRepo.EXPECT().
			GetByUserID(ctx, userID).
			Return(zeroBalance, nil)

		current, withdrawn, err := balanceService.GetBalance(ctx, userID)
		assert.NoError(t, err)
		assert.Equal(t, 0.0, current)
		assert.Equal(t, 0.0, withdrawn)
	})

	t.Run("ошибка репозитория", func(t *testing.T) {
		mockBalanceRepo.EXPECT().
			GetByUserID(ctx, userID).
			Return(nil, errors.New("db error"))

		current, withdrawn, err := balanceService.GetBalance(ctx, userID)
		assert.Error(t, err)
		assert.Equal(t, 0.0, current)
		assert.Equal(t, 0.0, withdrawn)
	})
}

func TestBalanceService_Withdraw(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockBalanceRepo := mock_storage.NewMockBalanceRepository(ctrl)
	mockWithdrawalRepo := mock_storage.NewMockWithdrawalRepository(ctrl)
	balanceService := NewBalanceService(mockBalanceRepo, mockWithdrawalRepo)
	ctx := context.Background()
	userID := uuid.New()

	t.Run("успешное списание", func(t *testing.T) {
		orderNumber := "12345678903" // валидный по Луне
		amount := 50.0
		balance := &models.UserBalance{
			UserID:  userID,
			Balance: 100.0,
		}
		mockBalanceRepo.EXPECT().
			GetByUserID(ctx, userID).
			Return(balance, nil)

		mockWithdrawalRepo.EXPECT().
			Create(ctx, gomock.Any()).
			DoAndReturn(func(_ context.Context, w *models.Withdrawal) error {
				assert.Equal(t, userID, w.UserID)
				assert.Equal(t, orderNumber, w.OrderNumber)
				assert.Equal(t, amount, w.Amount)
				assert.NotZero(t, w.ProcessedAt)
				w.ID = uuid.New()
				return nil
			})

		err := balanceService.Withdraw(ctx, userID, orderNumber, amount)
		assert.NoError(t, err)
	})

	t.Run("недостаточно средств", func(t *testing.T) {
		orderNumber := "12345678903"
		amount := 150.0
		balance := &models.UserBalance{
			UserID:  userID,
			Balance: 100.0,
		}
		mockBalanceRepo.EXPECT().
			GetByUserID(ctx, userID).
			Return(balance, nil)

		err := balanceService.Withdraw(ctx, userID, orderNumber, amount)
		assert.ErrorIs(t, err, ErrInsufficientFunds)
	})

	t.Run("невалидный номер заказа", func(t *testing.T) {
		orderNumber := "123" // не проходит Луну
		amount := 50.0
		err := balanceService.Withdraw(ctx, userID, orderNumber, amount)
		assert.ErrorIs(t, err, ErrInvalidOrderNumber)
	})

	t.Run("сумма меньше или равна нулю", func(t *testing.T) {
		orderNumber := "12345678903"
		err := balanceService.Withdraw(ctx, userID, orderNumber, 0)
		assert.ErrorIs(t, err, ErrInvalidAmount)

		err = balanceService.Withdraw(ctx, userID, orderNumber, -10)
		assert.ErrorIs(t, err, ErrInvalidAmount)
	})

	t.Run("ошибка репозитория при получении баланса", func(t *testing.T) {
		orderNumber := "12345678903"
		amount := 50.0
		mockBalanceRepo.EXPECT().
			GetByUserID(ctx, userID).
			Return(nil, errors.New("db error"))

		err := balanceService.Withdraw(ctx, userID, orderNumber, amount)
		assert.Error(t, err)
	})

	t.Run("ошибка создания списания", func(t *testing.T) {
		orderNumber := "12345678903"
		amount := 50.0
		balance := &models.UserBalance{
			UserID:  userID,
			Balance: 100.0,
		}
		mockBalanceRepo.EXPECT().
			GetByUserID(ctx, userID).
			Return(balance, nil)

		mockWithdrawalRepo.EXPECT().
			Create(ctx, gomock.Any()).
			Return(errors.New("insert failed"))

		err := balanceService.Withdraw(ctx, userID, orderNumber, amount)
		assert.Error(t, err)
	})
}

func TestBalanceService_GetWithdrawals(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockBalanceRepo := mock_storage.NewMockBalanceRepository(ctrl)
	mockWithdrawalRepo := mock_storage.NewMockWithdrawalRepository(ctrl)
	balanceService := NewBalanceService(mockBalanceRepo, mockWithdrawalRepo)
	ctx := context.Background()
	userID := uuid.New()

	t.Run("успешное получение списка списаний", func(t *testing.T) {
		expected := []*models.Withdrawal{
			{
				ID:          uuid.New(),
				UserID:      userID,
				OrderNumber: "12345678903",
				Amount:      50.0,
				ProcessedAt: time.Now(),
			},
			{
				ID:          uuid.New(),
				UserID:      userID,
				OrderNumber: "98765432105",
				Amount:      30.0,
				ProcessedAt: time.Now().Add(-time.Hour),
			},
		}
		mockWithdrawalRepo.EXPECT().
			FindByUserID(ctx, userID).
			Return(expected, nil)

		result, err := balanceService.GetWithdrawals(ctx, userID)
		assert.NoError(t, err)
		assert.Len(t, result, 2)
		assert.Equal(t, expected, result)
	})

	t.Run("нет списаний – пустой список", func(t *testing.T) {
		mockWithdrawalRepo.EXPECT().
			FindByUserID(ctx, userID).
			Return([]*models.Withdrawal{}, nil)

		result, err := balanceService.GetWithdrawals(ctx, userID)
		assert.NoError(t, err)
		assert.Empty(t, result)
	})

	t.Run("ошибка репозитория", func(t *testing.T) {
		mockWithdrawalRepo.EXPECT().
			FindByUserID(ctx, userID).
			Return(nil, errors.New("db error"))

		result, err := balanceService.GetWithdrawals(ctx, userID)
		assert.Error(t, err)
		assert.Nil(t, result)
	})
}
