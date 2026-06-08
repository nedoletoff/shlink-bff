package domain

import "time"

type AuditEntry struct {
	ID        int64
	UserSub   string
	Username  string
	Role      string
	Action    string
	Resource  string
	Result    string         // "success" | "denied" | "error"
	Details   map[string]any // JSONB — никогда не содержит API key
	IPAddress string
	UserAgent string
	CreatedAt time.Time
}

// Аудит-события для жизненного цикла ссылок.
const (
	ActionShortURLDeactivated        = "short_url_deactivated"
	ActionShortURLActivated          = "short_url_activated"
	ActionShortURLDeletedPermanently = "short_url_deleted_permanently"
)
