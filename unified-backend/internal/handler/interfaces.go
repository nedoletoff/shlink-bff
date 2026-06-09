package handler

import (
	"context"
	"unified-backend/internal/domain"
)

// OwnershipRepo – интерфейс для работы с владением и метаданными ссылок
type OwnershipRepo interface {
	// Базовые методы
	IsOwner(ctx context.Context, shortCode, domain, sub string) (bool, error)
	Deactivate(ctx context.Context, shortCode, domain, actorSub string) error
	Activate(ctx context.Context, shortCode, domain string) error
	SoftDelete(ctx context.Context, shortCode, domain, actorSub string) error
	HardDelete(ctx context.Context, shortCode, domain string) error
	GetShortCodeSet(ctx context.Context, ownerSub string) (map[string]struct{}, error)
	GetActiveCodeSet(ctx context.Context, ownerSub string) (map[string]struct{}, error)
	GetStatusCodeSet(ctx context.Context, ownerSub string) (map[string]bool, error)

	// Метаданные
	Save(ctx context.Context, shortCode, ownerSub, ownerUsername, domain string, metadata *domain.ShortURLMetadata) error
	GetOwnership(ctx context.Context, shortCode, domain string) (*domain.ShortURLMetadata, error)
	GetBatch(ctx context.Context, shortCodes []string, domain string) (map[string]*domain.ShortURLMetadata, error)
	GetAllByOwner(ctx context.Context, ownerSub string) ([]*domain.ShortURLMetadata, error)
}

// AuditRepo – интерфейс для аудита
type AuditRepo interface {
	Record(ctx context.Context, entry *domain.AuditEntry)
}

