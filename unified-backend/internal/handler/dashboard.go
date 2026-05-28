package handler

import (
	"log/slog"
	"net/http"

	"unified-backend/internal/middleware"
	"unified-backend/internal/service"
)

type DashboardHandler struct {
	shlinkSvc *service.ShlinkService
}

func NewDashboardHandler(svc *service.ShlinkService) *DashboardHandler {
	return &DashboardHandler{shlinkSvc: svc}
}

type DashboardResponse struct {
	TotalClicks    int            `json:"totalClicks"`
	ActiveLinks    int            `json:"activeLinks"`
	TopTags        []TagCount     `json:"topTags"`
	ClicksOverTime []ClickPoint   `json:"clicksOverTime"`
}

type TagCount struct {
	Tag   string `json:"tag"`
	Count int    `json:"count"`
}

type ClickPoint struct {
	Date   string `json:"date"`
	Clicks int    `json:"clicks"`
}

// GET /api/dashboard
func (h *DashboardHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromCtx(r.Context())
	if user == nil {
		writeJSON(w, map[string]string{"error": "forbidden"}, http.StatusForbidden)
		return
	}

	// Получаем short URLs для подсчёта активных ссылок и кликов
	urlsResp, err := h.shlinkSvc.Client().GetShortURLs(r.Context(), user.ShlinkAPIKey, "itemsPerPage=100")
	if err != nil {
		slog.Error("dashboard: get short-urls failed", "sub", user.Sub, "err", err)
		writeJSON(w, map[string]string{"error": "shlink unavailable"}, http.StatusBadGateway)
		return
	}

	urls := urlsResp.ShortURLs.Data

	var totalClicks int
	tagCountMap := map[string]int{}
	for _, u := range urls {
		totalClicks += u.VisitsSummary.Total
		for _, t := range u.Tags {
			tagCountMap[t]++
		}
	}

	// Топ-5 тегов
	topTags := topNTags(tagCountMap, 5)

	// Заглушка для clicksOverTime — в production подключить visits API
	clicksOverTime := []ClickPoint{
		{Date: "2026-05-22", Clicks: totalClicks / 7},
		{Date: "2026-05-23", Clicks: totalClicks / 6},
		{Date: "2026-05-24", Clicks: totalClicks / 5},
		{Date: "2026-05-25", Clicks: totalClicks / 4},
		{Date: "2026-05-26", Clicks: totalClicks / 3},
		{Date: "2026-05-27", Clicks: totalClicks / 2},
		{Date: "2026-05-28", Clicks: totalClicks},
	}

	resp := DashboardResponse{
		TotalClicks:    totalClicks,
		ActiveLinks:    len(urls),
		TopTags:        topTags,
		ClicksOverTime: clicksOverTime,
	}

	writeJSON(w, resp, http.StatusOK)
}

func topNTags(m map[string]int, n int) []TagCount {
	type kv struct {
		Key string
		Val int
	}
	var sorted []kv
	for k, v := range m {
		sorted = append(sorted, kv{k, v})
	}
	// Простая сортировка пузырьком для малых наборов
	for i := 0; i < len(sorted); i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[j].Val > sorted[i].Val {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}
	if len(sorted) > n {
		sorted = sorted[:n]
	}
	result := make([]TagCount, len(sorted))
	for i, kv := range sorted {
		result[i] = TagCount{Tag: kv.Key, Count: kv.Val}
	}
	return result
}
