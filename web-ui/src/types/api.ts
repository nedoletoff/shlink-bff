// ─── Auth ───────────────────────────────────────────────────────────────────────
export type UserRole = string

export interface MeResponse {
  sub: string
  username: string
  email: string
  role: UserRole
  permissions: Record<string, boolean>
  hasApiKey: boolean
  features: { userSlugPrefixEnabled: boolean; userTagInternalIdEnabled: boolean }
  slugPrefix?: string
}

// ─── Short URLs ──────────────────────────────────────────────────────────────────
export interface ShortURL {
  shortCode: string
  shortUrl: string
  longUrl: string
  title: string
  dateCreated: string
  tags: string[]
  visitsSummary: { total: number; nonBots: number }
  maxVisits?: number | null
  validSince?: string | null
  validUntil?: string | null
  isActive?: boolean
}

export interface ShortURLsResponse {
  shortUrls: {
    data: ShortURL[]
    pagination: {
      currentPage: number
      pagesCount: number
      itemsInCurrentPage: number
      itemsPerPage: number
      totalItems: number
    }
  }
}

export interface CreateShortURLPayload {
  longUrl: string
  title?: string
  customSlug?: string
  tags?: string[]
  maxVisits?: number
  validSince?: string
  validUntil?: string
}

export type EditShortURLPayload = Omit<CreateShortURLPayload, 'customSlug'>

// ─── URL Detail ───────────────────────────────────────────────────────────────────
export interface ClickPoint { date: string; clicks: number }
export interface NamedCount { name: string; count: number }
export interface DeviceBreakdown { desktop: number; mobile: number; tablet: number }

export interface VisitRow {
  date: string
  country?: string
  referer?: string
  browser: string
  os: string
  device: string
}

export interface URLDetailResponse {
  shortCode: string
  title: string
  shortUrl: string
  longUrl: string
  dateCreated: string
  visitsTotal: number
  clicksPerDay: ClickPoint[]
  devices: DeviceBreakdown
  browsers: NamedCount[]
  os: NamedCount[]
  visits: VisitRow[]
  isActive: boolean
  deactivatedAt?: string
  deactivatedBy?: string
}

// ─── Tags ──────────────────────────────────────────────────────────────────────────
export interface TagEntry {
  tag: string
  shortUrlsCount: number
  visitsSummary: { total: number }
}

// ─── Dashboard ─────────────────────────────────────────────────────────────────────
export interface HeatCell { weekday: number; hour: number; value: number }

export interface DashboardResponse {
  overview: {
    linksCount: number
    visitsTotal: number
    topLinks: Array<{ shortCode: string; shortUrl: string; longUrl: string; title: string; visitsTotal: number }>
    recentLinks: Array<{ shortCode: string; shortUrl: string; longUrl: string; title: string; visitsTotal: number }>
  }
  visits: { clicksPerDay: ClickPoint[]; clicksTotal: number }
  devices: {
    devices: DeviceBreakdown
    browsers: NamedCount[]
    os: NamedCount[]
    heatmap: HeatCell[]
  }
  users?: Array<{ sub: string; username: string; email: string; role: string; status: string }>
  tags?: Array<{ tag: string; visits: number; urls: number }>
}

// ─── Settings ────────────────────────────────────────────────────────────────────
export interface ServerSettings {
  shortCodeLength: number
  allowCustomSlugs: boolean
  userSlugPrefix: boolean
  domain: string
  shlinkVersion: string
  connected: boolean
  maxVisitsDefault: number
  linkTtlDefaultDays: number
  adminRole: string
  roleSource: string
  corsAllowedOrigins: string
  shlinkRunnerMode: string
  shlinkContainerName: string
}

export interface PatchSettingsPayload {
  shortCodeLength?: number
  allowCustomSlugs?: boolean
  userSlugPrefix?: boolean
  domain?: string
  maxVisitsDefault?: number
  linkTtlDefaultDays?: number
  adminRole?: string
  shlinkRunnerMode?: string
  shlinkContainerName?: string
}

// ─── Admin — Users ────────────────────────────────────────────────────────────────
export interface UserRecord {
  sub: string
  username: string
  email: string
  role: UserRole
  status: string
  slugPrefix?: string
  shlinkApiKey?: string
  createdAt?: string
  updatedAt?: string
}

// ─── Admin — Roles ────────────────────────────────────────────────────────────────
export interface RolePermissions {
  role: string
  updatedAt?: string
  canViewOwnLinks: boolean
  canViewAllLinks: boolean
  canCreateLinks: boolean
  canCreateWithCustomSlug: boolean
  canCreateWithoutSlug: boolean
  canEditOwnLinks: boolean
  canEditAllLinks: boolean
  canDeleteOwnLinks: boolean
  canDeleteAllLinks: boolean
  canDeactivateOwnLinks: boolean
  canDeactivateAllLinks: boolean
  canReactivateOwnLinks: boolean
  canReactivateAllLinks: boolean
  canDeleteOwnLinksPermanently: boolean
  canDeleteAllLinksPermanently: boolean
  canManageOwnTags: boolean
  canManageAllTags: boolean
  canViewOwnStats: boolean
  canViewAllStats: boolean
  canViewAuditLogs: boolean
  canManageUsers: boolean
  canManageRoles: boolean
}

export interface RoleEntry { role: string; permissions: string[] }
export interface RoleMapping { kcGroup: string; appRole: string }
export interface RolesResponse { roles: RoleEntry[]; mappings: RoleMapping[] }

// ─── Admin — Audit ────────────────────────────────────────────────────────────────
export interface AuditLog {
  id?: number
  userSub: string
  username: string
  role: string
  action: string
  resource: string
  result: string
  details?: Record<string, unknown>
  ipAddress: string
  userAgent?: string
  createdAt: string
}

export interface AuditLogsResponse {
  items: AuditLog[]
  page: number
  perPage: number
  total: number
}
