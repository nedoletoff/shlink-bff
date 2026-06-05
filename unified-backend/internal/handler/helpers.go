package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"sort"
	"strings"
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

type ClickPoint struct {
	Date   string `json:"date"`
	Clicks int    `json:"clicks"`
}

type namedCount struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

func urlDetailParseBrowser(ua string) string {
	ua = strings.ToLower(ua)
	switch {
	case strings.Contains(ua, "edg"):
		return "Edge"
	case strings.Contains(ua, "opr") || strings.Contains(ua, "opera"):
		return "Opera"
	case strings.Contains(ua, "chrome") || strings.Contains(ua, "chromium"):
		return "Chrome"
	case strings.Contains(ua, "firefox"):
		return "Firefox"
	case strings.Contains(ua, "safari"):
		return "Safari"
	default:
		return "Other"
	}
}

func urlDetailParseOS(ua string) string {
	ua = strings.ToLower(ua)
	switch {
	case strings.Contains(ua, "android"):
		return "Android"
	case strings.Contains(ua, "iphone") || strings.Contains(ua, "ipad") || strings.Contains(ua, "ipod"):
		return "iOS"
	case strings.Contains(ua, "windows"):
		return "Windows"
	case strings.Contains(ua, "mac os") || strings.Contains(ua, "macos") || strings.Contains(ua, "macintosh"):
		return "macOS"
	case strings.Contains(ua, "linux"):
		return "Linux"
	default:
		return "Other"
	}
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
