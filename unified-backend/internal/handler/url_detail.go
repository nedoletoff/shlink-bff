package handler

import (
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unified-backend/internal/controller"
	"unified-backend/internal/domain"
	"unified-backend/internal/middleware"
	"unified-backend/internal/service"
	"unified-backend/internal/shlink"

	"github.com/go-chi/chi/v5"
)

type URLDetailHandler struct {
	svc       *service.ShlinkService
	ownerRepo OwnershipRepo
	permCtrl  controller.PermChecker
}

func NewURLDetailHandler(
	svc *service.ShlinkService,
	ownerRepo OwnershipRepo,
	permCtrl controller.PermChecker,
) *URLDetailHandler {
	return &URLDetailHandler{svc: svc, ownerRepo: ownerRepo, permCtrl: permCtrl}
}

type urlDetailResponse struct {
	ShortCode     string          `json:"shortCode"`
	Title         string          `json:"title"`
	ShortURL      string          `json:"shortUrl"`
	LongURL       string          `json:"longUrl"`
	DateCreated   string          `json:"dateCreated"`
	VisitsTotal   int             `json:"visitsTotal"`
	ClicksPerDay  []ClickPoint    `json:"clicksPerDay"`
	Devices       deviceBreakdown `json:"devices"`
	Browsers      []namedCount    `json:"browsers"`
	OS            []namedCount    `json:"os"`
	Visits        []visitRow      `json:"visits"`
	IsActive      bool            `json:"isActive"`
	ValidSince    *string         `json:"validSince,omitempty"`
	ValidUntil    *string         `json:"validUntil,omitempty"`
	MaxVisits     int             `json:"maxVisits"`
	DeactivatedAt *string         `json:"deactivatedAt,omitempty"`
	DeactivatedBy *string         `json:"deactivatedBy,omitempty"`
}

type deviceBreakdown struct {
	Desktop int `json:"desktop"`
	Mobile  int `json:"mobile"`
	Tablet  int `json:"tablet"`
}

type visitRow struct {
	Date    string  `json:"date"`
	Country *string `json:"country,omitempty"`
	Referer *string `json:"referer,omitempty"`
	Browser string  `json:"browser"`
	OS      string  `json:"os"`
	Device  string  `json:"device"`
}

