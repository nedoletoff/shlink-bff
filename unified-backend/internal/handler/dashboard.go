package handler

import (
	"fmt"
	"log/slog"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"unified-backend/internal/middleware"
	"unified-backend/internal/repository/postgres"
	"unified-backend/internal/service"
)

type DashboardHandler struct {
	shlinkSvc *service.ShlinkService
	userRepo  *postgres.UserRepository
}

func NewDashboardHandler(svc *service.ShlinkService, userRepo *postgres.UserRepository) *DashboardHandler {
	return &DashboardHandler{shlinkSvc: svc, userRepo: userRepo}
}

type TagCount struct {
	Tag   string `json:"tag"`
	Count int    `json:"count"`
}

type ClickPoint struct {
	Date   string `json:"date"`
	Clicks int    `json:"clicks"`
}

// GET /api/dashboard/overview?period=N
func (h *DashboardHandler) GetOverview(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromCtx(r.Context())
	if user == nil {
		writeJSON(w, map[string]string{"error": "forbidden"}, http.StatusForbidden)
		return
	}

	period, _ := strconv.Atoi(r.URL.Query().Get("period"))
	if period <= 0 {
		period = 30
	}

	urlsResp, err := h.shlinkSvc.Client().GetShortURLs(r.Context(), user.ShlinkAPIKey, "itemsPerPage=1000")
	if err != nil {
		slog.Error("dashboard/overview: get short-urls failed", "err", err)
		writeJSON(w, map[string]string{"error": "shlink unavailable"}, http.StatusBadGateway)
		return
	}

	urls := urlsResp.ShortURLs.Data
	var totalClicks int
	for _, u := range urls {
		totalClicks += u.VisitsSummary.Total
	}

	cutoff := time.Now().AddDate(0, 0, -period)
	createdPeriod := 0
	for _, u := range urls {
		t, err := time.Parse(time.RFC3339, u.DateCreated)
		if err == nil && t.After(cutoff) {
			createdPeriod++
		}
	}

	type topLink struct {
		ShortCode string `json:"shortCode"`
		Title     string `json:"title"`
		ShortURL  string `json:"shortUrl"`
		Visits    int    `json:"visits"`
	}
	topLinks := make([]topLink, 0, len(urls))
	for _, u := range urls {
		topLinks = append(topLinks, topLink{
			ShortCode: u.ShortCode,
			Title:     u.Title,
			ShortURL:  u.ShortURL,
			Visits:    u.VisitsSummary.Total,
		})
	}
	sort.Slice(topLinks, func(i, j int) bool { return topLinks[i].Visits > topLinks[j].Visits })
	if len(topLinks) > 10 {
		topLinks = topLinks[:10]
	}

	clicksPerDay := h.buildClicksOverTimeN(r, user.ShlinkAPIKey, period)

	writeJSON(w, map[string]any{
		"totalClicks":   totalClicks,
		"activeLinks":   len(urls),
		"createdPeriod": createdPeriod,
		"clicksPerDay":  clicksPerDay,
		"topLinks":      topLinks,
	}, http.StatusOK)
}

