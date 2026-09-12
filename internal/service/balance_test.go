package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/AlexeyKurlevsky/go-diploma/internal/models"
	mock_storage "github.com/AlexeyKurlevsky/go-diploma/internal/storage/mocks"
)

func TestBalanceService_GetBalance(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockBalanceRepo := mock_storage.NewMockBalanceRepository(ctrl)
	mockWithdrawalRepo := mock_storage.NewMockWithdrawalRepository(ctrl)
	svc := NewBalanceService(mockBalanceRepo, mockWithdrawalRepo)
	ctx := context.Background()
	userID := uuid.New()

	t.Run("успешное получение баланса", func(t *testing.T) {
		expected := &models.UserBalance{
			UserID:         userID,
			Balance:        150.75,
			TotalAccrued:   200.00,
			TotalWithdrawn: 49.25,
		}
		mockBalanceRepo.EXPECT().
			GetByUserID(ctx, userID).
			Return(expected, nil)

		current, withdrawn, err := svc.GetBalance(ctx, userID)
		assert.NoError(t, err)
		assert.Equal(t, expected.Balance, current)
		assert.Equal(t, expected.TotalWithdrawn, withdrawn)
	})

	t.Run("пользователь без записей – нулевой баланс", func(t *testing.T) {
		zero := &models.UserBalance{UserID: userID}
		mockBalanceRepo.EXPECT().
			GetByUserID(ctx, userID).
			Return(zero, nil)

		current, withdrawn, err := svc.GetBalance(ctx, userID)
		assert.NoError(t, err)
		assert.Equal(t, 0.0, current)
		assert.Equal(t, 0.0, withdrawn)
	})

	t.Run("ошибка репозитория", func(t *testing.T) {
		mockBalanceRepo.EXPECT().
			GetByUserID(ctx, userID).
			Return(nil, errors.New("db error"))

		current, withdrawn, err := svc.GetBalance(ctx, userID)
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
	svc := NewBalanceService(mockBalanceRepo, mockWithdrawalRepo)
	ctx := context.Background()
	userID := uuid.New()

	t.Run("успешное списание", func(t *testing.T) {
		orderNumber := "12345678903" // валидный по Луне
		amount := 50.0

		mockWithdrawalRepo.EXPECT().
			CreateWithBalanceCheck(ctx, gomock.Any()).
			DoAndReturn(func(_ context.Context, w *models.Withdrawal) error {
				assert.Equal(t, userID, w.UserID)
				assert.Equal(t, orderNumber, w.OrderNumber)
				assert.Equal(t, amount, w.Amount)
				assert.False(t, w.ProcessedAt.IsZero())
				w.ID = uuid.New()
				return nil
			})

		err := svc.Withdraw(ctx, userID, orderNumber, amount)
		assert.NoError(t, err)
	})

	t.Run("недостаточно средств", func(t *testing.T) {
		orderNumber := "12345678903"
		amount := 150.0

		mockWithdrawalRepo.EXPECT().
			CreateWithBalanceCheck(ctx, gomock.Any()).
			Return(ErrInsufficientFunds)

		err := svc.Withdraw(ctx, userID, orderNumber, amount)
		assert.ErrorIs(t, err, ErrInsufficientFunds)
	})

	t.Run("невалидный номер заказа", func(t *testing.T) {
		err := svc.Withdraw(ctx, userID, "123", 50.0)
		assert.ErrorIs(t, err, ErrInvalidOrderNumber)
	})

	t.Run("номер заказа не число", func(t *testing.T) {
		err := svc.Withdraw(ctx, userID, "abcdef", 50.0)
		assert.ErrorIs(t, err, ErrInvalidOrderNumber)
	})

	t.Run("сумма меньше или равна нулю", func(t *testing.T) {
		err := svc.Withdraw(ctx, userID, "12345678903", 0)
		assert.ErrorIs(t, err, ErrInvalidAmount)

		err = svc.Withdraw(ctx, userID, "12345678903", -10)
		assert.ErrorIs(t, err, ErrInvalidAmount)
	})

	t.Run("ошибка репозитория при списании", func(t *testing.T) {
		orderNumber := "12345678903"
		amount := 50.0

		mockWithdrawalRepo.EXPECT().
			CreateWithBalanceCheck(ctx, gomock.Any()).
			Return(errors.New("insert failed"))

		err := svc.Withdraw(ctx, userID, orderNumber, amount)
		assert.Error(t, err)
	})
}

func TestBalanceService_GetWithdrawals(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockBalanceRepo := mock_storage.NewMockBalanceRepository(ctrl)
	mockWithdrawalRepo := mock_storage.NewMockWithdrawalRepository(ctrl)
	svc := NewBalanceService(mockBalanceRepo, mockWithdrawalRepo)
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

		result, err := svc.GetWithdrawals(ctx, userID)
		assert.NoError(t, err)
		require.Len(t, result, 2)
		assert.Equal(t, expected, result)
	})

	t.Run("нет списаний – пустой список", func(t *testing.T) {
		mockWithdrawalRepo.EXPECT().
			FindByUserID(ctx, userID).
			Return([]*models.Withdrawal{}, nil)

		result, err := svc.GetWithdrawals(ctx, userID)
		assert.NoError(t, err)
		assert.Empty(t, result)
	})

	t.Run("ошибка репозитория", func(t *testing.T) {
		mockWithdrawalRepo.EXPECT().
			FindByUserID(ctx, userID).
			Return(nil, errors.New("db error"))

		result, err := svc.GetWithdrawals(ctx, userID)
		assert.Error(t, err)
		assert.Nil(t, result)
	})
}