// GetURLDetail – GET /api/urls/{shortCode}/detail
func (h *URLDetailHandler) GetURLDetail(w http.ResponseWriter, r *http.Request) {
	shortCode := chi.URLParam(r, "shortCode")
	if shortCode == "" {
		writeJSON(w, map[string]string{"error": "shortCode required"}, http.StatusBadRequest)
		return
	}

	user := middleware.UserFromCtx(r.Context())
	if user == nil || user.ShlinkAPIKey == "" {
		writeJSON(w, map[string]string{"error": "unauthorized"}, http.StatusUnauthorized)
		return
	}

	// Проверка прав на просмотр статистики
	canViewAll, _ := h.permCtrl.Check(r.Context(), user.ID, domain.PermShortURLsViewStatsAll)
	canViewOwn, _ := h.permCtrl.Check(r.Context(), user.ID, domain.PermShortURLsViewStatsOwn)

	if !canViewAll && !canViewOwn {
		writeJSON(w, map[string]string{"error": "forbidden"}, http.StatusForbidden)
		return
	}
	if !canViewAll {
		isOwner, err := h.ownerRepo.IsOwner(r.Context(), shortCode, "", user.Sub)
		if err != nil || !isOwner {
			writeJSON(w, map[string]string{"error": "forbidden"}, http.StatusForbidden)
			return
		}
	}

	// Получаем данные из Shlink
	info, err := h.svc.Client().GetShortURL(r.Context(), user.ShlinkAPIKey, shortCode)
	if err != nil {
		writeJSON(w, map[string]string{"error": "not found"}, http.StatusNotFound)
		return
	}

	// Метаданные из БД
	meta, _ := h.ownerRepo.GetOwnership(r.Context(), shortCode, "")
	isActive := true
	var deactivatedAt, deactivatedBy *string
	var validSince, validUntil *string
	maxVisits := 0
	if meta != nil {
		isActive = meta.IsActive
		if meta.DeactivatedAt != nil {
			s := meta.DeactivatedAt.Format(time.RFC3339)
			deactivatedAt = &s
		}
		deactivatedBy = meta.DeactivatedBy
		if meta.ValidSince != nil {
			s := meta.ValidSince.Format(time.RFC3339)
			validSince = &s
		}
		if meta.ValidUntil != nil {
			s := meta.ValidUntil.Format(time.RFC3339)
			validUntil = &s
		}
		maxVisits = meta.MaxVisits
	}

	// Период для статистики
	period := 30
	if ps := r.URL.Query().Get("period"); ps != "" {
		if n, err := strconv.Atoi(ps); err == nil && n > 0 && n <= 365 {
			period = n
		}
	}
	end := time.Now()
	start := end.AddDate(0, 0, -period)

	visitsResp, err := h.svc.Client().GetShortURLVisits(
		r.Context(), user.ShlinkAPIKey, shortCode,
		start.Format(time.RFC3339), end.Format(time.RFC3339), 1000,
	)
	if err != nil {
		slog.Warn("url detail: get visits failed", "shortCode", shortCode, "err", err)
		writeJSON(w, map[string]string{"error": "failed to load visits"}, http.StatusBadGateway)
		return
	}

	var visits []shlink.Visit
	if visitsResp != nil {
		visits = visitsResp.Visits.Data
	}

	// Агрегация визитов
	const dayFmt = "2006-01-02"
	buckets := make(map[string]int, period)
	ordered := make([]string, 0, period)
	for i := period - 1; i >= 0; i-- {
		d := end.AddDate(0, 0, -i).Format(dayFmt)
		buckets[d] = 0
		ordered = append(ordered, d)
	}

	desktop, mobile, tablet := 0, 0, 0
	browsersMap := map[string]int{}
	osMap := map[string]int{}
	visitRows := make([]visitRow, 0, len(visits))

	for _, v := range visits {
		t, err := time.Parse(time.RFC3339, v.Date)
		if err == nil {
			d := t.Format(dayFmt)
			if _, ok := buckets[d]; ok {
				buckets[d]++
			}
		}
		ua := strings.ToLower(v.UserAgent)
		dev := urlDetailDevice(ua)
		switch dev {
		case "mobile":
			mobile++
		case "tablet":
			tablet++
		default:
			desktop++
		}
		browsersMap[urlDetailBrowser(ua)]++
		osMap[urlDetailOS(ua)]++

		visitRows = append(visitRows, visitRow{
			Date:    v.Date,
			Country: urlDetailNullStr(v.VisitLocation.CountryName),
			Referer: urlDetailNullStr(v.Referer),
			Browser: urlDetailBrowser(ua),
			OS:      urlDetailOS(ua),
			Device:  dev,
		})
	}

	points := make([]ClickPoint, 0, period)
	for _, d := range ordered {
		points = append(points, ClickPoint{Date: d, Clicks: buckets[d]})
	}

	resp := urlDetailResponse{
		ShortCode:     info.ShortCode,
		Title:         info.Title,
		ShortURL:      info.ShortURL,
		LongURL:       info.LongURL,
		DateCreated:   info.DateCreated,
		VisitsTotal:   info.VisitsSummary.Total,
		ClicksPerDay:  points,
		Devices:       deviceBreakdown{Desktop: desktop, Mobile: mobile, Tablet: tablet},
		Browsers:      topCountSlice(browsersMap, 10),
		OS:            topCountSlice(osMap, 10),
		Visits:        visitRows,
		IsActive:      isActive,
		ValidSince:    validSince,
		ValidUntil:    validUntil,
		MaxVisits:     maxVisits,
		DeactivatedAt: deactivatedAt,
		DeactivatedBy: deactivatedBy,
	}
	writeJSON(w, resp, http.StatusOK)
}

// helpers
func urlDetailNullStr(s string) *string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	return &s
}

func urlDetailDevice(ua string) string {
	if strings.Contains(ua, "mobi") || strings.Contains(ua, "android") {
		return "mobile"
	}
	if strings.Contains(ua, "tablet") || strings.Contains(ua, "ipad") {
		return "tablet"
	}
	return "desktop"
}

func urlDetailBrowser(ua string) string {
	switch {
	case strings.Contains(ua, "firefox"):
		return "Firefox"
	case strings.Contains(ua, "chrome") && !strings.Contains(ua, "chromium"):
		return "Chrome"
	case strings.Contains(ua, "safari") && !strings.Contains(ua, "chrome"):
		return "Safari"
	case strings.Contains(ua, "edge"):
		return "Edge"
	case strings.Contains(ua, "opera") || strings.Contains(ua, "opr"):
		return "Opera"
	default:
		return "Other"
	}
}

func urlDetailOS(ua string) string {
	switch {
	case strings.Contains(ua, "windows"):
		return "Windows"
	case strings.Contains(ua, "mac os"):
		return "macOS"
	case strings.Contains(ua, "linux"):
		return "Linux"
	case strings.Contains(ua, "android"):
		return "Android"
	case strings.Contains(ua, "iphone") || strings.Contains(ua, "ipad"):
		return "iOS"
	default:
		return "Other"
	}
}

