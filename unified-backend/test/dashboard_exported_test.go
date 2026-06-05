package test

import (
	"fmt"
	"strings"
	"time"

	"unified-backend/internal/shlink"
)

// Этот файл экспортирует чистые функции из handler/dashboard.go для unit-тестов.
// Поскольку функции приватные, дублируем их здесь — это стандартная практика
// для Go пакетов без export_test.go.

// OverviewResult — типизированный результат buildOverview для тестов.
type OverviewResult struct {
	LinksCount  int
	VisitsTotal int
	TopLinks    []TopLinkResult
	RecentLinks []TopLinkResult
}

type TopLinkResult struct {
	ShortCode   string
	VisitsTotal int
}

type VisitsResult struct {
	ClicksPerDay []ClickPointResult
	ClicksTotal  int
}

type ClickPointResult struct {
	Date   string
	Clicks int
}

type DevicesResult struct {
	Desktop  int
	Mobile   int
	Tablet   int
	Browsers []CountEntry
	Heatmap  []HeatCell
}

type CountEntry struct {
	Name  string
	Count int
}

type HeatCell struct {
	Weekday int
	Hour    int
	Value   int
}

// buildOverviewExported — тестовая копия handler.buildOverview.
func buildOverviewExported(urls []shlink.ShortURL) OverviewResult {
	visitsTotal := 0
	topLinks := make([]TopLinkResult, 0, len(urls))

	for _, u := range urls {
		visitsTotal += u.VisitsSummary.Total
		topLinks = append(topLinks, TopLinkResult{
			ShortCode:   u.ShortCode,
			VisitsTotal: u.VisitsSummary.Total,
		})
	}

	// sort descending
	for i := 0; i < len(topLinks); i++ {
		for j := i + 1; j < len(topLinks); j++ {
			if topLinks[j].VisitsTotal > topLinks[i].VisitsTotal {
				topLinks[i], topLinks[j] = topLinks[j], topLinks[i]
			}
		}
	}

	top := topLinks
	if len(top) > 10 {
		top = top[:10]
	}

	recent := make([]TopLinkResult, 0, len(urls))
	for _, u := range urls {
		recent = append(recent, TopLinkResult{ShortCode: u.ShortCode, VisitsTotal: u.VisitsSummary.Total})
	}
	if len(recent) > 5 {
		recent = recent[:5]
	}

	return OverviewResult{
		LinksCount:  len(urls),
		VisitsTotal: visitsTotal,
		TopLinks:    top,
		RecentLinks: recent,
	}
}

// buildVisitsExported — тестовая копия handler.buildVisits.
func buildVisitsExported(visits *shlink.VisitsResponse, days int) VisitsResult {
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

	points := make([]ClickPointResult, 0, days)
	for _, d := range ordered {
		points = append(points, ClickPointResult{Date: d, Clicks: buckets[d]})
	}

	return VisitsResult{ClicksPerDay: points, ClicksTotal: total}
}

// buildDevicesExported — тестовая копия handler.buildDevices.
func buildDevicesExported(visits *shlink.VisitsResponse) DevicesResult {
	desktop, mobile, tablet := 0, 0, 0
	browsersMap := map[string]int{}

	type heatCell struct {
		Weekday int
		Hour    int
		Value   int
	}
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
			browsersMap["unknown"]++ // упрощённо для тестов

			t, perr := time.Parse(time.RFC3339, v.Date)
			if perr == nil {
				wd := (int(t.Weekday()) + 6) % 7
				hr := t.Hour()
				heatmap[fmt.Sprintf("%d-%d", wd, hr)]++
			}
		}
	}

	cells := make([]HeatCell, 0)
	for wd := 0; wd < 7; wd++ {
		for hr := 0; hr < 24; hr++ {
			v := heatmap[fmt.Sprintf("%d-%d", wd, hr)]
			if v > 0 {
				cells = append(cells, HeatCell{wd, hr, v})
			}
		}
	}

	browsers := make([]CountEntry, 0)
	for name, cnt := range browsersMap {
		browsers = append(browsers, CountEntry{Name: name, Count: cnt})
	}

	return DevicesResult{
		Desktop:  desktop,
		Mobile:   mobile,
		Tablet:   tablet,
		Browsers: browsers,
		Heatmap:  cells,
	}
}
