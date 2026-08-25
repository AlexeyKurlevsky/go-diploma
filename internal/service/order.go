package service

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/AlexeyKurlevsky/go-diploma/internal/client"
	"github.com/AlexeyKurlevsky/go-diploma/internal/models"
	"github.com/AlexeyKurlevsky/go-diploma/internal/storage"
	"github.com/theplant/luhn"

	"github.com/google/uuid"
)

const (
	StatusRegistered string = "REGISTERED"
	StatusProcessing string = "PROCESSING"
	StatusInvalid    string = "INVALID"
	StatusProcessed  string = "PROCESSED"
)

type OrderService interface {
	UploadOrder(ctx context.Context, userID uuid.UUID, number string) (*models.Order, error)
	GetUserOrders(ctx context.Context, userID uuid.UUID) ([]*models.Order, error)
	ProcessPendingOrders(ctx context.Context, limit int) error
}

type orderService struct {
	orderRepo     storage.OrderRepository
	accrualClient client.AccrualClient
}

func NewOrderService(orderRepo storage.OrderRepository, accrualClient client.AccrualClient) OrderService {
	return &orderService{
		orderRepo:     orderRepo,
		accrualClient: accrualClient,
	}
}

func (s *orderService) UploadOrder(ctx context.Context, userID uuid.UUID, orderNumber string) (*models.Order, error) {
	// 1. Валидация номера заказа (алгоритм Луна)
	num, err := strconv.Atoi(orderNumber)
	if err != nil {
		fmt.Println("Error during conversion:", err)
		return nil, ErrInvalidOrderNumber
	}
	if !luhn.Valid(num) {
		return nil, ErrInvalidOrderNumber
	}
	existing, err := s.orderRepo.FindByNumber(ctx, orderNumber)
	if err != nil && !errors.Is(err, storage.ErrOrderNotFound) {
		return nil, fmt.Errorf("check existing: %w", err)
	}
	if existing != nil {
		if existing.UserID == userID {
			return existing, ErrOrderAlreadyUploadedByUser
		}
		return nil, ErrOrderConflict
	}
	order := &models.Order{
		UserID:     userID,
		Number:     orderNumber,
		Status:     models.StatusNew,
		UploadedAt: time.Now(),
		UpdatedAt:  time.Now(),
	}
	if err := s.orderRepo.Create(ctx, order); err != nil {
		return nil, fmt.Errorf("create order: %w", err)
	}
	go func() {
		ctxBg := context.Background()
		s.processOrder(ctxBg, order.ID, orderNumber)
	}()
	return order, nil
}

func (s *orderService) GetUserOrders(ctx context.Context, userID uuid.UUID) ([]*models.Order, error) {
	return s.orderRepo.FindByUserID(ctx, userID)
}

func (s *orderService) processOrder(ctx context.Context, orderID uuid.UUID, number string) {
	resp, err := s.accrualClient.CheckOrder(ctx, number)
	if err != nil {
		return
	}
	var newStatus models.OrderStatus
	var accrual *float64
	switch resp.Status {
	case StatusRegistered, StatusProcessing:
		newStatus = models.StatusProcessing
	case StatusInvalid:
		newStatus = models.StatusInvalid
		accrual = nil
	case StatusProcessed:
		newStatus = models.StatusProcessed
		accrual = resp.Accrual
	default:
		newStatus = models.StatusNew
	}
	_ = s.orderRepo.UpdateStatusAndAccrual(ctx, orderID, newStatus, accrual)
}

func (s *orderService) ProcessPendingOrders(ctx context.Context, limit int) error {
	orders, err := s.orderRepo.FindPendingOrders(ctx, limit)
	if err != nil {
		return fmt.Errorf("find pending: %w", err)
	}
	for _, order := range orders {
		s.processOrder(ctx, order.ID, order.Number)
	}
	return nil
}
