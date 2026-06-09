package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"
	"unified-backend/internal/config"
	"unified-backend/internal/domain"
	"unified-backend/internal/shlink"
)

type StatsOwnershipRepo interface {
	GetBatch(ctx context.Context, shortCodes []string, domain string) (map[string]*domain.ShortURLMetadata, error)
	GetAllByOwner(ctx context.Context, ownerSub string) ([]*domain.ShortURLMetadata, error)
}

type ShlinkClientIface interface {
	GetShortURLs(ctx context.Context, apiKey, rawQuery string) (*shlink.ShortURLsResponse, error)
	GetShortURL(ctx context.Context, apiKey, shortCode string) (*shlink.ShortURL, error)
	GetShortURLVisits(ctx context.Context, apiKey, shortCode, startDate, endDate string, itemsPerPage int) (*shlink.VisitsResponse, error)
	CreateShortURL(ctx context.Context, apiKey string, body io.Reader) (*shlink.ShortURL, error)
	UpdateShortURL(ctx context.Context, apiKey, shortCode string, body io.Reader) (*shlink.ShortURL, error)
	DeleteShortURL(ctx context.Context, apiKey, shortCode string) error
	GetTags(ctx context.Context, apiKey string) (*shlink.TagsWithStatsResponse, error)
	CreateTag(ctx context.Context, apiKey string, body io.Reader) (*shlink.TagsWithStatsResponse, error)
	RenameTag(ctx context.Context, apiKey string, body io.Reader) error
	DeleteTags(ctx context.Context, apiKey string, tags []string) error
	GetNonOrphanVisits(ctx context.Context, apiKey, startDate, endDate string, itemsPerPage int) (*shlink.VisitsResponse, error)
	PatchSettings(ctx context.Context, adminAPIKey string, shortCodeLength int) error
	GetHealth(ctx context.Context) (*shlink.HealthResponse, error)
	ValidateVersion(ctx context.Context, minMajor int, attempts int, delay time.Duration) error
}

type ShlinkService struct {
	client ShlinkClientIface
	cfg    *config.Config
}

func NewShlinkService(client ShlinkClientIface, cfg *config.Config) *ShlinkService {
	return &ShlinkService{client: client, cfg: cfg}
}

func (s *ShlinkService) Client() ShlinkClientIface {
	return s.client
}

func (s *ShlinkService) resolveAPIKey(user *domain.User) string {
	if user != nil && user.ShlinkAPIKey != "" {
		return user.ShlinkAPIKey
	}
	return s.cfg.ShlinkAdminAPIKey
}

