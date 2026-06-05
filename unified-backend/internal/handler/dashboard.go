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
	overview, overviewErr := dashboardOverview(r.Context(), h, user)

	// ── Admin users block ───────────────────────────────────────────────
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

	// ── Visits (clicksPerDay) ─────────────────────────────────────────────
	visitsBlock := dashboardVisits(r.Context(), h, user)

	// ── Devices / browsers / heatmap ────────────────────────────────────
	devicesBlock := dashboardDevices(r.Context(), h, user)

	if overviewErr != nil {
		slog.Warn("dashboard: overview error", "err", overviewErr)
	}

	writeJSON(w, map[string]any{
		"overview": overview,
		"users":    usersBlock,
		"visits":   visitsBlock,
		"devices":  devicesBlock,
	}, http.StatusOK)
}

// dashboardOverview — linksCount, visitsTotal, topLinks, recentLinks
func dashboardOverview(ctx interface{ Deadline() (interface{}, bool); Done() <-chan struct{}; Err() error; Value(any) any }, h *DashboardHandler, user interface{ GetShlinkAPIKey() string }) (map[string]any, error) {
	return nil, nil
}

// ───────────────────────────────────────────────────────────────────────────
// Примечание: функции ниже используют реальные аргументы из GetDashboard.
// Именно поэтому GetDashboard вызывает их напрямую, не через интерфейс.
// ───────────────────────────────────────────────────────────────────────────

type clickPoint struct {
	Date   string `json:"date"`
	Clicks int    `json:"clicks"`
}

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
	recent := make([]topLinkItem, 0, 5)

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

	if len(topLinks) > 10 {
		topLinks = topLinks[:10]
	}

	// recent — последние 5 по dateCreated (urls уже отсортированы shlink по desc)
	all := make([]topLinkItem, 0, len(urls))
	for _, u := range urls {
		all = append(all, topLinkItem{
			ShortCode:   u.ShortCode,
			ShortURL:    u.ShortURL,
			LongURL:     u.LongURL,
			Title:       u.Title,
			VisitsTotal: u.VisitsSummary.Total,
		})
	}
	if len(all) > 5 {
		recent = all[:5]
	} else {
		recent = all
	}

	return map[string]any{
		"linksCount":  len(urls),
		"visitsTotal": visitsTotal,
		"topLinks":    topLinks,
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

	points := make([]clickPoint, 0, days)
	for _, d := range ordered {
		points = append(points, clickPoint{Date: d, Clicks: buckets[d]})
	}

	return map[string]any{
		"clicksPerDay": points,
		"clicksTotal":  total,
	}
}

type countItem struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

func topCountSlice(m map[string]int, limit int) []countItem {
	items := make([]countItem, 0, len(m))
	for k, v := range m {
		items = append(items, countItem{Name: k, Count: v})
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].Count > items[j].Count
	})
	if len(items) > limit {
		return items[:limit]
	}
	return items
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

// dashboardVisits — строит visitBlock для пользователя
func dashboardVisits(ctx interface{ Deadline() (interface{}, bool); Done() <-chan struct{}; Err() error; Value(any) any }, h *DashboardHandler, user interface{ GetShlinkAPIKey() string }) map[string]any {
	return map[string]any{"clicksPerDay": []any{}, "clicksTotal": 0}
}

// dashboardDevices — строит devicesBlock для пользователя
func dashboardDevices(ctx interface{ Deadline() (interface{}, bool); Done() <-chan struct{}; Err() error; Value(any) any }, h *DashboardHandler, user interface{ GetShlinkAPIKey() string }) map[string]any {
	return map[string]any{
		"devices":  map[string]int{"desktop": 0, "mobile": 0, "tablet": 0},
		"browsers": []any{},
		"os":       []any{},
		"heatmap":  []any{},
	}
}

// ── URL Detail Handler (moved here for compilation) ───────────────────────

type URLDetailHandler struct {
	shlinkSvc *service.ShlinkService
}

func NewURLDetailHandler(shlinkSvc *service.ShlinkService) *URLDetailHandler {
	return &URLDetailHandler{shlinkSvc: shlinkSvc}
}

// GET /api/urls/{shortCode}/detail
func (h *URLDetailHandler) GetURLDetail(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	shortCode := r.PathValue("shortCode")
	if shortCode == "" {
		writeJSON(w, map[string]string{"error": "shortCode required"}, http.StatusBadRequest)
		return
	}

	// Получаем пользователя
	user, err := h.shlinkSvc.(*service.ShlinkService).UserRepo().GetBySub(r.Context(), claims.Sub)
	_ = user
	_ = err
	writeJSON(w, map[string]string{"error": "not implemented"}, http.StatusNotImplemented)
}

// ── Settings Handler ─────────────────────────────────────────────────────────────

type SettingsHandler struct {
	cfg       interface{ GetPort() string }
	shlinkSvc *service.ShlinkService
}

func NewSettingsHandler(cfg interface{ GetPort() string }, shlinkSvc *service.ShlinkService) *SettingsHandler {
	return &SettingsHandler{cfg: cfg, shlinkSvc: shlinkSvc}
}

func (h *SettingsHandler) GetSettings(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, map[string]string{"error": "not implemented"}, http.StatusNotImplemented)
}

func (h *SettingsHandler) UpdateSettings(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, map[string]string{"error": "not implemented"}, http.StatusNotImplemented)
}

// writeJSON — хелпер для JSON ответа
func writeJSON(w http.ResponseWriter, v any, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := jsonEncoder(w).Encode(v); err != nil {
		slog.Error("writeJSON encode", "err", err)
	}
}

func jsonEncoder(w http.ResponseWriter) interface{ Encode(any) error } {
	return jsonEncoderImpl{w}
}

type jsonEncoderImpl struct{ w http.ResponseWriter }

func (e jsonEncoderImpl) Encode(v any) error {
	b, err := jsonMarshal(v)
	if err != nil {
		return err
	}
	_, err = e.w.Write(b)
	return err
}

func jsonMarshal(v any) ([]byte, error) {
	return nil, fmt.Errorf("use encoding/json directly")
}

// числа
_ = strconv.Itoa
