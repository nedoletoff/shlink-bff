package shlink

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	neturl "net/url"
	"strconv"
	"strings"
	"time"
)

// Client — HTTP-клиент к внутреннему shlink-api.
type Client struct {
	baseURL    string
	httpClient *http.Client
}

func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

// --- Типы ответов Shlink API (v5.x) ---

type VisitsSummary struct {
	Total int `json:"total"`
}

type ShortURL struct {
	ShortCode     string        `json:"shortCode"`
	ShortURL      string        `json:"shortUrl"`
	LongURL       string        `json:"longUrl"`
	Title         string        `json:"title"`
	Tags          []string      `json:"tags"`
	VisitsSummary VisitsSummary `json:"visitsSummary"`
	DateCreated   string        `json:"dateCreated"`
	// Дополнительные поля, возвращаемые Shlink v5:
	ValidSince *string `json:"validSince,omitempty"` // RFC3339
	ValidUntil *string `json:"validUntil,omitempty"` // RFC3339
	MaxVisits  *int    `json:"maxVisits,omitempty"`
	Enabled    bool    `json:"enabled"` // активна ли ссылка
}

type Pagination struct {
	CurrentPage        int `json:"currentPage"`
	PagesCount         int `json:"pagesCount"`
	ItemsPerPage       int `json:"itemsPerPage"`
	ItemsInCurrentPage int `json:"itemsInCurrentPage"`
	TotalItems         int `json:"totalItems"`
}

type ShortURLsResponse struct {
	ShortURLs struct {
		Data       []ShortURL `json:"data"`
		Pagination Pagination `json:"pagination"`
	} `json:"shortUrls"`
}

type TagsResponse struct {
	Tags struct {
		Data []string `json:"data"`
	} `json:"tags"`
}

type TagsWithStatsResponse struct {
	Tags struct {
		Data []TagStats `json:"data"`
	} `json:"tags"`
}

type TagStats struct {
	Tag            string `json:"tag"`
	ShortURLsCount int    `json:"shortUrlsCount"`
	VisitsSummary  struct {
		Total int `json:"total"`
	} `json:"visitsSummary"`
}

type VisitsResponse struct {
	Visits struct {
		Data       []Visit    `json:"data"`
		Pagination Pagination `json:"pagination"`
	} `json:"visits"`
}

type Visit struct {
	Referer       string        `json:"referer"`
	Date          string        `json:"date"`
	UserAgent     string        `json:"userAgent"`
	VisitLocation VisitLocation `json:"visitLocation"`
}

type VisitLocation struct {
	CountryName string `json:"countryName"`
	CityName    string `json:"cityName"`
}

// CreateShortURLRequest используется при необходимости дополнить тело запроса.
type CreateShortURLRequest struct {
	ShortCodeLength int `json:"shortCodeLength,omitempty"`
}

// ShlinkSettings — тело PATCH /rest/v3/settings.
type ShlinkSettings struct {
	ShortURLCreation *ShlinkShortURLCreationSettings `json:"shortUrlCreation,omitempty"`
}

type ShlinkShortURLCreationSettings struct {
	DefaultShortCodesLength int `json:"defaultShortCodesLength"`
}

// --- Методы клиента ---

func (c *Client) GetShortURLs(ctx context.Context, apiKey, rawQuery string) (*ShortURLsResponse, error) {
	url := c.baseURL + "/rest/v3/short-urls"
	if rawQuery != "" {
		url += "?" + rawQuery
	}
	return doRequest[ShortURLsResponse](ctx, c, http.MethodGet, url, apiKey, nil)
}

func (c *Client) GetShortURL(ctx context.Context, apiKey, shortCode string) (*ShortURL, error) {
	url := fmt.Sprintf("%s/rest/v3/short-urls/%s", c.baseURL, shortCode)
	return doRequest[ShortURL](ctx, c, http.MethodGet, url, apiKey, nil)
}

func (c *Client) GetShortURLVisits(ctx context.Context, apiKey, shortCode, startDate, endDate string, itemsPerPage int) (*VisitsResponse, error) {
	if itemsPerPage <= 0 {
		itemsPerPage = 1000
	}
	q := neturl.Values{}
	if startDate != "" {
		q.Set("startDate", startDate)
	}
	if endDate != "" {
		q.Set("endDate", endDate)
	}
	q.Set("itemsPerPage", strconv.Itoa(itemsPerPage))
	url := fmt.Sprintf("%s/rest/v3/short-urls/%s/visits?%s", c.baseURL, shortCode, q.Encode())
	return doRequest[VisitsResponse](ctx, c, http.MethodGet, url, apiKey, nil)
}

func (c *Client) CreateShortURL(ctx context.Context, apiKey string, body io.Reader) (*ShortURL, error) {
	return doRequest[ShortURL](ctx, c, http.MethodPost, c.baseURL+"/rest/v3/short-urls", apiKey, body)
}

// CreateShortURLWithConfig — вставляет shortCodeLength в тело запроса перед отправкой.
func (c *Client) CreateShortURLWithConfig(ctx context.Context, apiKey string, body io.Reader, shortCodeLength int) (*ShortURL, error) {
	if shortCodeLength <= 0 {
		return c.CreateShortURL(ctx, apiKey, body)
	}
	var payload map[string]json.RawMessage
	if err := json.NewDecoder(body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("shlink_client: failed to decode create request body: %w", err)
	}
	lengthBytes, err := json.Marshal(shortCodeLength)
	if err != nil {
		return nil, err
	}
	payload["shortCodeLength"] = json.RawMessage(lengthBytes)
	modified, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	return doRequest[ShortURL](ctx, c, http.MethodPost, c.baseURL+"/rest/v3/short-urls", apiKey, bytes.NewReader(modified))
}

