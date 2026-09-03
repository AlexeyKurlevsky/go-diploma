package storage

import (
	"context"
	"errors"

	"github.com/AlexeyKurlevsky/go-diploma/internal/models"

	"github.com/google/uuid"
)

var (
	ErrUserExists         = errors.New("user already exists")
	ErrUserNotFound       = errors.New("user not found")
	ErrOrderNotFound      = errors.New("order not found")
	ErrOrderAlreadyExists = errors.New("order already exists")
)

type UserRepository interface {
	Create(ctx context.Context, user *models.User) error
	FindByLogin(ctx context.Context, login string) (*models.User, error)
}

type OrderRepository interface {
	Create(ctx context.Context, order *models.Order) error
	FindByNumber(ctx context.Context, number string) (*models.Order, error)
	FindByUserID(ctx context.Context, userID uuid.UUID) ([]*models.Order, error)
	UpdateStatusAndAccrual(ctx context.Context, orderID uuid.UUID, status models.OrderStatus, accrual *float64) error
	FindPendingOrders(ctx context.Context, limit int) ([]*models.Order, error)
}

type WithdrawalRepository interface {
	Create(ctx context.Context, withdrawal *models.Withdrawal) error
	FindByUserID(ctx context.Context, userID uuid.UUID) ([]*models.Withdrawal, error)
	SumByUser(ctx context.Context, userID uuid.UUID) (float64, error)
}

type BalanceRepository interface {
	GetByUserID(ctx context.Context, userID uuid.UUID) (*models.UserBalance, error)
}
