package router

import (
	"log/slog"
	"net/http"

	"github.com/AlexeyKurlevsky/go-diploma/internal/handlers"
	mymiddleware "github.com/AlexeyKurlevsky/go-diploma/internal/middleware"
	"github.com/AlexeyKurlevsky/go-diploma/internal/service"
)

// recoverer — middleware для восстановления после паники.
func recoverer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				slog.Error("panic recovered", "error", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// NewRouter создаёт маршрутизатор с использованием стандартного http.ServeMux.
// Все маршруты защищены глобальными middleware (логгер и восстановление).
func NewRouter(
	authHandler *handlers.AuthHandler,
	orderHandler *handlers.OrderHandler,
	balanceHandler *handlers.BalanceHandler,
	authService service.AuthService,
) http.Handler {
	mainMux := http.NewServeMux()

	// Открытые маршруты (без аутентификации)
	mainMux.HandleFunc("POST /api/user/register", authHandler.Register)
	mainMux.HandleFunc("POST /api/user/login", authHandler.Login)

	// Защищённые маршруты – выделяем в отдельный мультиплексор
	protectedMux := http.NewServeMux()
	protectedMux.HandleFunc("POST /orders", orderHandler.UploadOrder)
	protectedMux.HandleFunc("GET /orders", orderHandler.GetOrders)
	protectedMux.HandleFunc("GET /balance", balanceHandler.GetBalance)
	protectedMux.HandleFunc("POST /balance/withdraw", balanceHandler.Withdraw)
	protectedMux.HandleFunc("GET /withdrawals", balanceHandler.GetWithdrawals)

	// Применяем middleware аутентификации к защищённому роутеру
	authProtected := mymiddleware.AuthMiddleware(authService)(protectedMux)

	// Монтируем защищённые маршруты по префиксу /api/user/
	mainMux.Handle("/api/user/", http.StripPrefix("/api/user", authProtected))

	// Навешиваем глобальные middleware (логгер и восстановление)
	handler := mymiddleware.RequestLogger(recoverer(mainMux))
	return handler
}
