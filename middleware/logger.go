package middleware

import (
	"log/slog"
	"net/http"
	"time"
)

// Logger is a middleware that logs HTTP requests using the default slog logger.
func Logger(next http.HandlerFunc) http.HandlerFunc {
	logger := slog.Default()

	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		next(w, r)

		duration := time.Since(start)

		logger.Info("HTTP Request",
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.String("query", r.URL.RawQuery),
			slog.String("remote_addr", r.RemoteAddr),
			slog.Duration("duration", duration),
			slog.String("protocol", r.Proto),
		)
	}
}

// CustomLogger is a middleware that logs HTTP requests using a custom slog logger.
func CustomLogger(logger *slog.Logger, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		next(w, r)

		duration := time.Since(start)

		logger.Info("HTTP Request",
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.String("query", r.URL.RawQuery),
			slog.String("remote_addr", r.RemoteAddr),
			slog.Duration("duration", duration),
			slog.String("protocol", r.Proto),
		)
	}
}
