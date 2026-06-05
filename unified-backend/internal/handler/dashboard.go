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
	claims := middleware.ClaimsFromContext(r.Context())

	user, err := h.userRepo.GetBySub(r.Context(), claims.Sub)
	if err != nil || user == nil {
		writeJSON(w, map[string]string{"error": "user not found"}, http.StatusUnauthorized)
		return
	}

	if user.ShlinkAPIKey == "" {
		writeJSON(w, map[string]any{
			"overview": map[string]any{
				"linksCount":  0,
				"visitsTotal": 0,
				"topLinks":    []any{},
				"recentLinks": []any{},
			},
			"users":   nil,
			"visits":  map[string]any{"clicksPerDay": []any{}, "clicksTotal": 0},
			"devices": map[string]any{"devices": map[string]int{"desktop": 0, "mobile": 0, "tablet": 0}, "browsers": []any{}, "os": []any{}, "heatmap": []any{}},
		}, http.StatusOK)
		return
	}

	p := h.shlinkSvc.Perms(user)

	// ── Overview ────────────────────────────────────────────────────────────
	urlsResp, err := h.shlinkSvc.Client().GetShortURLs(r.Context(), user.ShlinkAPIKey, "itemsPerPage=1000")
	var urls []shlink.ShortURL
	if err != nil {
		slog.Warn("dashboard: get short urls failed", "err", err)
	} else if urlsResp != nil {
		urls = h.shlinkSvc.FilterShortURLsByUser(urlsResp.ShortURLs.Data, user)
	}
	overview := buildOverview(urls)

	// ── Admin users block ────────────────────────────────────────────────────
	var usersBlock any
	if p.CanViewAllLinks {
		users, err := h.userRepo.ListAll(r.Context())
		if err != nil {
			slog.Warn("dashboard: list users failed", "err", err)
		} else {
			type userRow struct {
				Sub      string `json:"sub"`
				Username string `json:"username"`
				Email    string `json:"email"`
				Role     string `json:"role"`
				Status   string `json:"status"`
			}
			rows := make([]userRow, 0, len(users))
			for _, u := range users {
				rows = append(rows, userRow{
					Sub:      u.Sub,
					Username: u.Username,
					Email:    u.Email,
					Role:     string(u.Role),
					Status:   string(u.Status),
				})
			}
			usersBlock = rows
		}
	}

	// ── Visits (clicksPerDay 30d) ────────────────────────────────────────────
	endDate := time.Now().Format(time.RFC3339)
	startDate := time.Now().AddDate(0, 0, -30).Format(time.RFC3339)
	visits, err := h.shlinkSvc.Client().GetNonOrphanVisits(r.Context(), user.ShlinkAPIKey, startDate, endDate, 1000)
	if err != nil {
		slog.Warn("dashboard: get visits failed", "err", err)
	}
	visitsBlock := buildVisits(visits, 30)

	// ── Devices / browsers / heatmap ────────────────────────────────────────
	devicesBlock := buildDevices(visits)

	writeJSON(w, map[string]any{
		"overview": overview,
		"users":    usersBlock,
		"visits":   visitsBlock,
		"devices":  devicesBlock,
	}, http.StatusOK)
}

// ─────────────────────────────────────────────────────────────────────────────

type topLinkItem struct {
	ShortCode   string `json:"shortCode"`
	ShortURL    string `json:"shortUrl"`
	LongURL     string `json:"longUrl"`
	Title       string `json:"title"`
	VisitsTotal int    `json:"visitsTotal"`
}

func buildOverview(urls []shlink.ShortURL) map[string]any {
	visitsTotal := 0
	topLinks := make([]topLinkItem, 0, len(urls))

	for _, u := range urls {
		visitsTotal += u.VisitsSummary.Total
		topLinks = append(topLinks, topLinkItem{
			ShortCode:   u.ShortCode,
			ShortURL:    u.ShortURL,
			LongURL:     u.LongURL,
			Title:       u.Title,
			VisitsTotal: u.VisitsSummary.Total,
		})
	}

	sort.Slice(topLinks, func(i, j int) bool {
		return topLinks[i].VisitsTotal > topLinks[j].VisitsTotal
	})

	top := topLinks
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
			browsersMap[urlDetailParseBrowser(ua)]++
			osMap[urlDetailParseOS(ua)]++

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
