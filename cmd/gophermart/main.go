package main

import (
	"log"
	"net/http"

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
	defer db.Close()

	// Инициализация репозиториев
	userRepo := storage.NewUserRepository(db.DbConn)

	// Инициализация сервисов
	authService := service.NewAuthService(userRepo, cfg.JWTSecret, cfg.JWTExpireTime)

	// Инициализация хендлеров
	authHandler := handlers.NewAuthHandler(authService)
	// другие хендлеры...

	// Роутер
	r := router.NewRouter(authHandler, authService /*, ... */)

	logger.Log.Info("Config",
		zap.String("ServerAddr", cfg.ServerAddr),
	)

	if err := http.ListenAndServe(cfg.ServerAddr, r); err != nil {
		logger.Log.Fatal("Server failed: %v", zap.Error(err))
	}
}