// GetDashboardStats – собирает полную статистику для дашборда пользователя.
// Если у пользователя есть доступ к чужим ссылкам (view_all), статистика считается по всем ссылкам.
// Иначе – только по его собственным.
func (s *ShlinkService) GetDashboardStats(ctx context.Context, user *domain.User, canViewAll bool, ownerRepo StatsOwnershipRepo) (*domain.DashboardStats, error) {
	apiKey := s.resolveAPIKey(user)

	// Получаем все короткие ссылки пользователя (или все, если canViewAll)
	// Shlink возвращает постранично, запросим достаточно большой лимит
	resp, err := s.client.GetShortURLs(ctx, apiKey, "itemsPerPage=1000")
	if err != nil {
		return nil, fmt.Errorf("failed to get short urls: %w", err)
	}
	allShortURLs := resp.ShortURLs.Data

	// Собираем метаданные (владелец, активность, теги) из БД
	var ownershipMap map[string]*domain.ShortURLMetadata
	if canViewAll {
		// Для админа/аудитора – все ссылки, метаданные грузим по кодам
		codes := make([]string, len(allShortURLs))
		for i, u := range allShortURLs {
			codes[i] = u.ShortCode
		}
		ownershipMap, _ = ownerRepo.GetBatch(ctx, codes, "")
	} else {
		// Для обычного пользователя – сразу грузим его метаданные
		ownMeta, err := ownerRepo.GetAllByOwner(ctx, user.Sub)
		if err == nil {
			ownershipMap = make(map[string]*domain.ShortURLMetadata, len(ownMeta))
			for _, m := range ownMeta {
				ownershipMap[m.ShortCode] = m
			}
		}
	}

	// Сопоставляем Shlink-данные с метаданными
	type enriched struct {
		short   shlink.ShortURL
		meta    *domain.ShortURLMetadata
		isOwner bool
	}
	var enrichedList []enriched
	for _, u := range allShortURLs {
		meta, _ := ownershipMap[u.ShortCode]
		isOwner := meta != nil && meta.OwnerSub == user.Sub
		// Если canViewAll – показываем все ссылки, иначе только те, где пользователь владелец
		if canViewAll || isOwner {
			enrichedList = append(enrichedList, enriched{
				short:   u,
				meta:    meta,
				isOwner: isOwner,
			})
		}
	}

	// Базовые счётчики
	totalLinks := len(enrichedList)
	activeLinks := 0
	for _, e := range enrichedList {
		active := true
		if e.meta != nil {
			active = e.meta.IsActive
		}
		if active {
			activeLinks++
		}
	}

	// Получаем визиты за последние 30 дней (один запрос на все не-orphan визиты)
	now := time.Now()
	start30 := now.AddDate(0, 0, -30).Format(time.RFC3339)
	end := now.Format(time.RFC3339)
	visitsResp, err := s.client.GetNonOrphanVisits(ctx, apiKey, start30, end, 5000)
	if err != nil {
		// Не фатально, просто статистика визитов будет неполной
		visitsResp = nil
	}
	var visits []shlink.Visit
	if visitsResp != nil {
		visits = visitsResp.Visits.Data
	}

	// Агрегируем клики по дням
	dayBuckets := make(map[string]int)
	last7Days := 0
	for _, v := range visits {
		t, err := time.Parse(time.RFC3339, v.Date)
		if err != nil {
			continue
		}
		day := t.Format("2006-01-02")
		dayBuckets[day]++
		if t.After(now.AddDate(0, 0, -7)) {
			last7Days++
		}
	}

	// Упорядочиваем дни за последние 30 дней
	clicksByDay := make([]domain.ClicksByDay, 0, 30)
	for i := 29; i >= 0; i-- {
		day := now.AddDate(0, 0, -i).Format("2006-01-02")
		clicks := dayBuckets[day]
		clicksByDay = append(clicksByDay, domain.ClicksByDay{Date: day, Clicks: clicks})
	}
	totalClicks := 0
	for _, c := range clicksByDay {
		totalClicks += c.Clicks
	}
	periodClicks := totalClicks // за 30 дней

	// Топ-5 ссылок по кликам
	type linkStat struct {
		short  shlink.ShortURL
		clicks int
		meta   *domain.ShortURLMetadata
	}
	var stats []linkStat
	for _, e := range enrichedList {
		stats = append(stats, linkStat{
			short:  e.short,
			clicks: e.short.VisitsSummary.Total,
			meta:   e.meta,
		})
	}
	sort.Slice(stats, func(i, j int) bool { return stats[i].clicks > stats[j].clicks })
	topLinks := make([]domain.TopLink, 0, 5)
	for i := 0; i < len(stats) && i < 5; i++ {
		title := ""
		if stats[i].meta != nil {
			title = stats[i].meta.Title
		}
		topLinks = append(topLinks, domain.TopLink{
			ShortCode: stats[i].short.ShortCode,
			LongURL:   stats[i].short.LongURL,
			ShortURL:  stats[i].short.ShortURL,
			Clicks:    stats[i].clicks,
			Title:     title,
		})
	}

	// Топ-10 тегов (из метаданных пользователя)
	tagCount := make(map[string]int)
	for _, e := range enrichedList {
		if e.meta != nil && len(e.meta.Tags) > 0 {
			for _, tag := range e.meta.Tags {
				tagCount[tag]++
			}
		}
	}
	topTags := make([]domain.TopTag, 0, len(tagCount))
	for tag, cnt := range tagCount {
		topTags = append(topTags, domain.TopTag{Tag: tag, Count: cnt})
	}
	sort.Slice(topTags, func(i, j int) bool { return topTags[i].Count > topTags[j].Count })
	if len(topTags) > 10 {
		topTags = topTags[:10]
	}

	return &domain.DashboardStats{
		TotalLinks:      totalLinks,
		ActiveLinks:     activeLinks,
		TotalClicks:     totalClicks,
		PeriodClicks:    periodClicks,
		Last7DaysClicks: last7Days,
		ClicksByDay:     clicksByDay,
		TopLinks:        topLinks,
		TopTags:         topTags,
	}, nil
}

func (s *ShlinkService) GetShortURLs(ctx context.Context, user *domain.User, rawQuery string) (*shlink.ShortURLsResponse, error) {
	return s.client.GetShortURLs(ctx, s.resolveAPIKey(user), rawQuery)
}

