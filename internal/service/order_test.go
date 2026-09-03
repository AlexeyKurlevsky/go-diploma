package service

import (
	"context"
	"testing"

	"github.com/AlexeyKurlevsky/go-diploma/internal/client"
	"github.com/AlexeyKurlevsky/go-diploma/internal/models"
	"github.com/AlexeyKurlevsky/go-diploma/internal/storage"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	mock_client "github.com/AlexeyKurlevsky/go-diploma/internal/client/mocks"

	mock_storage "github.com/AlexeyKurlevsky/go-diploma/internal/storage/mocks"
)

func TestOrderService_UploadOrder(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockOrderRepo := mock_storage.NewMockOrderRepository(ctrl)
	mockAccrualClient := mock_client.NewMockAccrualClient(ctrl)
	mockBalanceRepo := mock_storage.NewMockBalanceRepository(ctrl)

	orderService := NewOrderService(mockOrderRepo, mockAccrualClient, mockBalanceRepo)
	ctx := context.Background()
	userID := uuid.New()

	t.Run("successful upload", func(t *testing.T) {
		orderNumber := "12345678903"
		mockOrderRepo.EXPECT().
			FindByNumber(ctx, orderNumber).
			Return(nil, storage.ErrOrderNotFound)
		mockOrderRepo.EXPECT().
			Create(ctx, gomock.Any()).
			Return(nil)
		// Разрешаем вызов CheckOrder (асинхронный)
		mockAccrualClient.EXPECT().
			CheckOrder(gomock.Any(), orderNumber).
			Return(&client.AccrualResponse{
				Order:   orderNumber,
				Status:  "REGISTERED",
				Accrual: nil,
			}, nil).
			AnyTimes()

		order, err := orderService.UploadOrder(ctx, userID, orderNumber)
		require.NoError(t, err)
		assert.Equal(t, orderNumber, order.Number)
		assert.Equal(t, models.StatusNew, order.Status)
	})

	t.Run("invalid order number", func(t *testing.T) {
		_, err := orderService.UploadOrder(ctx, userID, "123")
		assert.ErrorIs(t, err, ErrInvalidOrderNumber)
	})

	t.Run("already uploaded by same user", func(t *testing.T) {
		orderNumber := "12345678903"
		existing := &models.Order{UserID: userID, Number: orderNumber}
		mockOrderRepo.EXPECT().
			FindByNumber(ctx, orderNumber).
			Return(existing, nil)

		_, err := orderService.UploadOrder(ctx, userID, orderNumber)
		assert.ErrorIs(t, err, ErrOrderAlreadyUploadedByUser)
	})

	t.Run("already uploaded by another user", func(t *testing.T) {
		orderNumber := "12345678903"
		otherUser := uuid.New()
		existing := &models.Order{UserID: otherUser, Number: orderNumber}
		mockOrderRepo.EXPECT().
			FindByNumber(ctx, orderNumber).
			Return(existing, nil)

		_, err := orderService.UploadOrder(ctx, userID, orderNumber)
		assert.ErrorIs(t, err, ErrOrderConflict)
	})
}
