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

func TestBalanceHandler_GetBalance(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockBalanceService := mock_service.NewMockBalanceService(ctrl)
	handler := NewBalanceHandler(mockBalanceService)

	t.Run("успешное получение баланса", func(t *testing.T) {
		userID := uuid.New()
		expectedCurrent := 150.75
		expectedWithdrawn := 49.25

		mockBalanceService.EXPECT().
			GetBalance(gomock.Any(), userID).
			Return(expectedCurrent, expectedWithdrawn, nil)

		req := httptest.NewRequest(http.MethodGet, "/api/user/balance", nil)
		ctx := req.Context()
		ctx = context.WithValue(ctx, middleware.UserIDKey, userID)
		req = req.WithContext(ctx)

		w := httptest.NewRecorder()
		handler.GetBalance(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

		var resp map[string]interface{}
		err := json.NewDecoder(w.Body).Decode(&resp)
		require.NoError(t, err)
		assert.Equal(t, expectedCurrent, resp["current"])
		assert.Equal(t, expectedWithdrawn, resp["withdrawn"])
	})

	t.Run("отсутствие userID в контексте – 401", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/user/balance", nil)
		w := httptest.NewRecorder()
		handler.GetBalance(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("ошибка сервиса – 500", func(t *testing.T) {
		userID := uuid.New()
		mockBalanceService.EXPECT().
			GetBalance(gomock.Any(), userID).
			Return(0.0, 0.0, errors.New("db error"))

		req := httptest.NewRequest(http.MethodGet, "/api/user/balance", nil)
		ctx := context.WithValue(req.Context(), middleware.UserIDKey, userID)
		req = req.WithContext(ctx)

		w := httptest.NewRecorder()
		handler.GetBalance(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestBalanceHandler_Withdraw(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockBalanceService := mock_service.NewMockBalanceService(ctrl)
	handler := NewBalanceHandler(mockBalanceService)

	t.Run("успешное списание", func(t *testing.T) {
		userID := uuid.New()
		orderNumber := "12345678903"
		amount := 50.0

		mockBalanceService.EXPECT().
			Withdraw(gomock.Any(), userID, orderNumber, amount).
			Return(nil)

		body := map[string]interface{}{
			"order": orderNumber,
			"sum":   amount,
		}
		jsonBody, _ := json.Marshal(body)

		req := httptest.NewRequest(http.MethodPost, "/api/user/balance/withdraw", bytes.NewReader(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		ctx := context.WithValue(req.Context(), middleware.UserIDKey, userID)
		req = req.WithContext(ctx)

		w := httptest.NewRecorder()
		handler.Withdraw(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("отсутствие userID – 401", func(t *testing.T) {
		body := map[string]interface{}{
			"order": "12345678903",
			"sum":   50,
		}
		jsonBody, _ := json.Marshal(body)
		req := httptest.NewRequest(http.MethodPost, "/api/user/balance/withdraw", bytes.NewReader(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		handler.Withdraw(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("невалидный JSON – 400", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/user/balance/withdraw", bytes.NewReader([]byte("{invalid")))
		req.Header.Set("Content-Type", "application/json")
		ctx := context.WithValue(req.Context(), middleware.UserIDKey, uuid.New())
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()
		handler.Withdraw(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("невалидный номер заказа – 422", func(t *testing.T) {
		userID := uuid.New()
		orderNumber := "123"
		amount := 50.0

		mockBalanceService.EXPECT().
			Withdraw(gomock.Any(), userID, orderNumber, amount).
			Return(service.ErrInvalidOrderNumber)

		body := map[string]interface{}{
			"order": orderNumber,
			"sum":   amount,
		}
		jsonBody, _ := json.Marshal(body)

		req := httptest.NewRequest(http.MethodPost, "/api/user/balance/withdraw", bytes.NewReader(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		ctx := context.WithValue(req.Context(), middleware.UserIDKey, userID)
		req = req.WithContext(ctx)

		w := httptest.NewRecorder()
		handler.Withdraw(w, req)

		assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	})

	t.Run("недостаточно средств – 402", func(t *testing.T) {
		userID := uuid.New()
		orderNumber := "12345678903"
		amount := 150.0

		mockBalanceService.EXPECT().
			Withdraw(gomock.Any(), userID, orderNumber, amount).
			Return(service.ErrInsufficientFunds)

		body := map[string]interface{}{
			"order": orderNumber,
			"sum":   amount,
		}
		jsonBody, _ := json.Marshal(body)

		req := httptest.NewRequest(http.MethodPost, "/api/user/balance/withdraw", bytes.NewReader(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		ctx := context.WithValue(req.Context(), middleware.UserIDKey, userID)
		req = req.WithContext(ctx)

		w := httptest.NewRecorder()
		handler.Withdraw(w, req)

		assert.Equal(t, http.StatusPaymentRequired, w.Code)
	})

	t.Run("ошибка сервиса – 500", func(t *testing.T) {
		userID := uuid.New()
		orderNumber := "12345678903"
		amount := 50.0

		mockBalanceService.EXPECT().
			Withdraw(gomock.Any(), userID, orderNumber, amount).
			Return(errors.New("db error"))

		body := map[string]interface{}{
			"order": orderNumber,
			"sum":   amount,
		}
		jsonBody, _ := json.Marshal(body)

		req := httptest.NewRequest(http.MethodPost, "/api/user/balance/withdraw", bytes.NewReader(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		ctx := context.WithValue(req.Context(), middleware.UserIDKey, userID)
		req = req.WithContext(ctx)

		w := httptest.NewRecorder()
		handler.Withdraw(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestBalanceHandler_GetWithdrawals(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockBalanceService := mock_service.NewMockBalanceService(ctrl)
	handler := NewBalanceHandler(mockBalanceService)

	t.Run("успешное получение списка списаний", func(t *testing.T) {
		userID := uuid.New()
		now := time.Now()
		withdrawals := []*models.Withdrawal{
			{
				ID:          uuid.New(),
				UserID:      userID,
				OrderNumber: "12345678903",
				Amount:      50.0,
				ProcessedAt: now,
			},
			{
				ID:          uuid.New(),
				UserID:      userID,
				OrderNumber: "98765432105",
				Amount:      30.0,
				ProcessedAt: now.Add(-time.Hour),
			},
		}

		mockBalanceService.EXPECT().
			GetWithdrawals(gomock.Any(), userID).
			Return(withdrawals, nil)

		req := httptest.NewRequest(http.MethodGet, "/api/user/withdrawals", nil)
		ctx := context.WithValue(req.Context(), middleware.UserIDKey, userID)
		req = req.WithContext(ctx)

		w := httptest.NewRecorder()
		handler.GetWithdrawals(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

		var resp []map[string]interface{}
		err := json.NewDecoder(w.Body).Decode(&resp)
		require.NoError(t, err)
		assert.Len(t, resp, 2)
		// Проверяем структуру первого элемента
		first := resp[0]
		assert.Equal(t, withdrawals[0].OrderNumber, first["order"])
		assert.Equal(t, withdrawals[0].Amount, first["sum"])
		assert.Equal(t, withdrawals[0].ProcessedAt.Format(time.RFC3339), first["processed_at"])
	})

	t.Run("пустой список – 204", func(t *testing.T) {
		userID := uuid.New()
		mockBalanceService.EXPECT().
			GetWithdrawals(gomock.Any(), userID).
			Return([]*models.Withdrawal{}, nil)

		req := httptest.NewRequest(http.MethodGet, "/api/user/withdrawals", nil)
		ctx := context.WithValue(req.Context(), middleware.UserIDKey, userID)
		req = req.WithContext(ctx)

		w := httptest.NewRecorder()
		handler.GetWithdrawals(w, req)

		assert.Equal(t, http.StatusNoContent, w.Code)
	})

	t.Run("отсутствие userID – 401", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/user/withdrawals", nil)
		w := httptest.NewRecorder()
		handler.GetWithdrawals(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("ошибка сервиса – 500", func(t *testing.T) {
		userID := uuid.New()
		mockBalanceService.EXPECT().
			GetWithdrawals(gomock.Any(), userID).
			Return(nil, errors.New("db error"))

		req := httptest.NewRequest(http.MethodGet, "/api/user/withdrawals", nil)
		ctx := context.WithValue(req.Context(), middleware.UserIDKey, userID)
		req = req.WithContext(ctx)

		w := httptest.NewRecorder()
		handler.GetWithdrawals(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}
