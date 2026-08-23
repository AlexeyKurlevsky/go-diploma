package service

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/AlexeyKurlevsky/go-diploma/internal/models"
	"github.com/AlexeyKurlevsky/go-diploma/internal/storage"

	"github.com/AlexeyKurlevsky/go-diploma/internal/client"
	"github.com/theplant/luhn"
)

const (
	StatusRegistered string = "REGISTERED"
	StatusProcessing string = "PROCESSING"
	StatusInvalid    string = "INVALID"
	StatusProcessed  string = "PROCESSED"
)

type OrderService interface {
	UploadOrder(ctx context.Context, userID int64, number int64) (*models.Order, error)
	GetUserOrders(ctx context.Context, userID int64) ([]*models.Order, error)
	ProcessPendingOrders(ctx context.Context, limit int) error // для воркера
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

func (s *orderService) UploadOrder(ctx context.Context, userID int64, number int64) (*models.Order, error) {
	// 1. Валидация номера (алгоритм Луна)
	if !luhn.Valid(int(number)) {
		return nil, ErrInvalidOrderNumber
	}
	numberStr := strconv.FormatInt(number, 10)

	// 2. Проверяем, существует ли уже такой номер
	existing, err := s.orderRepo.FindByNumber(ctx, numberStr)
	if err != nil && !errors.Is(err, storage.ErrOrderNotFound) {
		return nil, fmt.Errorf("check existing: %w", err)
	}
	if existing != nil {
		if existing.UserID == userID {
			// Уже загружен этим пользователем – возвращаем его (код 200)
			return existing, ErrOrderAlreadyUploadedByUser
		}
		// Загружен другим пользователем – конфликт
		return nil, ErrOrderConflict
	}

	// 3. Создаём запись заказа со статусом NEW
	order := &models.Order{
		UserID:     userID,
		Number:     numberStr,
		Status:     models.StatusNew,
		UploadedAt: time.Now(),
		UpdatedAt:  time.Now(),
	}
	if err := s.orderRepo.Create(ctx, order); err != nil {
		return nil, fmt.Errorf("create order: %w", err)
	}

	// 4. Синхронно пробуем получить информацию из внешнего сервиса
	// (если сервис недоступен, заказ останется в NEW и будет обработан воркером)
	go func() {
		// Используем новый контекст, чтобы не зависеть от контекста запроса
		ctxBg := context.Background()
		s.processOrder(ctxBg, order.ID, numberStr)
	}()

	return order, nil
}

func (s *orderService) GetUserOrders(ctx context.Context, userID int64) ([]*models.Order, error) {
	orders, err := s.orderRepo.FindByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get user orders: %w", err)
	}
	return orders, nil
}

// processOrder — вызывает внешний сервис и обновляет статус
func (s *orderService) processOrder(ctx context.Context, orderID int64, number string) {
	resp, err := s.accrualClient.CheckOrder(ctx, number)
	if err != nil {
		// Если ошибка, заказ остаётся в текущем статусе (NEW или PROCESSING)
		// Можно залогировать
		return
	}

	// Маппинг статусов внешнего сервиса на наши
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

	if err := s.orderRepo.UpdateStatusAndAccrual(ctx, orderID, newStatus, accrual); err != nil {
		// Логируем ошибку
	}
}

// ProcessPendingOrders — для фонового воркера, обрабатывает заказы со статусами NEW и PROCESSING
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
