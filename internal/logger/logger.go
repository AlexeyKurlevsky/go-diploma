package logger

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"
)

// Logger — обёртка над slog.Logger с методом Fatal.
type Logger struct {
	*slog.Logger
}

// Fatal логирует сообщение на уровне Error и завершает программу с кодом 1.
func (l *Logger) Fatal(msg string, args ...any) {
	l.Logger.Error(msg, args...)
	os.Exit(1)
}

// Log — глобальный экземпляр логгера.
var Log *Logger

// Initialize настраивает глобальный логгер.
// level: "debug", "info", "warn", "error".
func Initialize(level string) error {
	l := strings.ToLower(level)
	var lvl slog.Level
	switch l {
	case "debug":
		lvl = slog.LevelDebug
	case "info":
		lvl = slog.LevelInfo
	case "warn":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	default:
		return fmt.Errorf("unknown log level: %s", level)
	}

	opts := &slog.HandlerOptions{
		Level: lvl,
	}
	handler := slog.NewJSONHandler(os.Stderr, opts)
	base := slog.New(handler)
	slog.SetDefault(base)
	Log = &Logger{Logger: base}
	return nil
}

// Fatal — вспомогательная функция для логирования фатальной ошибки без использования глобальной переменной.
func Fatal(msg string, args ...any) {
	slog.Error(msg, args...)
	os.Exit(1)
}

// MyResponseWriter — обёртка для захвата кода статуса и размера ответа.
type MyResponseWriter struct {
	http.ResponseWriter
	Status int
	Size   int
}

func (rw *MyResponseWriter) WriteHeader(status int) {
	rw.Status = status
	rw.ResponseWriter.WriteHeader(status)
}

func (rw *MyResponseWriter) Write(b []byte) (int, error) {
	n, err := rw.ResponseWriter.Write(b)
	rw.Size += n
	return n, err
}
