package handler

import (
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"unified-backend/internal/middleware"
	"unified-backend/internal/service"
	"unified-backend/internal/shlink"
)

type URLDetailHandler struct {
	svc       *service.ShlinkService
	ownerRepo OwnershipRepo
}

func NewURLDetailHandler(svc *service.ShlinkService, ownerRepo OwnershipRepo) *URLDetailHandler {
	return &URLDetailHandler{svc: svc, ownerRepo: ownerRepo}
}

type urlDetailResponse struct {
	ShortCode    string          `json:"shortCode"`
	Title        string          `json:"title"`
	ShortURL     string          `json:"shortUrl"`
	LongURL      string          `json:"longUrl"`
	DateCreated  string          `json:"dateCreated"`
	VisitsTotal  int             `json:"visitsTotal"`
	ClicksPerDay []ClickPoint    `json:"clicksPerDay"`
	Devices      deviceBreakdown `json:"devices"`
	Browsers     []namedCount    `json:"browsers"`
	OS           []namedCount    `json:"os"`
	Visits       []visitRow      `json:"visits"`
	IsActive     bool            `json:"isActive"`
	DeactivatedAt *string        `json:"deactivatedAt,omitempty"`
	DeactivatedBy *string        `json:"deactivatedBy,omitempty"`
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

// GET /api/urls/{shortCode}/detail
func (h *URLDetailHandler) GetURLDetail(w http.ResponseWriter, r *http.Request) {
	shortCode := chi.URLParam(r, "shortCode")
	if shortCode == "" {
		writeJSON(w, map[string]string{"error": "shortCode required"}, http.StatusBadRequest)
		return
	}

	user := middleware.UserFromCtx(r.Context())
	if user == nil || user.ShlinkAPIKey == "" {
		writeJSON(w, map[string]string{"error": "user or API key missing"}, http.StatusUnauthorized)
		return
	}

	query := r.URL.Query()
	period := 30
	if ps := query.Get("period"); ps != "" {
		if n, err := strconv.Atoi(ps); err == nil && n > 0 && n <= 365 {
			period = n
		}
	}

	info, err := h.svc.Client().GetShortURL(r.Context(), user.ShlinkAPIKey, shortCode)
	if err != nil {
		slog.Warn("url detail: get short url failed", "shortCode", shortCode, "err", err)
		writeJSON(w, map[string]string{"error": "not found"}, http.StatusNotFound)
		return
	}

	perms := h.svc.Perms(user)
	if !perms.CanViewAllLinks {
		if !perms.CanViewOwnLinks {
			writeJSON(w, map[string]string{"error": "forbidden"}, http.StatusForbidden)
			return
		}
		isOwner, ownerErr := h.ownerRepo.IsOwner(r.Context(), shortCode, "", user.Sub)
		if ownerErr != nil {
			slog.Error("url detail: ownership check failed", "shortCode", shortCode, "err", ownerErr)
			writeJSON(w, map[string]string{"error": "internal error"}, http.StatusInternalServerError)
			return
		}
		if !isOwner {
			writeJSON(w, map[string]string{"error": "forbidden"}, http.StatusForbidden)
			return
		}
	}

	isActive := true
	var deactivatedAt, deactivatedBy *string
	if ownership, owErr := h.ownerRepo.GetOwnership(r.Context(), shortCode, ""); owErr == nil && ownership != nil {
		isActive = ownership.IsActive
		if ownership.DeactivatedAt != nil {
			s := ownership.DeactivatedAt.Format(time.RFC3339)
			deactivatedAt = &s
		}
		deactivatedBy = ownership.DeactivatedBy
	}

	end := time.Now()
	start := end.AddDate(0, 0, -period)
	visitsResp, err := h.svc.Client().GetShortURLVisits(
		r.Context(),
		user.ShlinkAPIKey,
		shortCode,
		start.Format(time.RFC3339),
		end.Format(time.RFC3339),
		1000,
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
		Devices: deviceBreakdown{
			Desktop: desktop,
			Mobile:  mobile,
			Tablet:  tablet,
		},
		Browsers:      topCountSlice(browsersMap, 10),
		OS:            topCountSlice(osMap, 10),
		Visits:        visitRows,
		IsActive:      isActive,
		DeactivatedAt: deactivatedAt,
		DeactivatedBy: deactivatedBy,
	}

	writeJSON(w, resp, http.StatusOK)
}

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
