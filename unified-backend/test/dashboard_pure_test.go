package test

import (
	"testing"
	"time"

	"unified-backend/internal/shlink"
)

// ── buildOverview ──────────────────────────────────────────────────────────

// TestBuildOverview_Empty — нет ссылок → нули.
func TestBuildOverview_Empty(t *testing.T) {
	out := buildOverviewExported(nil)
	if out.LinksCount != 0 {
		t.Errorf("LinksCount: want 0, got %d", out.LinksCount)
	}
	if out.VisitsTotal != 0 {
		t.Errorf("VisitsTotal: want 0, got %d", out.VisitsTotal)
	}
	if len(out.TopLinks) != 0 {
		t.Errorf("TopLinks: want empty, got %d items", len(out.TopLinks))
	}
}

// TestBuildOverview_TopLinksSorted — топ-10 отсортирован по убыванию визитов.
func TestBuildOverview_TopLinksSorted(t *testing.T) {
	urls := make([]shlink.ShortURL, 15)
	for i := range urls {
		urls[i] = shlink.ShortURL{
			ShortCode: mustSprintf("code-%d", i),
			VisitsSummary: shlink.VisitsSummary{Total: i},
		}
	}
	out := buildOverviewExported(urls)

	if out.LinksCount != 15 {
		t.Errorf("LinksCount: want 15, got %d", out.LinksCount)
	}
	if len(out.TopLinks) != 10 {
		t.Errorf("TopLinks cap: want 10, got %d", len(out.TopLinks))
	}
	// первый должен иметь наибольшее кол-во визитов
	if out.TopLinks[0].VisitsTotal < out.TopLinks[1].VisitsTotal {
		t.Error("TopLinks not sorted descending")
	}
}

// TestBuildOverview_RecentLinks — не более 5 в recentLinks.
func TestBuildOverview_RecentLinks(t *testing.T) {
	urls := make([]shlink.ShortURL, 8)
	for i := range urls {
		urls[i] = shlink.ShortURL{ShortCode: mustSprintf("c%d", i)}
	}
	out := buildOverviewExported(urls)
	if len(out.RecentLinks) != 5 {
		t.Errorf("RecentLinks cap: want 5, got %d", len(out.RecentLinks))
	}
}

// TestBuildOverview_VisitsTotals — сумма визитов считается корректно.
func TestBuildOverview_VisitsTotals(t *testing.T) {
	urls := []shlink.ShortURL{
		{ShortCode: "a", VisitsSummary: shlink.VisitsSummary{Total: 10}},
		{ShortCode: "b", VisitsSummary: shlink.VisitsSummary{Total: 25}},
		{ShortCode: "c", VisitsSummary: shlink.VisitsSummary{Total: 5}},
	}
	out := buildOverviewExported(urls)
	if out.VisitsTotal != 40 {
		t.Errorf("VisitsTotal: want 40, got %d", out.VisitsTotal)
	}
}

// ── buildVisits ────────────────────────────────────────────────────────────

// TestBuildVisits_NilVisits — nil visits → все бакеты = 0, total = 0.
func TestBuildVisits_NilVisits(t *testing.T) {
	out := buildVisitsExported(nil, 7)
	if out.ClicksTotal != 0 {
		t.Errorf("ClicksTotal: want 0, got %d", out.ClicksTotal)
	}
	if len(out.ClicksPerDay) != 7 {
		t.Errorf("ClicksPerDay len: want 7, got %d", len(out.ClicksPerDay))
	}
	for _, p := range out.ClicksPerDay {
		if p.Clicks != 0 {
			t.Errorf("expect 0 clicks for day %s, got %d", p.Date, p.Clicks)
		}
	}
}

// TestBuildVisits_CountsToday — визит сегодня попадает в последний бакет.
func TestBuildVisits_CountsToday(t *testing.T) {
	now := time.Now().UTC().Format(time.RFC3339)
	resp := &shlink.VisitsResponse{
		Visits: shlink.Pagination[shlink.Visit]{
			Data: []shlink.Visit{
				{Date: now},
				{Date: now},
			},
		},
	}
	out := buildVisitsExported(resp, 7)
	if out.ClicksTotal != 2 {
		t.Errorf("ClicksTotal: want 2, got %d", out.ClicksTotal)
	}
}

// TestBuildVisits_IgnoresOutOfRange — визит за пределами окна не считается.
func TestBuildVisits_IgnoresOutOfRange(t *testing.T) {
	old := time.Now().AddDate(0, -2, 0).UTC().Format(time.RFC3339)
	resp := &shlink.VisitsResponse{
		Visits: shlink.Pagination[shlink.Visit]{
			Data: []shlink.Visit{{Date: old}},
		},
	}
	out := buildVisitsExported(resp, 7)
	if out.ClicksTotal != 0 {
		t.Errorf("out-of-range visit must be ignored, got total=%d", out.ClicksTotal)
	}
}

