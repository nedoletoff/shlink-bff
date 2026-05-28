package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"unified-backend/internal/domain"
)

type AuditRepository struct {
	pool *pgxpool.Pool
}

func NewAuditRepository(pool *pgxpool.Pool) *AuditRepository {
	return &AuditRepository{pool: pool}
}

// sensitiveKeys — поля, которые никогда не попадают в JSONB аудита
var sensitiveKeys = []string{
	"shlink_api_key", "api_key", "apikey", "x-api-key",
	"authorization", "password", "secret", "token",
}

func sanitizeDetails(d map[string]any) map[string]any {
	if d == nil {
		return nil
	}
	result := make(map[string]any, len(d))
	for k, v := range d {
		kl := strings.ToLower(k)
		sensitive := false
		for _, sk := range sensitiveKeys {
			if kl == sk {
				sensitive = true
				break
			}
		}
		if !sensitive {
			result[k] = v
		}
	}
	return result
}

func (r *AuditRepository) Record(ctx context.Context, e *domain.AuditEntry) {
	clean := sanitizeDetails(e.Details)
	detailsJSON, _ := json.Marshal(clean)

	const q = `
		INSERT INTO audit_logs
			(user_sub, username, role, action, resource, result, details, ip_address, user_agent)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`

	if _, err := r.pool.Exec(ctx, q,
		e.UserSub, e.Username, e.Role,
		e.Action, e.Resource, e.Result,
		detailsJSON, e.IPAddress, e.UserAgent,
	); err != nil {
		slog.Error("audit: failed to record",
			"action", e.Action, "sub", e.UserSub, "err", err)
	}
}

type AuditFilter struct {
	Username string
	Action   string
	Result   string
	DateFrom *time.Time
	DateTo   *time.Time
	Page     int
	Limit    int
}

type AuditPage struct {
	Logs  []domain.AuditEntry
	Total int
}

func (r *AuditRepository) List(ctx context.Context, f AuditFilter) (*AuditPage, error) {
	if f.Limit <= 0 {
		f.Limit = 50
	}
	if f.Page <= 0 {
		f.Page = 1
	}
	offset := (f.Page - 1) * f.Limit

	args := []any{}
	where := []string{"1=1"}
	argN := 1

	addArg := func(cond string, val any) {
		where = append(where, fmt.Sprintf(cond, argN))
		args = append(args, val)
		argN++
	}

	if f.Username != "" {
		addArg("username ILIKE $%d", "%"+f.Username+"%")
	}
	if f.Action != "" {
		addArg("action = $%d", f.Action)
	}
	if f.Result != "" {
		addArg("result = $%d", f.Result)
	}
	if f.DateFrom != nil {
		addArg("created_at >= $%d", *f.DateFrom)
	}
	if f.DateTo != nil {
		addArg("created_at <= $%d", *f.DateTo)
	}

	whereClause := strings.Join(where, " AND ")

	countQ := "SELECT COUNT(*) FROM audit_logs WHERE " + whereClause
	var total int
	if err := r.pool.QueryRow(ctx, countQ, args...).Scan(&total); err != nil {
		return nil, err
	}

	listQ := fmt.Sprintf(`
		SELECT id, user_sub, username, role, action, resource, result,
		       COALESCE(details::text, '{}'), COALESCE(ip_address,''), COALESCE(user_agent,''), created_at
		FROM audit_logs
		WHERE %s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d`,
		whereClause, argN, argN+1,
	)
	listArgs := append(args, f.Limit, offset)

	rows, err := r.pool.Query(ctx, listQ, listArgs...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []domain.AuditEntry
	for rows.Next() {
		var e domain.AuditEntry
		var detailsStr string
		if err := rows.Scan(
			&e.ID, &e.UserSub, &e.Username, &e.Role,
			&e.Action, &e.Resource, &e.Result,
			&detailsStr, &e.IPAddress, &e.UserAgent, &e.CreatedAt,
		); err != nil {
			return nil, err
		}
		_ = json.Unmarshal([]byte(detailsStr), &e.Details)
		logs = append(logs, e)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return &AuditPage{Logs: logs, Total: total}, nil
}
