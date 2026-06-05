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
)

// URLDetailHandler отвечает за GET /api/urls/{shortCode}/detail
type URLDetailHandler struct {
	shlinkSvc *service.ShlinkService
}

func NewURLDetailHandler(svc *service.ShlinkService) *URLDetailHandler {
	return &URLDetailHandler{shlinkSvc: svc}
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
}

type namedCount struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

type deviceBreakdown struct {
	Desktop int `json:"desktop"`
	Mobile  int `json:"mobile"`
	Tablet  int `json:"tablet"`
}

type visitRow struct {
	Date    string  `json:"date"`
	Device  string  `json:"device"`
	OS      string  `json:"os"`
	Browser string  `json:"browser"`
	Country *string `json:"country"`
	Referer *string `json:"referer"`
}

// GET /api/urls/{shortCode}/detail?period=30
func (h *URLDetailHandler) GetURLDetail(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromCtx(r.Context())
	if user == nil {
		writeJSON(w, map[string]string{"error": "forbidden"}, http.StatusForbidden)
		return
	}

	shortCode := chi.URLParam(r, "shortCode")
	if shortCode == "" {
		writeJSON(w, map[string]string{"error": "shortCode required"}, http.StatusBadRequest)
		return
	}

	period, _ := strconv.Atoi(r.URL.Query().Get("period"))
	if period <= 0 {
		period = 30
	}

	now := time.Now()
	endDate := now.Format("2006-01-02")
	startDate := now.AddDate(0, 0, -(period - 1)).Format("2006-01-02")

	// Метаданные ссылки
	urlInfo, err := h.shlinkSvc.Client().GetShortURL(r.Context(), user.ShlinkAPIKey, shortCode)
	if err != nil {
		slog.Error("url_detail: get short-url failed", "shortCode", shortCode, "err", err)
		writeJSON(w, map[string]string{"error": "not found"}, http.StatusNotFound)
		return
	}

	// Визиты
	visitsResp, err := h.shlinkSvc.Client().GetShortURLVisits(r.Context(), user.ShlinkAPIKey, shortCode, startDate, endDate, 0)
	if err != nil {
		slog.Warn("url_detail: get visits failed", "shortCode", shortCode, "err", err)
	}

	// Строим buckets для clicksPerDay
	const dayFmt = "2006-01-02"
	buckets := make(map[string]int, period)
	ordered := make([]string, 0, period)
	for i := period - 1; i >= 0; i-- {
		d := now.AddDate(0, 0, -i).Format(dayFmt)
		buckets[d] = 0
		ordered = append(ordered, d)
	}

	browsersMap := map[string]int{}
	osMap := map[string]int{}
	dev := deviceBreakdown{}
	var rows []visitRow

	if visitsResp != nil {
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

			ua := strings.ToLower(v.UserAgent)
			devType := urlDetailDetectDevice(ua)
			switch devType {
			case "mobile":
				dev.Mobile++
			case "tablet":
				dev.Tablet++
			default:
				dev.Desktop++
			}

			browsersMap[urlDetailParseBrowser(ua)]++
			osMap[urlDetailParseOS(ua)]++

			country := nullStrPtr(v.VisitLocation.CountryName)
			referer := nullStrPtr(v.Referer)
			rows = append(rows, visitRow{
				Date:    v.Date,
				Device:  devType,
				OS:      urlDetailParseOS(ua),
				Browser: urlDetailParseBrowser(ua),
				Country: country,
				Referer: referer,
			})
		}
	}

	points := make([]ClickPoint, 0, period)
	for _, d := range ordered {
		points = append(points, ClickPoint{Date: d, Clicks: buckets[d]})
	}

	if rows == nil {
		rows = []visitRow{}
	}

	writeJSON(w, urlDetailResponse{
		ShortCode:    urlInfo.ShortCode,
		Title:        urlInfo.Title,
		ShortURL:     urlInfo.ShortURL,
		LongURL:      urlInfo.LongURL,
		DateCreated:  urlInfo.DateCreated,
		VisitsTotal:  urlInfo.VisitsSummary.Total,
		ClicksPerDay: points,
		Devices:      dev,
		Browsers:     topCountSlice(browsersMap, 10),
		OS:           topCountSlice(osMap, 10),
		Visits:       rows,
	}, http.StatusOK)
}

func nullStrPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func urlDetailDetectDevice(ua string) string {
	if strings.Contains(ua, "mobile") || (strings.Contains(ua, "android") && !strings.Contains(ua, "tablet")) {
		return "mobile"
	}
	if strings.Contains(ua, "tablet") || strings.Contains(ua, "ipad") {
		return "tablet"
	}
	return "desktop"
}

func urlDetailParseBrowser(ua string) string {
	switch {
	case strings.Contains(ua, "edg"):
		return "Edge"
	case strings.Contains(ua, "chrome") && !strings.Contains(ua, "chromium"):
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
	switch {
	case strings.Contains(ua, "windows"):
		return "Windows"
	case strings.Contains(ua, "android"):
		return "Android"
	case strings.Contains(ua, "iphone") || strings.Contains(ua, "ipad"):
		return "iOS"
	case strings.Contains(ua, "mac"):
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
	// simple insertion sort (small slices)
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && out[j].Count > out[j-1].Count; j-- {
			out[j], out[j-1] = out[j-1], out[j]
		}
	}
	if len(out) > n {
		out = out[:n]
	}
	return out
}
