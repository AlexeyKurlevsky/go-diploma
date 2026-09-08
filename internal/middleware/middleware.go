package middleware

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/AlexeyKurlevsky/go-diploma/internal/logger"
)

func RequestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rw := &logger.MyResponseWriter{
			ResponseWriter: w,
			Status:         http.StatusOK,
		}

		// Обрабатываем запрос
		next.ServeHTTP(rw, r)

		// Логируем все необходимые поля
		slog.Info("HTTP request",
			"method", r.Method,
			"uri", r.URL.RequestURI(),
			"duration", time.Since(start),
			"status", rw.Status,
			"response_size", rw.Size,
		)
	})
}
