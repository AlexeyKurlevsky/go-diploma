package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/google/uuid"

	"github.com/AlexeyKurlevsky/go-diploma/internal/client"
	"github.com/AlexeyKurlevsky/go-diploma/internal/models"
	"github.com/AlexeyKurlevsky/go-diploma/internal/storage"
	"github.com/theplant/luhn"
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
	FindPendingOrders(ctx context.Context, limit int) ([]*models.Order, error)
	ProcessOrder(ctx context.Context, orderID uuid.UUID, number string) (time.Duration, bool)
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

	// Асинхронная первая попытка опроса. Если не получится —
	// воркер подхватит заказ на следующей итерации.
	go func(orderID uuid.UUID, num string) {
		ctxBg, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		_, _ = s.ProcessOrder(ctxBg, orderID, num)
	}(order.ID, order.Number)

	return order, nil
}

func (s *orderService) GetUserOrders(ctx context.Context, userID uuid.UUID) ([]*models.Order, error) {
	return s.orderRepo.FindByUserID(ctx, userID)
}

// FindPendingOrders возвращает заказы в статусах NEW и PROCESSING —
// те, которые ещё нужно опросить во внешнем сервисе.
func (s *orderService) FindPendingOrders(ctx context.Context, limit int) ([]*models.Order, error) {
	return s.orderRepo.FindPendingOrders(ctx, limit)
}

// ProcessOrder делает одну попытку опросить внешний сервис для заказа.
// Возвращает:
//   - retryAfter — сколько подождать перед следующим опросом (актуально для 429);
//   - retry      — нужно ли повторить позже.
func (s *orderService) ProcessOrder(ctx context.Context, orderID uuid.UUID, number string) (time.Duration, bool) {
	resp, err := s.accrualClient.CheckOrder(ctx, number)
	if err != nil {
		var tooMany *client.ErrTooManyRequests
		if errors.As(err, &tooMany) {
			log.Printf("accrual 429 for order %s, retry in %s", number, tooMany.RetryAfter)
			return tooMany.RetryAfter, true
		}
		log.Printf("accrual check error for order %s: %v", number, err)
		// Ошибку не считаем поводом для backoff — воркер вернётся по общему таймеру.
		return 0, false
	}

	var (
		newStatus models.OrderStatus
		accrual   *float64
	)
	switch resp.Status {
	case StatusRegistered, StatusProcessing:
		newStatus = models.StatusProcessing
	case StatusInvalid:
		newStatus = models.StatusInvalid
		accrual = nil
	case StatusProcessed:
		newStatus = models.StatusProcessed
		accrual = resp.Accrual
		log.Printf("Order %s processed, accrual: %v", number, accrual)
	default:
		return 0, false
	}

	if err := s.orderRepo.UpdateStatusAndAccrual(ctx, orderID, newStatus, accrual); err != nil {
		log.Printf("update order %s failed: %v", number, err)
	}
	return 0, false
}
