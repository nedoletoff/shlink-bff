package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"sort"
)

// writeJSON записывает JSON-ответ с указанным статус-кодом.
// Используется всеми хендлерами пакета.
func writeJSON(w http.ResponseWriter, v any, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.Debug("handler: failed to encode json response", "err", err)
	}
}

type ClickPoint struct {
	Date   string `json:"date"`
	Clicks int    `json:"clicks"`
}

type namedCount struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

func topCountSlice(m map[string]int, n int) []namedCount {
	out := make([]namedCount, 0, len(m))
	for k, v := range m {
		out = append(out, namedCount{Name: k, Count: v})
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Count > out[j].Count
	})
	if len(out) > n {
		return out[:n]
	}
	return out
}