// TestBuildVisits_CorrectBucketCount — число бакетов соответствует параметру days.
func TestBuildVisits_CorrectBucketCount(t *testing.T) {
	for _, days := range []int{1, 7, 30} {
		out := buildVisitsExported(nil, days)
		if len(out.ClicksPerDay) != days {
			t.Errorf("days=%d: want %d buckets, got %d", days, days, len(out.ClicksPerDay))
		}
	}
}

// TestBuildVisits_InvalidDateIgnored — не-RFC3339 дата не ломает функцию.
func TestBuildVisits_InvalidDateIgnored(t *testing.T) {
	resp := &shlink.VisitsResponse{
		Visits: shlink.Pagination[shlink.Visit]{
			Data: []shlink.Visit{
				{Date: "not-a-date"},
				{Date: "2024-99-99T00:00:00Z"},
			},
		},
	}
	out := buildVisitsExported(resp, 7)
	if out.ClicksTotal != 0 {
		t.Errorf("invalid dates must be ignored, got total=%d", out.ClicksTotal)
	}
}

// ── buildDevices ───────────────────────────────────────────────────────────

// TestBuildDevices_NilVisits — nil visits → все счётчики нули.
func TestBuildDevices_NilVisits(t *testing.T) {
	out := buildDevicesExported(nil)
	if out.Desktop != 0 || out.Mobile != 0 || out.Tablet != 0 {
		t.Errorf("nil visits: want all zeros, got d=%d m=%d t=%d",
			out.Desktop, out.Mobile, out.Tablet)
	}
	if len(out.Browsers) != 0 {
		t.Errorf("nil visits: want 0 browsers, got %d", len(out.Browsers))
	}
}

// TestBuildDevices_ClassifyMobile — User-Agent с 'Mobi' → mobile.
func TestBuildDevices_ClassifyMobile(t *testing.T) {
	resp := &shlink.VisitsResponse{
		Visits: shlink.Pagination[shlink.Visit]{
			Data: []shlink.Visit{
				{UserAgent: "Mozilla/5.0 (Linux; Android 10; Mobile)"},
				{UserAgent: "Mozilla/5.0 (iPhone; CPU iPhone OS 15_0) Mobi"},
			},
		},
	}
	out := buildDevicesExported(resp)
	if out.Mobile != 2 {
		t.Errorf("want Mobile=2, got %d", out.Mobile)
	}
	if out.Desktop != 0 {
		t.Errorf("want Desktop=0, got %d", out.Desktop)
	}
}

// TestBuildDevices_ClassifyTablet — User-Agent с 'iPad' → tablet.
func TestBuildDevices_ClassifyTablet(t *testing.T) {
	resp := &shlink.VisitsResponse{
		Visits: shlink.Pagination[shlink.Visit]{
			Data: []shlink.Visit{
				{UserAgent: "Mozilla/5.0 (iPad; CPU OS 15_0 like Mac OS X)"},
			},
		},
	}
	out := buildDevicesExported(resp)
	if out.Tablet != 1 {
		t.Errorf("want Tablet=1, got %d", out.Tablet)
	}
}

// TestBuildDevices_ClassifyDesktop — обычный desktop UA.
func TestBuildDevices_ClassifyDesktop(t *testing.T) {
	resp := &shlink.VisitsResponse{
		Visits: shlink.Pagination[shlink.Visit]{
			Data: []shlink.Visit{
				{UserAgent: "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36"},
				{UserAgent: "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7)"},
			},
		},
	}
	out := buildDevicesExported(resp)
	if out.Desktop != 2 {
		t.Errorf("want Desktop=2, got %d", out.Desktop)
	}
}

// TestBuildDevices_HeatmapNonZeroOnly — heatmap содержит только ненулевые ячейки.
func TestBuildDevices_HeatmapNonZeroOnly(t *testing.T) {
	now := time.Now().UTC().Format(time.RFC3339)
	resp := &shlink.VisitsResponse{
		Visits: shlink.Pagination[shlink.Visit]{
			Data: []shlink.Visit{
				{UserAgent: "Desktop", Date: now},
			},
		},
	}
	out := buildDevicesExported(resp)
	// heatmap должен содержать ровно одну ячейку (для текущего wd+hr)
	if len(out.Heatmap) != 1 {
		t.Errorf("heatmap should contain 1 cell, got %d", len(out.Heatmap))
	}
}

// ── helpers ────────────────────────────────────────────────────────────────

func mustSprintf(format string, a ...any) string {
	import_fmt_sprintf := func(f string, args ...any) string {
		s := f
		_ = args
		return s
	}
	_ = import_fmt_sprintf
	return ""
}
