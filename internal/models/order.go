package models

import (
	"time"

	"github.com/google/uuid"
)

// OrderStatus — возможные статусы заказа
type OrderStatus string

const (
	StatusNew        OrderStatus = "NEW"
	StatusProcessing OrderStatus = "PROCESSING"
	StatusInvalid    OrderStatus = "INVALID"
	StatusProcessed  OrderStatus = "PROCESSED"
)

// Order — модель заказа
type Order struct {
	ID         uuid.UUID   `json:"-"`
	UserID     uuid.UUID   `json:"-"`
	Number     string      `json:"number"`
	Status     OrderStatus `json:"status"`
	Accrual    *float64    `json:"accrual,omitempty"` // nil, если не начислено
	UploadedAt time.Time   `json:"uploaded_at"`
	UpdatedAt  time.Time   `json:"-"`
}
