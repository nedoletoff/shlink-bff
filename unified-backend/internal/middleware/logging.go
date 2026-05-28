package middleware

import (
	"log/slog"
	"net/http"
	"time"
)

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// RequestLogger — структурированное JSON-логирование входящих запросов.
// Намеренно НЕ логирует тело запроса и заголовки с API-ключами.
func RequestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(rw, r)

		id := IdentityFromCtx(r.Context())
		slog.Info("http_request",
			"method",   r.Method,
			"path",     r.URL.Path,
			"status",   rw.statusCode,
			"latency",  time.Since(start).String(),
			"sub",      id.Sub,
			"username", id.Username,
			"role",     id.Role,
			"remote",   r.RemoteAddr,
		)
	})
}
