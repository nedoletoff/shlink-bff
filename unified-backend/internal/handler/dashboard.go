package handler

import (
	"fmt"
	"log/slog"
	"net/http"
	"sort"
	"strings"
	"time"

	"unified-backend/internal/middleware"
	"unified-backend/internal/repository/postgres"
	"unified-backend/internal/service"
	"unified-backend/internal/shlink"
)

type DashboardHandler struct {
	shlinkSvc *service.ShlinkService
	userRepo  *postgres.UserRepository
}

func NewDashboardHandler(shlinkSvc *service.ShlinkService, userRepo *postgres.UserRepository) *DashboardHandler {
	return &DashboardHandler{shlinkSvc: shlinkSvc, userRepo: userRepo}
}

// GET /api/dashboard
func (h *DashboardHandler) GetDashboard(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromCtx(r.Context())
	if user == nil {
		writeJSON(w, map[string]string{"error": "user not found"}, http.StatusUnauthorized)
		return
	}

	query := r.URL.Query()
	days := 30
	if p := query.Get("period"); p != "" {
		var n int
		if _, err := fmt.Sscanf(p, "%d", &n); err == nil && n > 0 && n <= 365 {
			days = n
		}
	}

	p := h.shlinkSvc.Perms(user)

	// --- short URLs ---
	urls, err := h.shlinkSvc.Client().GetAllShortURLs(r.Context(), user.ShlinkAPIKey)
	if err != nil {
		slog.Error("dashboard: get short-urls failed", "sub", user.Sub, "err", err)
		writeJSON(w, map[string]string{"error": "shlink unavailable"}, http.StatusBadGateway)
		return
	}

	if !p.CanViewAllLinks {
		urls = []shlink.ShortURL{}
	}

	// --- visits ---
	now := time.Now()
	start := now.AddDate(0, 0, -days)
	visitsResp, err := h.shlinkSvc.Client().GetVisits(
		r.Context(), user.ShlinkAPIKey,
		start.Format(time.RFC3339), now.Format(time.RFC3339), 1000,
	)
	if err != nil {
		slog.Warn("dashboard: get visits failed", "sub", user.Sub, "err", err)
	}

	// --- users (admin only) ---
	var usersData any
	if p.CanManageUsers {
		users, err := h.userRepo.List(r.Context())
		if err != nil {
			slog.Warn("dashboard: list users failed", "sub", user.Sub, "err", err)
		} else {
			type userItem struct {
				Sub      string `json:"sub"`
				Username string `json:"username"`
				Email    string `json:"email"`
				Role     string `json:"role"`
				Status   string `json:"status"`
			}
			items := make([]userItem, 0, len(users))
			for _, u := range users {
				items = append(items, userItem{
					Sub:      u.Sub,
					Username: u.Username,
					Email:    u.Email,
					Role:     string(u.Role),
					Status:   string(u.Status),
				})
			}
			usersData = items
		}
	}

	resp := map[string]any{
		"overview": buildOverview(urls),
		"visits":   buildVisits(visitsResp, days),
		"devices":  buildDevices(visitsResp),
		"users":    usersData,
	}

	writeJSON(w, resp, http.StatusOK)
}

type ClickPoint struct {
	Date   string `json:"date"`
	Clicks int    `json:"clicks"`
}

type namedCount struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

type topLinkItem struct {
	ShortCode   string `json:"shortCode"`
	ShortURL    string `json:"shortUrl"`
	LongURL     string `json:"longUrl"`
	Title       string `json:"title"`
	VisitsTotal int    `json:"visitsTotal"`
}

func topCountSlice(m map[string]int, n int) []namedCount {
	slice := make([]namedCount, 0, len(m))
	for k, v := range m {
		slice = append(slice, namedCount{Name: k, Count: v})
	}
	sort.Slice(slice, func(i, j int) bool { return slice[i].Count > slice[j].Count })
	if len(slice) > n {
		slice = slice[:n]
	}
	return slice
}

