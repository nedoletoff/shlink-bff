package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

// writeJSON записывает JSON-ответ с указанным статус-кодом.
// Используется всеми хендлерами пакета.
func writeJSON(w http.ResponseWriter, v any, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		// WriteHeader уже вызван — ответ изменить нельзя, это не ошибка уровня Error,
		// а лишь диагностика (например, клиент разорвал соединение) — пишем Debug (#30).
		slog.Debug("handler: failed to encode json response", "err", err)
	}
}
