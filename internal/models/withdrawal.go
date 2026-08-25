package models

import (
	"time"

	"github.com/google/uuid"
)

type Withdrawal struct {
	ID          uuid.UUID `json:"-"`
	UserID      uuid.UUID `json:"-"`
	OrderNumber string    `json:"order"`
	Amount      float64   `json:"sum"`
	ProcessedAt time.Time `json:"processed_at"`
}
