package test

import (
	"fmt"
	"testing"
	"time"

	"unified-backend/internal/shlink"
)

// ── buildOverview ──────────────────────────────────────────────────────────

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

func TestBuildOverview_TopLinksSorted(t *testing.T) {
	urls := make([]shlink.ShortURL, 15)
	for i := range urls {
		urls[i] = shlink.ShortURL{
			ShortCode:     fmt.Sprintf("code-%d", i),
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
	if out.TopLinks[0].VisitsTotal < out.TopLinks[1].VisitsTotal {
		t.Error("TopLinks not sorted descending")
	}
}

func TestBuildOverview_RecentLinks(t *testing.T) {
	urls := make([]shlink.ShortURL, 8)
	for i := range urls {
		urls[i] = shlink.ShortURL{ShortCode: fmt.Sprintf("c%d", i)}
	}
	out := buildOverviewExported(urls)
	if len(out.RecentLinks) != 5 {
		t.Errorf("RecentLinks cap: want 5, got %d", len(out.RecentLinks))
	}
}

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

func TestBuildVisits_CountsToday(t *testing.T) {
	now := time.Now().UTC().Format(time.RFC3339)
	resp := &shlink.VisitsResponse{
		Visits: struct {
			Data       []shlink.Visit    `json:"data"`
			Pagination shlink.Pagination `json:"pagination"`
		}{
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

func TestBuildVisits_IgnoresOutOfRange(t *testing.T) {
	old := time.Now().AddDate(0, -2, 0).UTC().Format(time.RFC3339)
	resp := &shlink.VisitsResponse{
		Visits: struct {
			Data       []shlink.Visit    `json:"data"`
			Pagination shlink.Pagination `json:"pagination"`
		}{
			Data: []shlink.Visit{{Date: old}},
		},
	}
	out := buildVisitsExported(resp, 7)
	if out.ClicksTotal != 0 {
		t.Errorf("out-of-range visit must be ignored, got total=%d", out.ClicksTotal)
	}
}

func TestBuildVisits_CorrectBucketCount(t *testing.T) {
	for _, days := range []int{1, 7, 30} {
		out := buildVisitsExported(nil, days)
		if len(out.ClicksPerDay) != days {
			t.Errorf("days=%d: want %d buckets, got %d", days, days, len(out.ClicksPerDay))
		}
	}
}

func TestBuildVisits_InvalidDateIgnored(t *testing.T) {
	resp := &shlink.VisitsResponse{
		Visits: struct {
			Data       []shlink.Visit    `json:"data"`
			Pagination shlink.Pagination `json:"pagination"`
		}{
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

func TestBuildDevices_ClassifyMobile(t *testing.T) {
	resp := &shlink.VisitsResponse{
		Visits: struct {
			Data       []shlink.Visit    `json:"data"`
			Pagination shlink.Pagination `json:"pagination"`
		}{
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

func TestBuildDevices_ClassifyTablet(t *testing.T) {
	resp := &shlink.VisitsResponse{
		Visits: struct {
			Data       []shlink.Visit    `json:"data"`
			Pagination shlink.Pagination `json:"pagination"`
		}{
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

func TestBuildDevices_ClassifyDesktop(t *testing.T) {
	resp := &shlink.VisitsResponse{
		Visits: struct {
			Data       []shlink.Visit    `json:"data"`
			Pagination shlink.Pagination `json:"pagination"`
		}{
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

func TestBuildDevices_HeatmapNonZeroOnly(t *testing.T) {
	now := time.Now().UTC().Format(time.RFC3339)
	resp := &shlink.VisitsResponse{
		Visits: struct {
			Data       []shlink.Visit    `json:"data"`
			Pagination shlink.Pagination `json:"pagination"`
		}{
			Data: []shlink.Visit{
				{UserAgent: "Desktop", Date: now},
			},
		},
	}
	out := buildDevicesExported(resp)
	if len(out.Heatmap) != 1 {
		t.Errorf("heatmap should contain 1 cell, got %d", len(out.Heatmap))
	}
}
