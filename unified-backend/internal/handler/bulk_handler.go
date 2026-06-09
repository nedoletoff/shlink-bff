package handler

import (
	"encoding/json"
	"fmt"
	"net/http"

	"unified-backend/internal/config"
	"unified-backend/internal/controller"
	"unified-backend/internal/domain"
	"unified-backend/internal/middleware"
	"unified-backend/internal/service"
)

// BulkHandler — массовые операции над короткими ссылками.
type BulkHandler struct {
	shlinkSvc *service.ShlinkService
	ownerRepo OwnershipRepo
	permCtrl  controller.PermChecker
	cfg       *config.Config
}

func NewBulkHandler(
	shlinkSvc *service.ShlinkService,
	ownerRepo OwnershipRepo,
	permCtrl controller.PermChecker,
	cfg *config.Config,
) *BulkHandler {
	return &BulkHandler{
		shlinkSvc: shlinkSvc,
		ownerRepo: ownerRepo,
		permCtrl:  permCtrl,
		cfg:       cfg,
	}
}

// bulkCreateItem — один элемент запроса на создание.
type bulkCreateItem struct {
	LongURL    string   `json:"longUrl"`
	CustomSlug string   `json:"customSlug,omitempty"`
	Tags       []string `json:"tags,omitempty"`
}

// bulkCreateResult — результат обработки одного элемента.
type bulkCreateResult struct {
	Index     int    `json:"index"`
	ShortCode string `json:"shortCode,omitempty"`
	ShortURL  string `json:"shortUrl,omitempty"`
	Error     string `json:"error,omitempty"`
	Status    string `json:"status"` // "ok" | "error" | "forbidden"
}

// BulkCreate — POST /api/shlink/short-urls/bulk
// Тело: [{"longUrl": "...", "customSlug": "...", "tags": [...]}]
// Проверяет PermShortURLsCreate или PermShortURLsCreateOwn для каждого элемента.
func (h *BulkHandler) BulkCreate(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromCtx(r.Context())
	if user == nil {
		writeJSON(w, map[string]string{"error": "unauthorized"}, http.StatusUnauthorized)
		return
	}

	var items []bulkCreateItem
	if err := json.NewDecoder(r.Body).Decode(&items); err != nil {
		writeJSON(w, map[string]string{"error": "invalid request body"}, http.StatusBadRequest)
		return
	}

	limit := h.cfg.BulkOperationLimit
	if limit <= 0 {
		limit = 100
	}
	if len(items) > limit {
		writeJSON(w, map[string]string{
			"error": fmt.Sprintf("bulk limit exceeded: max %d items", limit),
		}, http.StatusBadRequest)
		return
	}

	// Проверяем права заранее (один раз)
	canGlobal, _ := h.permCtrl.Check(r.Context(), user.ID, domain.PermShortURLsCreate)
	canOwn, _    := h.permCtrl.Check(r.Context(), user.ID, domain.PermShortURLsCreateOwn)

	if !canGlobal && !canOwn {
		writeJSON(w, map[string]string{"error": "forbidden"}, http.StatusForbidden)
		return
	}

	results := make([]bulkCreateResult, 0, len(items))
	for i, item := range items {
		if item.LongURL == "" {
			results = append(results, bulkCreateResult{
				Index:  i,
				Error:  "longUrl is required",
				Status: "error",
			})
			continue
		}

		created, err := h.shlinkSvc.CreateShortURL(r.Context(), service.CreateShortURLRequest{
			LongURL:    item.LongURL,
			CustomSlug: item.CustomSlug,
			Tags:       item.Tags,
			OwnerSub:   user.Sub,
		})
		if err != nil {
			results = append(results, bulkCreateResult{
				Index:  i,
				Error:  err.Error(),
				Status: "error",
			})
			continue
		}

		// Сохраняем владельца асинхронно, не блокируя цикл
		sc := created.ShortCode
		dom := created.Domain
		go func() {
			_ = h.ownerRepo.Save(r.Context(), sc, user.Sub, user.Username, dom)
		}()

		results = append(results, bulkCreateResult{
			Index:     i,
			ShortCode: created.ShortCode,
			ShortURL:  created.ShortURL,
			Status:    "ok",
		})
	}

	writeJSON(w, map[string]any{"results": results}, http.StatusOK)
}

// bulkStatusItem — один элемент запроса на изменение статуса.
type bulkStatusItem struct {
	ShortCode string `json:"shortCode"`
	Domain    string `json:"domain,omitempty"`
	Active    bool   `json:"active"`
}

// bulkStatusResult — результат обработки одного элемента статуса.
type bulkStatusResult struct {
	Index     int    `json:"index"`
	ShortCode string `json:"shortCode"`
	Error     string `json:"error,omitempty"`
	Status    string `json:"status"` // "ok" | "error" | "forbidden"
}

// BulkSetStatus — PUT /api/shlink/short-urls/bulk/status
// Тело: [{"shortCode": "abc", "domain": "...", "active": true/false}]
// Проверяет владельца или PermShortURLsUpdate для каждого элемента.
func (h *BulkHandler) BulkSetStatus(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromCtx(r.Context())
	if user == nil {
		writeJSON(w, map[string]string{"error": "unauthorized"}, http.StatusUnauthorized)
		return
	}

	var items []bulkStatusItem
	if err := json.NewDecoder(r.Body).Decode(&items); err != nil {
		writeJSON(w, map[string]string{"error": "invalid request body"}, http.StatusBadRequest)
		return
	}

	limit := h.cfg.BulkOperationLimit
	if limit <= 0 {
		limit = 100
	}
	if len(items) > limit {
		writeJSON(w, map[string]string{
			"error": fmt.Sprintf("bulk limit exceeded: max %d items", limit),
		}, http.StatusBadRequest)
		return
	}

	canGlobal, _ := h.permCtrl.Check(r.Context(), user.ID, domain.PermShortURLsUpdate)

	results := make([]bulkStatusResult, 0, len(items))
	for i, item := range items {
		if item.ShortCode == "" {
			results = append(results, bulkStatusResult{
				Index:     i,
				ShortCode: item.ShortCode,
				Error:     "shortCode is required",
				Status:    "error",
			})
			continue
		}

		// Проверяем попуначально: владелец или глобальное право
		isOwner, _ := h.ownerRepo.IsOwner(r.Context(), item.ShortCode, item.Domain, user.Sub)
		if !isOwner && !canGlobal {
			results = append(results, bulkStatusResult{
				Index:     i,
				ShortCode: item.ShortCode,
				Error:     fmt.Sprintf("forbidden: shortCode=%s", item.ShortCode),
				Status:    "forbidden",
			})
			continue
		}

		var err error
		if item.Active {
			err = h.ownerRepo.Activate(r.Context(), item.ShortCode, item.Domain)
		} else {
			err = h.ownerRepo.Deactivate(r.Context(), item.ShortCode, item.Domain, user.Sub)
		}
		if err != nil {
			results = append(results, bulkStatusResult{
				Index:     i,
				ShortCode: item.ShortCode,
				Error:     err.Error(),
				Status:    "error",
			})
			continue
		}

		results = append(results, bulkStatusResult{
			Index:     i,
			ShortCode: item.ShortCode,
			Status:    "ok",
		})
	}

	// Если хоть один элемент forbidden — вернуть 207 Multi-Status + подробности
	statusCode := http.StatusOK
	for _, res := range results {
		if res.Status == "forbidden" || res.Status == "error" {
			statusCode = http.StatusMultiStatus
			break
		}
	}

	writeJSON(w, map[string]any{"results": results}, statusCode)
}
