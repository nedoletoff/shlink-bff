// Контракт GET /api/me — API key НИКОГДА не присутствует
export interface Permissions {
  canCreateShortUrl: boolean;
  canEditOwnLinks:   boolean;
  canDeleteOwnLinks: boolean;
  canManageOwnTags:  boolean;
  canViewAuditLogs:  boolean;
  canManageUsers:    boolean;
}

export interface FeatureFlags {
  userSlugPrefixEnabled:    boolean;
  userTagInternalIdEnabled: boolean;
}

export type UserRole = 'admin' | 'user';

export interface MeResponse {
  sub:         string;
  username:    string;
  email:       string;
  role:        UserRole;
  permissions: Permissions;
  hasApiKey:   boolean;  // только флаг — реальный ключ недоступен в браузере
  features:    FeatureFlags;
  slugPrefix?: string;
}

// Short URL
export interface VisitsSummary {
  total: number;
}

export interface ShortURL {
  shortCode:     string;
  shortUrl:      string;
  longUrl:       string;
  title:         string;
  tags:          string[];
  visitsSummary: VisitsSummary;
  dateCreated:   string;
}

export interface Pagination {
  currentPage:            number;
  pagesCount:             number;
  itemsPerPage:           number;
  itemsInCurrentPage:     number;
  totalItems:             number;
}

export interface ShortURLsListResponse {
  shortUrls: {
    data:       ShortURL[];
    pagination: Pagination;
  };
}

// Tags
export interface TagStats {
  tag:            string;
  shortUrlsCount: number;
  visitsCount:    number;
}

export interface TagsResponse {
  tags: {
    data: TagStats[];
  };
}

// Dashboard
export interface TagCount {
  tag:   string;
  count: number;
}

export interface ClickPoint {
  date:   string;
  clicks: number;
}

export interface DashboardResponse {
  totalClicks:    number;
  activeLinks:    number;
  topTags:        TagCount[];
  clicksOverTime: ClickPoint[];
}

// Audit log
export interface AuditLog {
  id:        number;
  userSub:   string;
  username:  string;
  role:      string;
  action:    string;
  resource:  string;
  result:    'success' | 'denied' | 'error';
  details:   Record<string, unknown>;
  ipAddress: string;
  createdAt: string;
}

export interface AuditLogsResponse {
  logs:  AuditLog[];
  total: number;
}

// Admin: пользователи (без API key)
export interface AdminUser {
  id:         string;
  sub:        string;
  username:   string;
  email:      string;
  role:       UserRole;
  slugPrefix: string;
  status:     'active' | 'disabled' | 'pending';
  hasApiKey:  boolean;
  createdAt:  string;
}
