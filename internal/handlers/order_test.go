package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/AlexeyKurlevsky/go-diploma/internal/middleware"
	"github.com/AlexeyKurlevsky/go-diploma/internal/models"
	"github.com/AlexeyKurlevsky/go-diploma/internal/service"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	mock_service "github.com/AlexeyKurlevsky/go-diploma/internal/service/mocks"
)

func TestOrderHandler_UploadOrder(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockOrderService := mock_service.NewMockOrderService(ctrl)
	handler := NewOrderHandler(mockOrderService)

	t.Run("успешная загрузка нового заказа – 202", func(t *testing.T) {
		userID := uuid.New()
		orderNumber := "12345678903"
		expectedOrder := &models.Order{
			ID:         uuid.New(),
			UserID:     userID,
			Number:     orderNumber,
			Status:     models.StatusNew,
			UploadedAt: time.Now(),
			UpdatedAt:  time.Now(),
		}

		mockOrderService.EXPECT().
			UploadOrder(gomock.Any(), userID, orderNumber).
			Return(expectedOrder, nil)

		req := httptest.NewRequest(http.MethodPost, "/api/user/orders", bytes.NewReader([]byte(orderNumber)))
		req.Header.Set("Content-Type", "text/plain")
		ctx := context.WithValue(req.Context(), middleware.UserIDKey, userID)
		req = req.WithContext(ctx)

		w := httptest.NewRecorder()
		handler.UploadOrder(w, req)

		assert.Equal(t, http.StatusAccepted, w.Code)
	})

	t.Run("заказ уже загружен этим пользователем – 200", func(t *testing.T) {
		userID := uuid.New()
		orderNumber := "12345678903"
		existingOrder := &models.Order{
			ID:         uuid.New(),
			UserID:     userID,
			Number:     orderNumber,
			Status:     models.StatusProcessed,
			Accrual:    ptrFloat64(500.0),
			UploadedAt: time.Now(),
			UpdatedAt:  time.Now(),
		}

		mockOrderService.EXPECT().
			UploadOrder(gomock.Any(), userID, orderNumber).
			Return(existingOrder, service.ErrOrderAlreadyUploadedByUser)

		req := httptest.NewRequest(http.MethodPost, "/api/user/orders", bytes.NewReader([]byte(orderNumber)))
		req.Header.Set("Content-Type", "text/plain")
		ctx := context.WithValue(req.Context(), middleware.UserIDKey, userID)
		req = req.WithContext(ctx)

		w := httptest.NewRecorder()
		handler.UploadOrder(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("заказ уже загружен другим пользователем – 409", func(t *testing.T) {
		userID := uuid.New()
		orderNumber := "12345678903"

		mockOrderService.EXPECT().
			UploadOrder(gomock.Any(), userID, orderNumber).
			Return(nil, service.ErrOrderConflict)

		req := httptest.NewRequest(http.MethodPost, "/api/user/orders", bytes.NewReader([]byte(orderNumber)))
		req.Header.Set("Content-Type", "text/plain")
		ctx := context.WithValue(req.Context(), middleware.UserIDKey, userID)
		req = req.WithContext(ctx)

		w := httptest.NewRecorder()
		handler.UploadOrder(w, req)

		assert.Equal(t, http.StatusConflict, w.Code)
	})

	t.Run("невалидный номер заказа – 422", func(t *testing.T) {
		userID := uuid.New()
		orderNumber := "123" // не проходит Луну

		mockOrderService.EXPECT().
			UploadOrder(gomock.Any(), userID, orderNumber).
			Return(nil, service.ErrInvalidOrderNumber)

		req := httptest.NewRequest(http.MethodPost, "/api/user/orders", bytes.NewReader([]byte(orderNumber)))
		req.Header.Set("Content-Type", "text/plain")
		ctx := context.WithValue(req.Context(), middleware.UserIDKey, userID)
		req = req.WithContext(ctx)

		w := httptest.NewRecorder()
		handler.UploadOrder(w, req)

		assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	})

	t.Run("отсутствие userID – 401", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/user/orders", bytes.NewReader([]byte("12345678903")))
		req.Header.Set("Content-Type", "text/plain")
		w := httptest.NewRecorder()
		handler.UploadOrder(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("пустое тело запроса – 400", func(t *testing.T) {
		userID := uuid.New()
		req := httptest.NewRequest(http.MethodPost, "/api/user/orders", bytes.NewReader([]byte("")))
		req.Header.Set("Content-Type", "text/plain")
		ctx := context.WithValue(req.Context(), middleware.UserIDKey, userID)
		req = req.WithContext(ctx)

		w := httptest.NewRecorder()
		handler.UploadOrder(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("ошибка сервиса – 500", func(t *testing.T) {
		userID := uuid.New()
		orderNumber := "12345678903"

		mockOrderService.EXPECT().
			UploadOrder(gomock.Any(), userID, orderNumber).
			Return(nil, errors.New("unexpected error"))

		req := httptest.NewRequest(http.MethodPost, "/api/user/orders", bytes.NewReader([]byte(orderNumber)))
		req.Header.Set("Content-Type", "text/plain")
		ctx := context.WithValue(req.Context(), middleware.UserIDKey, userID)
		req = req.WithContext(ctx)

		w := httptest.NewRecorder()
		handler.UploadOrder(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestOrderHandler_GetOrders(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockOrderService := mock_service.NewMockOrderService(ctrl)
	handler := NewOrderHandler(mockOrderService)

	t.Run("успешное получение списка заказов – 200", func(t *testing.T) {
		userID := uuid.New()
		now := time.Now()
		orders := []*models.Order{
			{
				ID:         uuid.New(),
				UserID:     userID,
				Number:     "12345678903",
				Status:     models.StatusProcessed,
				Accrual:    ptrFloat64(500.0),
				UploadedAt: now,
				UpdatedAt:  now,
			},
			{
				ID:         uuid.New(),
				UserID:     userID,
				Number:     "98765432105",
				Status:     models.StatusProcessing,
				Accrual:    nil,
				UploadedAt: now.Add(-time.Hour),
				UpdatedAt:  now.Add(-time.Hour),
			},
		}

		mockOrderService.EXPECT().
			GetUserOrders(gomock.Any(), userID).
			Return(orders, nil)

		req := httptest.NewRequest(http.MethodGet, "/api/user/orders", nil)
		ctx := context.WithValue(req.Context(), middleware.UserIDKey, userID)
		req = req.WithContext(ctx)

		w := httptest.NewRecorder()
		handler.GetOrders(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

		var resp []map[string]interface{}
		err := json.NewDecoder(w.Body).Decode(&resp)
		require.NoError(t, err)
		assert.Len(t, resp, 2)

		// Проверяем первый заказ
		first := resp[0]
		assert.Equal(t, orders[0].Number, first["number"])
		assert.Equal(t, string(orders[0].Status), first["status"])
		assert.Equal(t, *orders[0].Accrual, first["accrual"])
		assert.Equal(t, orders[0].UploadedAt.Format(time.RFC3339), first["uploaded_at"])

		// Второй заказ без начисления
		second := resp[1]
		assert.Equal(t, orders[1].Number, second["number"])
		assert.Equal(t, string(orders[1].Status), second["status"])
		_, hasAccrual := second["accrual"]
		assert.False(t, hasAccrual) // accrual не должно быть в ответе
		assert.Equal(t, orders[1].UploadedAt.Format(time.RFC3339), second["uploaded_at"])
	})

	t.Run("пустой список – 204", func(t *testing.T) {
		userID := uuid.New()
		mockOrderService.EXPECT().
			GetUserOrders(gomock.Any(), userID).
			Return([]*models.Order{}, nil)

		req := httptest.NewRequest(http.MethodGet, "/api/user/orders", nil)
		ctx := context.WithValue(req.Context(), middleware.UserIDKey, userID)
		req = req.WithContext(ctx)

		w := httptest.NewRecorder()
		handler.GetOrders(w, req)

		assert.Equal(t, http.StatusNoContent, w.Code)
	})

	t.Run("отсутствие userID – 401", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/user/orders", nil)
		w := httptest.NewRecorder()
		handler.GetOrders(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("ошибка сервиса – 500", func(t *testing.T) {
		userID := uuid.New()
		mockOrderService.EXPECT().
			GetUserOrders(gomock.Any(), userID).
			Return(nil, errors.New("db error"))

		req := httptest.NewRequest(http.MethodGet, "/api/user/orders", nil)
		ctx := context.WithValue(req.Context(), middleware.UserIDKey, userID)
		req = req.WithContext(ctx)

		w := httptest.NewRecorder()
		handler.GetOrders(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

// Вспомогательная функция для указателя на float64
func ptrFloat64(v float64) *float64 {
	return &v
}
