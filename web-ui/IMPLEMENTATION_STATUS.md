# Shlink BFF Web UI v2 — Implementation Status

> Last updated automatically. Edit manually when completing a stage.

## Stages

### ✅ Stage 1 — Infrastructure (commit 1/4)
- [x] `package.json` — React 19, Vite 8, Mantine 9, Router 7, Axios, TanStack Query 5, Recharts 3
- [x] `types/api.ts` — all TypeScript interfaces
- [x] `api/client.ts` — Axios instance + error interceptor
- [x] `api/endpoints/*` — all 9 endpoint files
- [x] `utils/date.ts`, `utils/errors.ts`

### ✅ Stage 2 — Shell & shared components (commit 2/4)
- [x] `main.tsx` — QueryClientProvider + MantineProvider + DatesProvider
- [x] `contexts/AuthContext.tsx` — React Query, `can()`, `isAdmin()`
- [x] `components/layout/AppShellWrapper.tsx`
- [x] `components/layout/Sidebar.tsx`
- [x] `components/layout/Header.tsx`
- [x] `components/layout/ThemeToggle.tsx`
- [x] `components/ui/ConfirmDialog.tsx`
- [x] `components/ui/EmptyState.tsx`
- [x] `components/ui/StatCard.tsx`
- [x] `components/ui/CopyButton.tsx`
- [x] `components/ui/StatusBadge.tsx`
- [x] `components/ui/RoleBadge.tsx`
- [x] `components/ui/PermissionGuard.tsx`
- [x] `App.tsx`

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
