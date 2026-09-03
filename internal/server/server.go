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
	balanceHandler *handlers.BalanceHandler,
	authService service.AuthService,
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

		r.Get("/api/user/balance", balanceHandler.GetBalance)
		r.Post("/api/user/balance/withdraw", balanceHandler.Withdraw)
		r.Get("/api/user/withdrawals", balanceHandler.GetWithdrawals)
	})

	return r
}
