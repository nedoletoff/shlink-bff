// Контракт GET /api/me — API key НИКОГДА не присутствует
export interface Permissions {
  canCreateShortUrl:      boolean;
  canCreateWithCustomSlug: boolean;
  canEditOwnLinks:        boolean;
  canDeleteOwnLinks:      boolean;
  canManageOwnTags:       boolean;
  canViewAuditLogs:       boolean;
  canManageUsers:         boolean;
}

export interface FeatureFlags {
  userSlugPrefixEnabled:    boolean;
  userTagInternalIdEnabled: boolean;
}

export type UserRole = 'admin' | 'user';
export type UserStatus = 'active' | 'disabled' | 'pending';

export interface MeResponse {
  sub:         string;
  username:    string;
  email:       string;
  role:        UserRole;
  permissions: Permissions;
  hasApiKey:   boolean;
  features:    FeatureFlags;
  slugPrefix?: string;
}

// Short URL
export interface VisitsSummary {
  total: number;
}

export interface ShortURL {
  shortCode:      string;
  shortUrl:       string;
  longUrl:        string;
  title:          string;
  tags:           string[];
  visitsSummary:  VisitsSummary;
  dateCreated:    string;
  ownerUsername?: string;   // возвращается только admin-у
  isActive?:      boolean;  // false = деактивирована
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
export interface TagStats {
  tag:            string;
  shortUrlsCount: number;
  visitsSummary:  VisitsSummary;
}

export interface TagsResponse {
  tags: { data: TagStats[] };
}

// Dashboard — базовые типы (Overview)
export interface TagCount    { tag:    string; count:  number; }
export interface ClickPoint  { date:   string; clicks: number; }

export interface DashboardResponse {
  totalClicks:    number;
  activeLinks:    number;
  topTags:        TagCount[];
  clicksOverTime: ClickPoint[];
}

// Dashboard — расширенные типы для вкладок 1–4
export interface TopLink {
  shortCode:  string;
  title:      string;
  shortUrl:   string;
  visits:     number;
}

export interface OverviewResponse {
  totalClicks:    number;
  activeLinks:    number;
  createdPeriod:  number;
  uniqueVisitors: number | null;
  clicksPerDay:   ClickPoint[];
  topLinks:       TopLink[];
}

export interface UserActivityRow {
  sub:            string;
  username:       string;
  linksCount:     number;
  visitsCount:    number;
  lastActivityAt: string | null;
}

export interface UserActivityResponse {
  users:          UserActivityRow[];
  newLinksPerDay: ClickPoint[];
}

export interface UrlStatRow {
  shortCode:   string;
  title:       string;
  shortUrl:    string;
  visitsToday: number;
  visits7d:    number;
  visitsTotal: number;
  status:      string;
  tags:        string[];
}

export interface UrlStatsResponse {
  urls: UrlStatRow[];
}

export interface DeviceBreakdown {
  desktop: number;
  mobile:  number;
  tablet:  number;
}

export interface NamedCount { name: string; count: number; }

export interface HeatmapCell {
  hour:    number;  // 0-23
  weekday: number;  // 0=вс, 1=пн … 6=сб
  value:   number;
}

export interface DevicesResponse {
  devices:  DeviceBreakdown;
  browsers: NamedCount[];
  os:       NamedCount[];
  heatmap?: HeatmapCell[];
}

// URL detail page
export interface VisitRow {
  date:    string;
  device:  string;
  os:      string;
  referer: string | null;
  // country намеренно убрана: shlink не возвращает её без GeoIP-подписки
}

export interface UrlDetailResponse {
  shortCode:      string;
  title:          string;
  shortUrl:       string;
  longUrl:        string;
  dateCreated:    string;
  ownerUsername?: string;
  isActive?:      boolean;
  clicksPerDay:   ClickPoint[];
  devices:        DeviceBreakdown;
  browsers:       NamedCount[];
  os:             NamedCount[];
  visits:         VisitRow[];
  visitsTotal:    number;
}

// User detail page
export interface UserDetailResponse {
  sub:         string;
  username:    string;
  email:       string;
  role:        UserRole;
  linksCount:  number;
  visitsTotal: number;
  activityPerDay: ClickPoint[];
  links:       ShortURL[];
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
  status:     UserStatus;
  hasApiKey:  boolean;
}

// Admin Roles
export interface RolePermissions {
  role:                    string;
  canViewOwnLinks:         boolean;
  canViewAllLinks:         boolean;
  canCreateLinks:          boolean;
  canCreateWithCustomSlug: boolean;
  canCreateWithoutSlug:    boolean;
  canEditOwnLinks:         boolean;
  canEditAllLinks:         boolean;
  canDeleteOwnLinks:       boolean;
  canDeleteAllLinks:       boolean;
  canManageOwnTags:        boolean;
  canManageAllTags:        boolean;
  canViewOwnStats:         boolean;
  canViewAllStats:         boolean;
  canViewAuditLogs:        boolean;
  canManageUsers:          boolean;
  canManageRoles:          boolean;
  updatedAt:               string;
}

export interface RoleEntry {
  role:        string;
  permissions: string[];
  usersCount:  number;
}

export interface RoleMapping {
  kcGroup: string;
  appRole: string;
}

export interface RolesResponse {
  roles:    RoleEntry[];
  mappings: RoleMapping[];
}

// Admin Settings
export interface ShlinkSettings {
  shortCodeLength:  number;
  allowCustomSlugs: boolean;
  userSlugPrefix:   boolean;
  domain:           string;
  shlinkVersion:    string;
  connected:        boolean;
}
