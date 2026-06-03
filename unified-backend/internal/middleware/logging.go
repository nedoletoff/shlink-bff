package middleware

import (
	"log/slog"
	"net/http"
	"time"
)

type responseWriter struct {
	http.ResponseWriter
	statusCode  int
	wroteHeader bool
}

func (rw *responseWriter) WriteHeader(code int) {
	if rw.wroteHeader {
		return
	}
	rw.statusCode = code
	rw.wroteHeader = true
	rw.ResponseWriter.WriteHeader(code)
}

// Write перехватывается, чтобы статус 200 фиксировался даже без явного WriteHeader (#29).
func (rw *responseWriter) Write(b []byte) (int, error) {
	if !rw.wroteHeader {
		rw.WriteHeader(http.StatusOK)
	}
	return rw.ResponseWriter.Write(b)
}

// RequestLogger — структурированное JSON-логирование входящих запросов.
// Намеренно НЕ логирует тело запроса и заголовки с API-ключами.
func RequestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
		// statusCode инициализирован 200 на случай, если handler ничего не напишет.

		next.ServeHTTP(rw, r)

		id := IdentityFromCtx(r.Context())
		slog.Info("http_request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", rw.statusCode,
			"latency", time.Since(start).String(),
			"sub", id.Sub,
			"username", id.Username,
			"role", id.Role,
			"remote", ClientIP(r),
		)
	})
}
