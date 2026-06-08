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
	ownerRepo OwnershipRepo
}

func NewDashboardHandler(
	shlinkSvc *service.ShlinkService,
	userRepo *postgres.UserRepository,
	ownerRepo OwnershipRepo,
) *DashboardHandler {
	return &DashboardHandler{shlinkSvc: shlinkSvc, userRepo: userRepo, ownerRepo: ownerRepo}
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

	// ── Short URLs ────────────────────────────────────────────────────────────
	urlsResp, err := h.shlinkSvc.Client().GetShortURLs(r.Context(), user.ShlinkAPIKey, "itemsPerPage=1000")
	var urls []shlink.ShortURL
	if err != nil {
		slog.Warn("dashboard: get short urls failed", "sub", user.Sub, "err", err)
	} else if urlsResp != nil {
		if p.CanViewAllLinks {
			urls = urlsResp.ShortURLs.Data
		} else if p.CanViewOwnLinks && h.ownerRepo != nil {
			ownedCodes, oErr := h.ownerRepo.GetShortCodeSet(r.Context(), user.Sub)
			if oErr != nil {
				slog.Warn("dashboard: get owned codes failed", "sub", user.Sub, "err", oErr)
			} else {
				for _, u := range urlsResp.ShortURLs.Data {
					if _, owned := ownedCodes[u.ShortCode]; owned {
						urls = append(urls, u)
					}
				}
			}
		}
	}
	overview := buildOverview(urls)

	// ── Visits ────────────────────────────────────────────────────────────────
	endDate := time.Now().Format(time.RFC3339)
	startDate := time.Now().AddDate(0, 0, -days).Format(time.RFC3339)
	var visitsResp *shlink.VisitsResponse
	if p.CanViewAllStats || p.CanViewOwnStats {
		if p.CanViewAllStats {
			v, verr := h.shlinkSvc.Client().GetNonOrphanVisits(r.Context(), user.ShlinkAPIKey, startDate, endDate, 1000)
			if verr != nil {
				slog.Warn("dashboard: get visits failed", "sub", user.Sub, "err", verr)
			} else {
				visitsResp = v
			}
		} else if h.ownerRepo != nil {
			ownedCodes, oErr := h.ownerRepo.GetShortCodeSet(r.Context(), user.Sub)
			if oErr == nil && len(ownedCodes) > 0 {
				var allVisits []shlink.Visit
				for sc := range ownedCodes {
					v, verr := h.shlinkSvc.Client().GetShortURLVisits(
						r.Context(), user.ShlinkAPIKey, sc, startDate, endDate, 1000,
					)
					if verr != nil {
						slog.Warn("dashboard: get url visits failed", "sub", user.Sub, "shortCode", sc, "err", verr)
						continue
					}
					if v != nil {
						allVisits = append(allVisits, v.Visits.Data...)
					}
				}
				if len(allVisits) > 0 {
					visitsResp = &shlink.VisitsResponse{}
					visitsResp.Visits.Data = allVisits
				}
			}
		}
	}
	visitsBlock := buildVisits(visitsResp, days)
	devicesBlock := buildDevices(visitsResp)

	// ── Admin: users list ─────────────────────────────────────────────────────
	var usersBlock any
	if p.CanManageUsers {
		users, uerr := h.userRepo.ListAll(r.Context())
		if uerr != nil {
			slog.Warn("dashboard: list users failed", "sub", user.Sub, "err", uerr)
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

	// ── Admin: tag stats ──────────────────────────────────────────────────────
	var tagsBlock any
	if p.CanManageAllTags {
		tagsResp, terr := h.shlinkSvc.Client().GetTags(r.Context(), user.ShlinkAPIKey)
		if terr != nil {
			slog.Warn("dashboard: get tags failed", "sub", user.Sub, "err", terr)
		} else if tagsResp != nil {
			type tagStat struct {
				Tag    string `json:"tag"`
				Visits int    `json:"visits"`
				URLs   int    `json:"urls"`
			}
			stats := make([]tagStat, 0, len(tagsResp.Tags.Data))
			for _, t := range tagsResp.Tags.Data {
				stats = append(stats, tagStat{
					Tag:    t.Tag,
					Visits: t.VisitsSummary.Total,
					URLs:   t.ShortURLsCount,
				})
			}
			sort.Slice(stats, func(i, j int) bool { return stats[i].Visits > stats[j].Visits })
			if len(stats) > 20 {
				stats = stats[:20]
			}
			tagsBlock = stats
		}
	}

	writeJSON(w, map[string]any{
		"overview": overview,
		"users":    usersBlock,
		"tags":     tagsBlock,
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
	items := make([]topLinkItem, 0, len(urls))
	for _, u := range urls {
		visitsTotal += u.VisitsSummary.Total
		items = append(items, topLinkItem{
			ShortCode:   u.ShortCode,
			ShortURL:    u.ShortURL,
			LongURL:     u.LongURL,
			Title:       u.Title,
			VisitsTotal: u.VisitsSummary.Total,
		})
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].VisitsTotal > items[j].VisitsTotal
	})
	top := items
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
			ua := strings.ToLower(v.UserAgent)
			switch {
			case strings.Contains(ua, "mobi") || strings.Contains(ua, "android"):
				mobile++
			case strings.Contains(ua, "tablet") || strings.Contains(ua, "ipad"):
				tablet++
			default:
				desktop++
			}
			browsersMap[urlDetailBrowser(ua)]++
			osMap[urlDetailOS(ua)]++

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
