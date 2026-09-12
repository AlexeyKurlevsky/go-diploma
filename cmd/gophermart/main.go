package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/AlexeyKurlevsky/go-diploma/internal/client"
	"github.com/AlexeyKurlevsky/go-diploma/internal/config"
	"github.com/AlexeyKurlevsky/go-diploma/internal/handlers"
	"github.com/AlexeyKurlevsky/go-diploma/internal/logger"
	router "github.com/AlexeyKurlevsky/go-diploma/internal/server"
	"github.com/AlexeyKurlevsky/go-diploma/internal/service"
	storage "github.com/AlexeyKurlevsky/go-diploma/internal/storage/postgres"
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
	defer db.Pool.Close()

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

	logger.Log.Info("Config",
		"ServerAddr", cfg.ServerAddr,
	)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Воркер опроса заказов
	go runOrderWorker(ctx, orderService)

	// HTTP-сервер запускается в горутине
	srv := &http.Server{Addr: cfg.ServerAddr, Handler: r}
	go func() {
		log.Printf("Starting server on %s", cfg.ServerAddr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("server error: %v", err)
		}
	}()

	// Ждём отмены контекста (сигнал)
	<-ctx.Done()
	log.Println("shutdown signal received")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("server shutdown error: %v", err)
	}
}

func runOrderWorker(ctx context.Context, svc service.OrderService) {
	const (
		baseInterval = 10 * time.Second
		maxInterval  = 5 * time.Minute
		batchSize    = 100
	)

	ticker := time.NewTicker(baseInterval)
	defer ticker.Stop()

	var skipUntil time.Time

	for {
		select {
		case <-ctx.Done():
			log.Println("order worker stopped")
			return
		case <-ticker.C:
			if time.Now().Before(skipUntil) {
				continue
			}

			opCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
			orders, err := svc.FindPendingOrders(opCtx, batchSize)
			cancel()
			if err != nil {
				log.Printf("worker: find pending: %v", err)
				continue
			}

			var maxRetry time.Duration
			for _, o := range orders {
				opCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
				retryAfter, _ := svc.ProcessOrder(opCtx, o.ID, o.Number)
				cancel()
				if retryAfter > maxRetry {
					maxRetry = retryAfter
				}
			}

			if maxRetry > 0 {
				if maxRetry > maxInterval {
					maxRetry = maxInterval
				}
				skipUntil = time.Now().Add(maxRetry)
				log.Printf("worker: backoff %s after 429", maxRetry)
			}
		}
	}
}
