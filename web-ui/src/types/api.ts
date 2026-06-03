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

// Статус пользователя — union вместо string (#35)
export type UserStatus = 'active' | 'disabled' | 'pending';

export interface MeResponse {
  sub:         string;
  username:    string;
  email:       string;
  role:        UserRole;
  permissions: Permissions;
  hasApiKey:   boolean; // только флаг — реальный ключ недоступен в браузере
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
  currentPage:        number;
  pagesCount:         number;
  itemsPerPage:       number;
  itemsInCurrentPage: number;
  totalItems:         number;
}

export interface ShortURLsListResponse {
  shortUrls: {
    data:       ShortURL[];
    pagination: Pagination;
  };
}

// Tags
// Shlink v5.x: visitsCount (number) replaced by visitsSummary object
export interface TagStats {
  tag:            string;
  shortUrlsCount: number;
  visitsSummary:  VisitsSummary;
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
  totalClicks:   number;
  activeLinks:   number;
  topTags:       TagCount[];
  clicksOverTime: ClickPoint[];
}


// Audit Logs
export interface AuditLog {
  id:         number;
  createdAt:  string;
  action:     string;
  username:   string | null;
  userSub:    string | null;
  result:     string;
  role:       string | null;
  resource:   string | null;
  ipAddress:  string | null;
}

export interface AuditLogsResponse {
  logs:  AuditLog[];
  total: number;
}

// Admin Users
export interface AdminUser {
  sub:        string;
  username:   string;
  email:      string;
  role:       UserRole;
  slugPrefix: string | null;
  status:     UserStatus; // union вместо string (#35)
  hasApiKey:  boolean;
}