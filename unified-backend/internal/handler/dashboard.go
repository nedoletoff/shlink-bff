package handler

import (
	"log/slog"
	"net/http"
	"sort"
	"time"

	"unified-backend/internal/middleware"
	"unified-backend/internal/service"
)

// daysWindow — сколько дней назад показывать в графике clicksOverTime.
const daysWindow = 7

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

	// Реальный график кликов за последние daysWindow дней (#2, #3):
	// динамические даты + бакетирование реальных визитов из Shlink visits API.
	clicksOverTime := h.buildClicksOverTime(r, user.ShlinkAPIKey)

	resp := DashboardResponse{
		TotalClicks:    totalClicks,
		ActiveLinks:    len(urls),
		TopTags:        topTags,
		ClicksOverTime: clicksOverTime,
	}

	writeJSON(w, resp, http.StatusOK)
}

// buildClicksOverTime строит реальный временной ряд кликов за последние daysWindow дней.
//
// Даты вычисляются динамически от текущей даты, а клики — это реальные визиты
// из Shlink visits API, сгруппированные по дням. Если visits API недоступен,
// возвращаем ряд с нулями (честное "нет данных"), а не выдуманные числа.
func (h *DashboardHandler) buildClicksOverTime(r *http.Request, apiKey string) []ClickPoint {
	const dayFmt = "2006-01-02"
	now := time.Now()

	// Инициализируем бакеты последних daysWindow дней нулями.
	buckets := make(map[string]int, daysWindow)
	ordered := make([]string, 0, daysWindow)
	for i := daysWindow - 1; i >= 0; i-- {
		d := now.AddDate(0, 0, -i).Format(dayFmt)
		buckets[d] = 0
		ordered = append(ordered, d)
	}

	startDate := now.AddDate(0, 0, -(daysWindow - 1)).Format(dayFmt)
	endDate := now.Format(dayFmt)

	visitsResp, err := h.shlinkSvc.Client().GetNonOrphanVisits(r.Context(), apiKey, startDate, endDate, 0)
	if err != nil {
		slog.Warn("dashboard: visits API unavailable, returning empty series", "err", err)
	} else {
		for _, v := range visitsResp.Visits.Data {
			// v.Date — ISO-8601 с временем; берём только день.
			t, perr := time.Parse(time.RFC3339, v.Date)
			if perr != nil {
				if len(v.Date) >= len(dayFmt) {
					if _, ok := buckets[v.Date[:len(dayFmt)]]; ok {
						buckets[v.Date[:len(dayFmt)]]++
					}
				}
				continue
			}
			day := t.Format(dayFmt)
			if _, ok := buckets[day]; ok {
				buckets[day]++
			}
		}
	}

	points := make([]ClickPoint, 0, daysWindow)
	for _, d := range ordered {
		points = append(points, ClickPoint{Date: d, Clicks: buckets[d]})
	}
	return points
}

func topNTags(m map[string]int, n int) []TagCount {
	type kv struct {
		Key string
		Val int
	}
	sorted := make([]kv, 0, len(m))
	for k, v := range m {
		sorted = append(sorted, kv{k, v})
	}
	// Сортировка по убыванию частоты (#31: замена O(n²) пузырька).
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Val > sorted[j].Val
	})
	if len(sorted) > n {
		sorted = sorted[:n]
	}
	result := make([]TagCount, len(sorted))
	for i, kv := range sorted {
		result[i] = TagCount{Tag: kv.Key, Count: kv.Val}
	}
	return result
}
