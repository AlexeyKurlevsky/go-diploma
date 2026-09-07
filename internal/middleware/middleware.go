package middleware

import (
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/AlexeyKurlevsky/go-diploma/internal/logger"
)

func GzipMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		contentType := r.Header.Get("Content-Type")
		ow := w

		if contentType == "application/json" || contentType == "text/html" || contentType == "text/plain" {

			acceptEncoding := r.Header.Get("Accept-Encoding")
			supportsGzip := strings.Contains(acceptEncoding, "gzip")
			if supportsGzip {
				cw := newCompressWriter(w)
				ow = cw
				defer cw.Close()
			}

			contentEncoding := r.Header.Get("Content-Encoding")
			sendsGzip := strings.Contains(contentEncoding, "gzip")
			if sendsGzip {
				cr, err := newCompressReader(r.Body)
				if err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				r.Body = cr
				defer cr.Close()
			}

		}

		next.ServeHTTP(ow, r)
	})
}

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
