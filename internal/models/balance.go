package models

import "github.com/google/uuid"

type UserBalance struct {
	UserID         uuid.UUID `json:"-"`
	Balance        float64   `json:"current"`
	TotalAccrued   float64   `json:"-"`
	TotalWithdrawn float64   `json:"withdrawn"`
}
