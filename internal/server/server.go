package router

import (
	"github.com/AlexeyKurlevsky/go-diploma/internal/handlers"
	mymiddleware "github.com/AlexeyKurlevsky/go-diploma/internal/middleware"
	"github.com/AlexeyKurlevsky/go-diploma/internal/service"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func NewRouter(
	authHandler *handlers.AuthHandler,
	orderHandler *handlers.OrderHandler,
	authService service.AuthService,
	// другие хендлеры (заказы, баланс) ...
) *chi.Mux {
	r := chi.NewRouter()

	// Глобальные middleware
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Post("/api/user/register", authHandler.Register)
	r.Post("/api/user/login", authHandler.Login)

	// Защищённые маршруты
	r.Group(func(r chi.Router) {
		r.Use(mymiddleware.AuthMiddleware(authService))
		r.Post("/api/user/orders", orderHandler.UploadOrder)
		r.Get("/api/user/orders", orderHandler.GetOrders)
	})

	return r
}