func buildOverview(urls []shlink.ShortURL) map[string]any {
	visitsTotal := 0
	for _, u := range urls {
		visitsTotal += u.VisitsSummary.Total
	}

	sorted := make([]shlink.ShortURL, len(urls))
	copy(sorted, urls)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].VisitsSummary.Total > sorted[j].VisitsSummary.Total
	})

	top := make([]topLinkItem, 0, len(sorted))
	for _, u := range sorted {
		top = append(top, topLinkItem{
			ShortCode:   u.ShortCode,
			ShortURL:    u.ShortURL,
			LongURL:     u.LongURL,
			Title:       u.Title,
			VisitsTotal: u.VisitsSummary.Total,
		})
	}
	if len(top) > 10 {
		top = top[:10]
	}

	recent := make([]topLinkItem, 0, len(urls))
	for _, u := range urls {
		recent = append(recent, topLinkItem{
			ShortCode:   u.ShortCode,
			ShortURL:    u.ShortURL,
			LongURL:     u.LongURL,
			Title:       u.Title,
			VisitsTotal: u.VisitsSummary.Total,
		})
	}
	if len(recent) > 5 {
		recent = recent[:5]
	}

	return map[string]any{
		"linksCount":  len(urls),
		"visitsTotal": visitsTotal,
		"topLinks":    top,
		"recentLinks": recent,
	}
}

func buildVisits(visits *shlink.VisitsResponse, days int) map[string]any {
	const dayFmt = "2006-01-02"
	now := time.Now()

	buckets := make(map[string]int, days)
	ordered := make([]string, 0, days)
	for i := days - 1; i >= 0; i-- {
		d := now.AddDate(0, 0, -i).Format(dayFmt)
		buckets[d] = 0
		ordered = append(ordered, d)
	}

	total := 0
	if visits != nil {
		for _, v := range visits.Visits.Data {
			t, err := time.Parse(time.RFC3339, v.Date)
			if err != nil {
				continue
			}
			d := t.Format(dayFmt)
			if _, ok := buckets[d]; ok {
				buckets[d]++
				total++
			}
		}
	}

	points := make([]ClickPoint, 0, days)
	for _, d := range ordered {
		points = append(points, ClickPoint{Date: d, Clicks: buckets[d]})
	}

	return map[string]any{
		"clicksPerDay": points,
		"clicksTotal":  total,
	}
}

func buildDevices(visits *shlink.VisitsResponse) map[string]any {
	desktop, mobile, tablet := 0, 0, 0
	browsersMap := map[string]int{}
	osMap := map[string]int{}
	heatmap := map[string]int{}

	if visits != nil {
		for _, v := range visits.Visits.Data {
			ua := v.UserAgent
			uaLower := strings.ToLower(ua)
			switch {
			case strings.Contains(uaLower, "mobi") || strings.Contains(uaLower, "android"):
				mobile++
			case strings.Contains(uaLower, "tablet") || strings.Contains(uaLower, "ipad"):
				tablet++
			default:
				desktop++
			}
			browsersMap[urlDetailBrowser(uaLower)]++
			osMap[urlDetailOS(uaLower)]++

			t, perr := time.Parse(time.RFC3339, v.Date)
			if perr == nil {
				wd := (int(t.Weekday()) + 6) % 7
				hr := t.Hour()
				heatmap[fmt.Sprintf("%d-%d", wd, hr)]++
			}
		}
	}

	type heatCell struct {
		Weekday int `json:"weekday"`
		Hour    int `json:"hour"`
		Value   int `json:"value"`
	}
	cells := make([]heatCell, 0)
	for wd := 0; wd < 7; wd++ {
		for hr := 0; hr < 24; hr++ {
			v := heatmap[fmt.Sprintf("%d-%d", wd, hr)]
			if v > 0 {
				cells = append(cells, heatCell{wd, hr, v})
			}
		}
	}

	return map[string]any{
		"devices": map[string]int{
			"desktop": desktop,
			"mobile":  mobile,
			"tablet":  tablet,
		},
		"browsers": topCountSlice(browsersMap, 10),
		"os":       topCountSlice(osMap, 10),
		"heatmap":  cells,
	}
}
