# Shlink BFF Web UI v2 — Implementation Status

> Last updated automatically. Edit manually when completing a stage.

## Stages

### ✅ Stage 1 — Infrastructure (commit 1/4)
- [x] `package.json` — updated to React 19, Vite 8, Mantine 9, Router 7, Recharts 3, Axios, TanStack Query 5
- [x] `types/api.ts` — all TypeScript interfaces
- [x] `api/client.ts` — Axios instance + error interceptor
- [x] `api/endpoints/auth.ts`
- [x] `api/endpoints/links.ts`
- [x] `api/endpoints/linkDetail.ts`
- [x] `api/endpoints/tags.ts`
- [x] `api/endpoints/dashboard.ts`
- [x] `api/endpoints/settings.ts`
- [x] `api/endpoints/adminUsers.ts`
- [x] `api/endpoints/adminRoles.ts`
- [x] `api/endpoints/adminAudit.ts`
- [x] `api/endpoints/adminSettings.ts`
- [x] `utils/date.ts`
- [x] `utils/errors.ts`

### ⏳ Stage 2 — Shell & shared components (commit 2/4)
- [ ] `main.tsx` — QueryClient provider + Mantine 9 provider
- [ ] `contexts/AuthContext.tsx` — `can()`, `isAdmin()`, React Query
- [ ] `components/layout/AppShell.tsx`
- [ ] `components/layout/Sidebar.tsx`
- [ ] `components/layout/Header.tsx`
- [ ] `components/layout/ThemeToggle.tsx`
- [ ] `components/ui/ConfirmDialog.tsx`
- [ ] `components/ui/EmptyState.tsx`
- [ ] `components/ui/StatCard.tsx`
- [ ] `components/ui/CopyButton.tsx`
- [ ] `components/ui/StatusBadge.tsx`
- [ ] `components/ui/RoleBadge.tsx`
- [ ] `components/ui/PermissionGuard.tsx`
- [ ] `App.tsx` — routes, RequireAuth, RequireAdmin

### ⏳ Stage 3 — User pages (commit 3/4)
- [ ] `pages/Dashboard.tsx`
- [ ] `pages/ShortUrls.tsx`
- [ ] `pages/UrlDetail.tsx`
- [ ] `pages/Tags.tsx`

### ⏳ Stage 4 — Admin pages (commit 4/4)
- [ ] `pages/admin/Users.tsx`
- [ ] `pages/admin/UserDetail.tsx`
- [ ] `pages/admin/Roles.tsx`
- [ ] `pages/admin/AuditLogs.tsx`
- [ ] `pages/admin/Settings.tsx`

## Stack

| Package | Version |
|---|---|
| React | 19 |
| Vite | 8 |
| TypeScript | 5 |
| Mantine | 9 |
| React Router | 7 |
| Axios | 1 |
| TanStack Query | 5 |
| Recharts | 3 |
| dayjs | 1 |
| @tabler/icons-react | 3 |
