package models

type UserBalance struct {
	UserID         int64   `json:"-"`
	Balance        float64 `json:"current"`
	TotalAccrued   float64 `json:"-"`
	TotalWithdrawn float64 `json:"withdrawn"`
}