// GET /api/dashboard/users?period=N
func (h *DashboardHandler) GetUsersActivity(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromCtx(r.Context())
	if user == nil {
		writeJSON(w, map[string]string{"error": "forbidden"}, http.StatusForbidden)
		return
	}

	p := h.shlinkSvc.Perms(user)
	if !p.CanManageUsers {
		writeJSON(w, map[string]string{"error": "forbidden"}, http.StatusForbidden)
		return
	}

	period, _ := strconv.Atoi(r.URL.Query().Get("period"))
	if period <= 0 {
		period = 30
	}

	dbUsers, err := h.userRepo.ListAll(r.Context())
	if err != nil {
		slog.Error("dashboard/users: list users failed", "err", err)
		writeJSON(w, map[string]string{"error": "internal error"}, http.StatusInternalServerError)
		return
	}

	urlsResp, err := h.shlinkSvc.Client().GetShortURLs(r.Context(), user.ShlinkAPIKey, "itemsPerPage=1000")
	if err != nil {
		slog.Warn("dashboard/users: get short-urls failed", "err", err)
	}

	type prefixStat struct {
		LinksCount  int
		VisitsTotal int
		LastDate    time.Time
	}
	prefixStats := map[string]*prefixStat{}
	if urlsResp != nil {
		for _, u := range urlsResp.ShortURLs.Data {
			for _, dbU := range dbUsers {
				if dbU.SlugPrefix != "" && strings.HasPrefix(u.ShortCode, dbU.SlugPrefix) {
					ps := prefixStats[dbU.Sub]
					if ps == nil {
						ps = &prefixStat{}
						prefixStats[dbU.Sub] = ps
					}
					ps.LinksCount++
					ps.VisitsTotal += u.VisitsSummary.Total
					if t, e := time.Parse(time.RFC3339, u.DateCreated); e == nil && t.After(ps.LastDate) {
						ps.LastDate = t
					}
					break
				}
			}
		}
	}

	type userActivityRow struct {
		Sub            string  `json:"sub"`
		Username       string  `json:"username"`
		LinksCount     int     `json:"linksCount"`
		VisitsCount    int     `json:"visitsCount"`
		LastActivityAt *string `json:"lastActivityAt"`
	}

	rows := make([]userActivityRow, 0, len(dbUsers))
	for _, dbU := range dbUsers {
		var lastAt *string
		linksCount := 0
		visitsTotal := 0
		if ps, ok := prefixStats[dbU.Sub]; ok {
			linksCount = ps.LinksCount
			visitsTotal = ps.VisitsTotal
			if !ps.LastDate.IsZero() {
				s := ps.LastDate.Format(time.RFC3339)
				lastAt = &s
			}
		}
		rows = append(rows, userActivityRow{
			Sub:            dbU.Sub,
			Username:       dbU.Username,
			LinksCount:     linksCount,
			VisitsCount:    visitsTotal,
			LastActivityAt: lastAt,
		})
	}

	const dayFmt = "2006-01-02"
	now := time.Now()
	newLinksBuckets := map[string]int{}
	newLinksOrdered := make([]string, 0, period)
	for i := period - 1; i >= 0; i-- {
		d := now.AddDate(0, 0, -i).Format(dayFmt)
		newLinksBuckets[d] = 0
		newLinksOrdered = append(newLinksOrdered, d)
	}
	if urlsResp != nil {
		for _, u := range urlsResp.ShortURLs.Data {
			if t, e := time.Parse(time.RFC3339, u.DateCreated); e == nil {
				d := t.Format(dayFmt)
				if _, ok := newLinksBuckets[d]; ok {
					newLinksBuckets[d]++
				}
			}
		}
	}
	newLinksPerDay := make([]ClickPoint, 0, period)
	for _, d := range newLinksOrdered {
		newLinksPerDay = append(newLinksPerDay, ClickPoint{Date: d, Clicks: newLinksBuckets[d]})
	}

	writeJSON(w, map[string]any{
		"users":          rows,
		"newLinksPerDay": newLinksPerDay,
	}, http.StatusOK)
}

// GET /api/dashboard/urls?period=N
func (h *DashboardHandler) GetUrlsStats(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromCtx(r.Context())
	if user == nil {
		writeJSON(w, map[string]string{"error": "forbidden"}, http.StatusForbidden)
		return
	}

	urlsResp, err := h.shlinkSvc.Client().GetShortURLs(r.Context(), user.ShlinkAPIKey, "itemsPerPage=1000")
	if err != nil {
		slog.Error("dashboard/urls: get short-urls failed", "err", err)
		writeJSON(w, map[string]string{"error": "shlink unavailable"}, http.StatusBadGateway)
		return
	}

	now := time.Now()
	startDate7d := now.AddDate(0, 0, -6).Format("2006-01-02")
	endDate := now.Format("2006-01-02")
	today := now.Format("2006-01-02")

	urls := urlsResp.ShortURLs.Data

	sorted := make([]int, len(urls))
	for i := range sorted {
		sorted[i] = i
	}
	sort.Slice(sorted, func(a, b int) bool {
		return urls[sorted[a]].VisitsSummary.Total > urls[sorted[b]].VisitsSummary.Total
	})

	todayMap := map[string]int{}
	days7Map := map[string]int{}

	limit := 50
	if len(sorted) < limit {
		limit = len(sorted)
	}
	for _, idx := range sorted[:limit] {
		u := urls[idx]
		if u.VisitsSummary.Total == 0 {
			continue
		}
		vr, verr := h.shlinkSvc.Client().GetShortURLVisits(
			r.Context(), user.ShlinkAPIKey, u.ShortCode, startDate7d, endDate, 1000,
		)
		if verr != nil {
			continue
		}
		var t7, td int
		for _, v := range vr.Visits.Data {
			t7++
			day := v.Date
			if len(day) >= 10 {
				day = day[:10]
			}
			if day == today {
				td++
			}
		}
		todayMap[u.ShortCode] = td
		days7Map[u.ShortCode] = t7
	}

	type urlStatRow struct {
		ShortCode   string   `json:"shortCode"`
		Title       string   `json:"title"`
		ShortURL    string   `json:"shortUrl"`
		VisitsToday int      `json:"visitsToday"`
		Visits7d    int      `json:"visits7d"`
		VisitsTotal int      `json:"visitsTotal"`
		Status      string   `json:"status"`
		Tags        []string `json:"tags"`
	}

	rows := make([]urlStatRow, 0, len(urls))
	for _, u := range urls {
		tags := u.Tags
		if tags == nil {
			tags = []string{}
		}
		rows = append(rows, urlStatRow{
			ShortCode:   u.ShortCode,
			Title:       u.Title,
			ShortURL:    u.ShortURL,
			VisitsToday: todayMap[u.ShortCode],
			Visits7d:    days7Map[u.ShortCode],
			VisitsTotal: u.VisitsSummary.Total,
			Status:      "active",
			Tags:        tags,
		})
	}

	writeJSON(w, map[string]any{"urls": rows}, http.StatusOK)
}

