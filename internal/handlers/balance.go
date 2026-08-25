package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/AlexeyKurlevsky/go-diploma/internal/middleware"
	"github.com/AlexeyKurlevsky/go-diploma/internal/service"
)

type BalanceHandler struct {
	balanceService service.BalanceService
}

func NewBalanceHandler(balanceService service.BalanceService) *BalanceHandler {
	return &BalanceHandler{balanceService: balanceService}
}

func (h *BalanceHandler) GetBalance(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	current, withdrawn, err := h.balanceService.GetBalance(r.Context(), userID)
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"current":   current,
		"withdrawn": withdrawn,
	})
}

func (h *BalanceHandler) Withdraw(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	var req struct {
		Order string  `json:"order"`
		Sum   float64 `json:"sum"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.Order == "" || req.Sum <= 0 {
		http.Error(w, "invalid order or sum", http.StatusBadRequest)
		return
	}
	err := h.balanceService.Withdraw(r.Context(), userID, req.Order, req.Sum)
	if err != nil {
		switch err {
		case service.ErrInvalidOrderNumber:
			http.Error(w, "invalid order number", http.StatusUnprocessableEntity)
		case service.ErrInsufficientFunds:
			http.Error(w, "insufficient funds", http.StatusPaymentRequired)
		default:
			http.Error(w, "internal server error", http.StatusInternalServerError)
		}
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *BalanceHandler) GetWithdrawals(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	withdrawals, err := h.balanceService.GetWithdrawals(r.Context(), userID)
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	if len(withdrawals) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	response := make([]map[string]interface{}, len(withdrawals))
	for i, w := range withdrawals {
		response[i] = map[string]interface{}{
			"order":        w.OrderNumber,
			"sum":          w.Amount,
			"processed_at": w.ProcessedAt.Format(time.RFC3339),
		}
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}
