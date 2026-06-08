package handler

import (
	"context"

	"unified-backend/internal/domain"
	"unified-backend/internal/repository/postgres"
)

// OwnershipRepo — единый интерфейс для всех хендлеров.
// Реализуется postgres.URLOwnershipRepository.
type OwnershipRepo interface {
	Save(ctx context.Context, shortCode, ownerSub, ownerUsername, domain string) error
	IsOwner(ctx context.Context, shortCode, domain, sub string) (bool, error)
	HardDelete(ctx context.Context, shortCode, domain string) error
	SetActive(ctx context.Context, shortCode, domain string, active bool) error
	GetShortCodeSet(ctx context.Context, ownerSub string) (map[string]struct{}, error)
	GetActiveCodeSet(ctx context.Context, ownerSub string) (map[string]struct{}, error)
	GetStatusCodeSet(ctx context.Context, ownerSub string) (map[string]bool, error)
	Deactivate(ctx context.Context, shortCode, domain, actorSub string) error
	Activate(ctx context.Context, shortCode, domain string) error
	SoftDelete(ctx context.Context, shortCode, domain, actorSub string) error
	GetOwnership(ctx context.Context, shortCode, domain string) (*postgres.URLOwnershipRecord, error)
}

// AuditRepo — единый интерфейс для записи аудита.
// Реализуется postgres.AuditRepository.
type AuditRepo interface {
	Record(ctx context.Context, entry *domain.AuditEntry)
}

// Compile-time checks.
var _ OwnershipRepo = (*postgres.URLOwnershipRepository)(nil)
var _ AuditRepo = (*postgres.AuditRepository)(nil)
