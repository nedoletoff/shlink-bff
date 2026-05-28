package shlink

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

// Client — HTTP-клиент к внутреннему shlink-api.
// X-Api-Key подставляется здесь, на стороне backend.
// Ключ никогда не логируется и не отдаётся в ответах UI.
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

// --- Типы ответов Shlink API ---

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
}

type Pagination struct {
	CurrentPage            int `json:"currentPage"`
	PagesCount             int `json:"pagesCount"`
	ItemsPerPage           int `json:"itemsPerPage"`
	ItemsInCurrentPage     int `json:"itemsInCurrentPage"`
	TotalItems             int `json:"totalItems"`
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
	Tag         string `json:"tag"`
	ShortURLsCount int `json:"shortUrlsCount"`
	VisitsCount    int `json:"visitsCount"`
}

type VisitsResponse struct {
	Visits struct {
		Data       []Visit    `json:"data"`
		Pagination Pagination `json:"pagination"`
	} `json:"visits"`
}

type Visit struct {
	Referer   string    `json:"referer"`
	Date      string    `json:"date"`
	UserAgent string    `json:"userAgent"`
	VisitLocation VisitLocation `json:"visitLocation"`
}

type VisitLocation struct {
	CountryName string `json:"countryName"`
	CityName    string `json:"cityName"`
}

// --- Методы клиента ---

func (c *Client) GetShortURLs(ctx context.Context, apiKey, rawQuery string) (*ShortURLsResponse, error) {
	url := c.baseURL + "/rest/v3/short-urls"
	if rawQuery != "" {
		url += "?" + rawQuery
	}
	return doRequest[ShortURLsResponse](ctx, c, http.MethodGet, url, apiKey, nil)
}

func (c *Client) CreateShortURL(ctx context.Context, apiKey string, body io.Reader) (*ShortURL, error) {
	return doRequest[ShortURL](ctx, c, http.MethodPost, c.baseURL+"/rest/v3/short-urls", apiKey, body)
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
	return doRequest[map[string]any](ctx, c, http.MethodGet, url, apiKey, nil)
}

func (c *Client) GetHealth(ctx context.Context) (map[string]any, error) {
	url := c.baseURL + "/rest/v2/health"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var result map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&result)
	return result, nil
}

// doRequest — обобщённый исполнитель HTTP-запросов к Shlink.
// API-ключ подставляется в заголовок X-Api-Key здесь, на сервере.
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
	req.Header.Set("X-Api-Key", apiKey) // ключ подставляется только здесь

	resp, err := c.httpClient.Do(req)
	if err != nil {
		slog.Error("shlink_client: request failed",
			"method", method, "path", extractPath(url), "err", err)
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