// GET /api/dashboard/devices?period=N
func (h *DashboardHandler) GetDevicesStats(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromCtx(r.Context())
	if user == nil {
		writeJSON(w, map[string]string{"error": "forbidden"}, http.StatusForbidden)
		return
	}

	period, _ := strconv.Atoi(r.URL.Query().Get("period"))
	if period <= 0 {
		period = 30
	}

	now := time.Now()
	startDate := now.AddDate(0, 0, -(period - 1)).Format("2006-01-02")
	endDate := now.Format("2006-01-02")

	visitsResp, err := h.shlinkSvc.Client().GetNonOrphanVisits(r.Context(), user.ShlinkAPIKey, startDate, endDate, 0)
	if err != nil {
		slog.Warn("dashboard/devices: visits unavailable", "err", err)
		writeJSON(w, map[string]any{
			"devices":  map[string]int{"desktop": 0, "mobile": 0, "tablet": 0},
			"browsers": []any{},
			"os":       []any{},
			"heatmap":  []any{},
		}, http.StatusOK)
		return
	}

	desktop, mobile, tablet := 0, 0, 0
	browsersMap := map[string]int{}
	osMap := map[string]int{}
	heatmap := map[string]int{}

	for _, v := range visitsResp.Visits.Data {
		ua := strings.ToLower(v.UserAgent)
		switch {
		case strings.Contains(ua, "mobile") || (strings.Contains(ua, "android") && !strings.Contains(ua, "tablet")):
			mobile++
		case strings.Contains(ua, "tablet") || strings.Contains(ua, "ipad"):
			tablet++
		default:
			desktop++
		}
		browsersMap[urlDetailParseBrowser(ua)]++
		osMap[urlDetailParseOS(ua)]++

		t, perr := time.Parse(time.RFC3339, v.Date)
		if perr == nil {
			wd := int(t.Weekday())
			hr := t.Hour()
			heatmap[fmt.Sprintf("%d-%d", wd, hr)]++
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

	writeJSON(w, map[string]any{
		"devices": map[string]int{
			"desktop": desktop,
			"mobile":  mobile,
			"tablet":  tablet,
		},
		"browsers": topCountSlice(browsersMap, 10),
		"os":       topCountSlice(osMap, 10),
		"heatmap":  cells,
	}, http.StatusOK)
}

// buildClicksOverTimeN строит ряд кликов за последние period дней.
func (h *DashboardHandler) buildClicksOverTimeN(r *http.Request, apiKey string, period int) []ClickPoint {
	const dayFmt = "2006-01-02"
	now := time.Now()
	buckets := make(map[string]int, period)
	ordered := make([]string, 0, period)
	for i := period - 1; i >= 0; i-- {
		d := now.AddDate(0, 0, -i).Format(dayFmt)
		buckets[d] = 0
		ordered = append(ordered, d)
	}
	startDate := now.AddDate(0, 0, -(period - 1)).Format(dayFmt)
	endDate := now.Format(dayFmt)
	visitsResp, err := h.shlinkSvc.Client().GetNonOrphanVisits(r.Context(), apiKey, startDate, endDate, 0)
	if err != nil {
		slog.Warn("dashboard: visits API unavailable", "err", err)
	} else {
		for _, v := range visitsResp.Visits.Data {
			t, perr := time.Parse(time.RFC3339, v.Date)
			day := v.Date
			if perr == nil {
				day = t.Format(dayFmt)
			} else if len(day) >= 10 {
				day = day[:10]
			}
			if _, ok := buckets[day]; ok {
				buckets[day]++
			}
		}
	}
	points := make([]ClickPoint, 0, period)
	for _, d := range ordered {
		points = append(points, ClickPoint{Date: d, Clicks: buckets[d]})
	}
	return points
}
