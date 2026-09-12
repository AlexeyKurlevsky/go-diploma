package models

import "time"

type BalanceResponse struct {
	Current   float64 `json:"current"`
	Withdrawn float64 `json:"withdrawn"`
}

type OrderResponse struct {
	Number     string      `json:"number"`
	Status     OrderStatus `json:"status"`
	Accrual    *float64    `json:"accrual,omitempty"`
	UploadedAt string      `json:"uploaded_at"`
}

type WithdrawalResponse struct {
	Order       string  `json:"order"`
	Sum         float64 `json:"sum"`
	ProcessedAt string  `json:"processed_at"`
}

type WithdrawRequest struct {
	Order string  `json:"order"`
	Sum   float64 `json:"sum"`
}

func NewOrderResponse(o *Order) OrderResponse {
	return OrderResponse{
		Number:     o.Number,
		Status:     o.Status,
		Accrual:    o.Accrual,
		UploadedAt: o.UploadedAt.Format(time.RFC3339),
	}
}

func NewOrderResponses(orders []*Order) []OrderResponse {
	res := make([]OrderResponse, 0, len(orders))
	for _, o := range orders {
		res = append(res, NewOrderResponse(o))
	}
	return res
}

func NewWithdrawalResponse(w *Withdrawal) WithdrawalResponse {
	return WithdrawalResponse{
		Order:       w.OrderNumber,
		Sum:         w.Amount,
		ProcessedAt: w.ProcessedAt.Format(time.RFC3339),
	}
}

func NewWithdrawalResponses(ws []*Withdrawal) []WithdrawalResponse {
	res := make([]WithdrawalResponse, 0, len(ws))
	for _, w := range ws {
		res = append(res, NewWithdrawalResponse(w))
	}
	return res
}
