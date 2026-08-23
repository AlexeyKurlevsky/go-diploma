package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/AlexeyKurlevsky/go-diploma/internal/client"
	"github.com/AlexeyKurlevsky/go-diploma/internal/config"
	"github.com/AlexeyKurlevsky/go-diploma/internal/handlers"
	"github.com/AlexeyKurlevsky/go-diploma/internal/logger"
	router "github.com/AlexeyKurlevsky/go-diploma/internal/server"
	"github.com/AlexeyKurlevsky/go-diploma/internal/service"
	storage "github.com/AlexeyKurlevsky/go-diploma/internal/storage/postgres"
	"go.uber.org/zap"
)

func main() {
	// Загрузка конфигурации из env
	cfg, err := config.NewConfig()
	if err != nil {
		log.Fatalf("Incorrect config: %v", err)
	}

	if err := logger.Initialize(cfg.LogLevel); err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}

	// Подключение к БД
	db, err := storage.NewPostgresStorage(cfg.DatabaseURI)
	if err != nil {
		log.Fatal("failed to connect to DB:", err)
	}

	// Инициализация репозиториев
	userRepo := storage.NewUserRepository(db.Pool)
	orderRepo := storage.NewOrderRepository(db.Pool)
	withdrawalRepo := storage.NewWithdrawalRepository(db.Pool)
	balanceRepo := storage.NewBalanceRepository(db.Pool)

	// Инициализация сервисов
	authService := service.NewAuthService(userRepo, cfg.JWTSecret, cfg.JWTExpireTime)
	accrualClient := client.NewAccrualClient(cfg.AccrualAddr, 10*time.Second)
	orderService := service.NewOrderService(orderRepo, accrualClient)
	balanceService := service.NewBalanceService(balanceRepo, withdrawalRepo)

	// Инициализация хендлеров
	authHandler := handlers.NewAuthHandler(authService)
	orderHandler := handlers.NewOrderHandler(orderService)
	balanceHandler := handlers.NewBalanceHandler(balanceService)

	// Роутер
	r := router.NewRouter(authHandler, orderHandler, balanceHandler, authService)

	// Фоновый воркер для обновления MV с балансом
	go func() {
		ticker := time.NewTicker(30 * time.Second) // обновляем каждые 30 секунд
		defer ticker.Stop()
		for range ticker.C {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			if err := balanceService.RefreshBalanceView(ctx); err != nil {
				log.Printf("failed to refresh balance view: %v", err)
			}
			cancel()
		}
	}()

	logger.Log.Info("Config",
		zap.String("ServerAddr", cfg.ServerAddr),
	)

	if err := http.ListenAndServe(cfg.ServerAddr, r); err != nil {
		logger.Log.Fatal("Server failed: %v", zap.Error(err))
	}
}
