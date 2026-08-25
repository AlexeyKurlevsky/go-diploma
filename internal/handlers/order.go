package handlers

import (
	"encoding/json"
	"io"
	"net/http"
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

func (h *OrderHandler) UploadOrder(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	orderNumber := string(body)
	if orderNumber == "" {
		http.Error(w, "empty order number", http.StatusBadRequest)
		return
	}
	_, err = h.orderService.UploadOrder(r.Context(), userID, orderNumber)
	if err != nil {
		switch err {
		case service.ErrInvalidOrderNumber:
			http.Error(w, "invalid order number format", http.StatusUnprocessableEntity)
		case service.ErrOrderConflict:
			http.Error(w, "order already uploaded by another user", http.StatusConflict)
		case service.ErrOrderAlreadyUploadedByUser:
			w.WriteHeader(http.StatusOK)
		default:
			http.Error(w, "internal server error", http.StatusInternalServerError)
		}
		return
	}
	w.WriteHeader(http.StatusAccepted)
}

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