func (c *Client) UpdateShortURL(ctx context.Context, apiKey, shortCode string, body io.Reader) (*ShortURL, error) {
	url := fmt.Sprintf("%s/rest/v3/short-urls/%s", c.baseURL, shortCode)
	return doRequest[ShortURL](ctx, c, http.MethodPatch, url, apiKey, body)
}

func (c *Client) DeleteShortURL(ctx context.Context, apiKey, shortCode string) error {
	url := fmt.Sprintf("%s/rest/v3/short-urls/%s", c.baseURL, shortCode)
	_, err := doRequest[struct{}](ctx, c, http.MethodDelete, url, apiKey, nil)
	return err
}

func (c *Client) GetTags(ctx context.Context, apiKey string) (*TagsWithStatsResponse, error) {
	url := c.baseURL + "/rest/v3/tags?withStats=true"
	return doRequest[TagsWithStatsResponse](ctx, c, http.MethodGet, url, apiKey, nil)
}

func (c *Client) CreateTag(ctx context.Context, apiKey string, body io.Reader) (*TagsWithStatsResponse, error) {
	url := c.baseURL + "/rest/v3/tags"
	return doRequest[TagsWithStatsResponse](ctx, c, http.MethodPost, url, apiKey, body)
}

func (c *Client) RenameTag(ctx context.Context, apiKey string, body io.Reader) error {
	url := c.baseURL + "/rest/v3/tags"
	_, err := doRequest[struct{}](ctx, c, http.MethodPut, url, apiKey, body)
	return err
}

func (c *Client) DeleteTags(ctx context.Context, apiKey string, tags []string) error {
	url := c.baseURL + "/rest/v3/tags?tags[]=" + strings.Join(tags, "&tags[]=")
	_, err := doRequest[struct{}](ctx, c, http.MethodDelete, url, apiKey, nil)
	return err
}

func (c *Client) GetVisitsSummary(ctx context.Context, apiKey string) (map[string]any, error) {
	url := c.baseURL + "/rest/v3/visits"
	res, err := doRequest[map[string]any](ctx, c, http.MethodGet, url, apiKey, nil)
	if err != nil {
		return nil, err
	}
	return *res, nil
}

func (c *Client) GetNonOrphanVisits(ctx context.Context, apiKey, startDate, endDate string, itemsPerPage int) (*VisitsResponse, error) {
	if itemsPerPage <= 0 {
		itemsPerPage = 1000
	}
	q := neturl.Values{}
	if startDate != "" {
		q.Set("startDate", startDate)
	}
	if endDate != "" {
		q.Set("endDate", endDate)
	}
	q.Set("itemsPerPage", strconv.Itoa(itemsPerPage))
	url := c.baseURL + "/rest/v3/visits/non-orphan?" + q.Encode()
	return doRequest[VisitsResponse](ctx, c, http.MethodGet, url, apiKey, nil)
}

func (c *Client) PatchSettings(ctx context.Context, adminAPIKey string, shortCodeLength int) error {
	if adminAPIKey == "" {
		return fmt.Errorf("shlink: admin API key not configured")
	}
	payload := ShlinkSettings{
		ShortURLCreation: &ShlinkShortURLCreationSettings{
			DefaultShortCodesLength: shortCodeLength,
		},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	url := c.baseURL + "/rest/v3/settings"
	_, err = doRequest[struct{}](ctx, c, http.MethodPatch, url, adminAPIKey, bytes.NewReader(body))
	return err
}

type HealthResponse struct {
	Status  string `json:"status"`
	Version string `json:"version"`
}

func (c *Client) GetHealth(ctx context.Context) (*HealthResponse, error) {
	url := c.baseURL + "/rest/health"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("shlink health returned %d", resp.StatusCode)
	}
	var result HealthResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) ValidateVersion(ctx context.Context, minMajor int, attempts int, delay time.Duration) error {
	if attempts <= 0 {
		attempts = 1
	}
	var lastErr error
	for i := 0; i < attempts; i++ {
		health, err := c.GetHealth(ctx)
		if err != nil {
			lastErr = err
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
			}
			continue
		}
		major := majorVersion(health.Version)
		if major < minMajor {
			return fmt.Errorf("incompatible Shlink version %q: requires major >= %d", health.Version, minMajor)
		}
		slog.Info("shlink_client: version validated", "version", health.Version, "status", health.Status)
		return nil
	}
	return fmt.Errorf("shlink health unavailable after %d attempts: %w", attempts, lastErr)
}

func majorVersion(v string) int {
	parts := strings.SplitN(strings.TrimSpace(v), ".", 2)
	if len(parts) == 0 {
		return 0
	}
	n, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0
	}
	return n
}

func doRequest[T any](
	ctx context.Context,
	c *Client,
	method, url, apiKey string,
	body io.Reader,
) (*T, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("X-Api-Key", apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		slog.Error("shlink_client: request failed", "method", method, "path", extractPath(url), "err", err)
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNoContent {
		return new(T), nil
	}
	if resp.StatusCode >= 400 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("shlink returned %d: %s", resp.StatusCode, string(bodyBytes))
	}
	var result T
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return &result, nil
}

func extractPath(url string) string {
	if idx := strings.Index(url, "/rest/"); idx >= 0 {
		return url[idx:]
	}
	return url
}