func (s *ShlinkService) GetShortURL(ctx context.Context, user *domain.User, shortCode string) (*shlink.ShortURL, error) {
	return s.client.GetShortURL(ctx, s.resolveAPIKey(user), shortCode)
}

func (s *ShlinkService) GetShortURLVisits(ctx context.Context, user *domain.User, shortCode, startDate, endDate string, itemsPerPage int) (*shlink.VisitsResponse, error) {
	return s.client.GetShortURLVisits(ctx, s.resolveAPIKey(user), shortCode, startDate, endDate, itemsPerPage)
}

func (s *ShlinkService) CreateShortURL(ctx context.Context, user *domain.User, body io.Reader) (*shlink.ShortURL, error) {
	return s.client.CreateShortURL(ctx, s.resolveAPIKey(user), body)
}

func (s *ShlinkService) UpdateShortURL(ctx context.Context, user *domain.User, shortCode string, body io.Reader) (*shlink.ShortURL, error) {
	return s.client.UpdateShortURL(ctx, s.resolveAPIKey(user), shortCode, body)
}

func (s *ShlinkService) DeleteShortURL(ctx context.Context, user *domain.User, shortCode string) error {
	return s.client.DeleteShortURL(ctx, s.resolveAPIKey(user), shortCode)
}

func (s *ShlinkService) GetTags(ctx context.Context, user *domain.User) (*shlink.TagsWithStatsResponse, error) {
	return s.client.GetTags(ctx, s.resolveAPIKey(user))
}

func (s *ShlinkService) RenameTag(ctx context.Context, user *domain.User, body io.Reader) error {
	return s.client.RenameTag(ctx, s.resolveAPIKey(user), body)
}

func (s *ShlinkService) DeleteTags(ctx context.Context, user *domain.User, tags []string) error {
	return s.client.DeleteTags(ctx, s.resolveAPIKey(user), tags)
}

func (s *ShlinkService) GetNonOrphanVisits(ctx context.Context, user *domain.User, startDate, endDate string, itemsPerPage int) (*shlink.VisitsResponse, error) {
	return s.client.GetNonOrphanVisits(ctx, s.resolveAPIKey(user), startDate, endDate, itemsPerPage)
}

func (s *ShlinkService) PatchSettings(ctx context.Context, shortCodeLength int) error {
	return s.client.PatchSettings(ctx, s.cfg.ShlinkAdminAPIKey, shortCodeLength)
}

func (s *ShlinkService) GetHealth(ctx context.Context) (*shlink.HealthResponse, error) {
	return s.client.GetHealth(ctx)
}

func (s *ShlinkService) FilterShortURLsByUser(urls []shlink.ShortURL, _ *domain.User, ownedCodes map[string]struct{}) []shlink.ShortURL {
	result := make([]shlink.ShortURL, 0)
	for _, u := range urls {
		if _, ok := ownedCodes[u.ShortCode]; ok {
			result = append(result, u)
		}
	}
	return result
}

func (s *ShlinkService) EnforceSlugPrefix(_ context.Context, user *domain.User, customSlug *string) (string, error) {
	if customSlug == nil || *customSlug == "" {
		return "", nil
	}
	if user == nil || user.SlugPrefix == "" {
		return *customSlug, nil
	}
	prefix := user.SlugPrefix + "-"
	if strings.HasPrefix(*customSlug, prefix) {
		return *customSlug, nil
	}
	return prefix + *customSlug, nil
}

func (s *ShlinkService) BuildSlug(user *domain.User, slug string) string {
	if user == nil || user.SlugPrefix == "" {
		return slug
	}
	if slug == "" {
		return user.SlugPrefix
	}
	return strings.Join([]string{user.SlugPrefix, slug}, "-")
}

func (s *ShlinkService) EnforceDomain(_ context.Context, user *domain.User, requestedDomain string) error {
	if user == nil || user.AllowedDomains == "" {
		return nil
	}
	if requestedDomain == "" {
		return nil
	}
	var allowed []string
	if err := json.Unmarshal([]byte(user.AllowedDomains), &allowed); err != nil {
		return nil
	}
	if len(allowed) == 0 {
		return nil
	}
	for _, d := range allowed {
		if strings.EqualFold(d, requestedDomain) {
			return nil
		}
	}
	return fmt.Errorf("domain %q is not allowed for this user", requestedDomain)
}

