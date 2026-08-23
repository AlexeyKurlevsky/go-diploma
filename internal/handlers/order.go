package handlers

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/AlexeyKurlevsky/go-diploma/internal/middleware"
	"github.com/AlexeyKurlevsky/go-diploma/internal/service"
)

type OrderHandler struct {
	orderService service.OrderService
}

func NewOrderHandler(orderService service.OrderService) *OrderHandler {
	return &OrderHandler{orderService: orderService}
}

// UploadOrder — POST /api/user/orders
func (h *OrderHandler) UploadOrder(w http.ResponseWriter, r *http.Request) {
	// Получаем userID из контекста (установлен middleware)
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	orderNumInt, err := strconv.ParseInt(string(bodyBytes), 10, 64)
	if err != nil {
		http.Error(w, "Invalid integer format", http.StatusBadRequest)
		return
	}

	// Вызываем сервис
	_, err = h.orderService.UploadOrder(r.Context(), userID, orderNumInt)
	if err != nil {
		switch err {
		case service.ErrInvalidOrderNumber:
			http.Error(w, "invalid order number format", http.StatusUnprocessableEntity)
		case service.ErrOrderConflict:
			http.Error(w, "order already uploaded by another user", http.StatusConflict)
		case service.ErrOrderAlreadyUploadedByUser:
			// Возвращаем 200 OK, тело не требуется (по ТЗ)
			w.WriteHeader(http.StatusOK)
		default:
			http.Error(w, "internal server error", http.StatusInternalServerError)
		}
		return
	}

	// Новый заказ принят
	w.WriteHeader(http.StatusAccepted)
}

// GetOrders — GET /api/user/orders
func (h *OrderHandler) GetOrders(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	orders, err := h.orderService.GetUserOrders(r.Context(), userID)
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	if len(orders) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	// Формируем ответ, убираем поля, которые не нужны в JSON
	response := make([]map[string]interface{}, len(orders))
	for i, o := range orders {
		item := map[string]interface{}{
			"number":      o.Number,
			"status":      string(o.Status),
			"uploaded_at": o.UploadedAt.Format(time.RFC3339),
		}
		if o.Accrual != nil {
			item["accrual"] = *o.Accrual
		}
		response[i] = item
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}
