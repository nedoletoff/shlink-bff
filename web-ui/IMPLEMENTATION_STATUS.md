# Shlink BFF Web UI v2 — Implementation Status

> Last updated: 2026-06-08. ✅ = done, ❌ = not done / broken, 🔧 = partially done.

---

## ✅ Done (infrastructure & shell)

- [x] `package.json` — React 19, Vite 6, Mantine 9, Router 7, Axios, TanStack Query 5, Recharts 3, postcss plugins
- [x] `types/api.ts` — all TypeScript interfaces incl. `UserRole`
- [x] `api/client.ts` — Axios instance + error interceptor
- [x] `api/endpoints/*` — all endpoint files
- [x] `utils/date.ts`, `utils/errors.ts`
- [x] `main.tsx`, `App.tsx`, routing, guards
- [x] `AuthContext.tsx` — `can()`, `isAdmin()`
- [x] Layout: `AppShellWrapper`, `Sidebar`, `Header`, `ThemeToggle`
- [x] Shared UI components: `EmptyState`, `ConfirmModal`, `CopyButton`, etc.
- [x] `pages/admin/Users.tsx`, `UserDetail.tsx`, `Roles.tsx`
- [x] `pages/UrlDetail.tsx`

---

## ❌ Remaining work

### 1. Short URLs — Create / Edit form (`pages/ShortUrls.tsx` + modal)

- [ ] Поле **Срок жизни** (`validSince` / `validUntil`) — два `DateTimePicker` поля в форме создания/редактирования
- [ ] Поле **Теги** (`tags`) — `MultiSelect` или `TagsInput` в форме, передающий `tags[]` на backend

### 2. Short URLs — List (`pages/ShortUrls.tsx`)

- [ ] Ссылки помечаются неактивными независимо от поля `isActive` — исправить логику Badge: зелёный если `isActive === true`, серый если `false`
- [ ] Кнопка **Сделать неактивной / Активировать** — toggle `PATCH /links/{code}/deactivate` или `/activate`, icon `IconBan` / `IconCircleCheck`
- [ ] Неактивные ссылки **не отображаются** в списке — добавить фильтр `showInactive: boolean` и тоггл, передавать параметр в API

### 3. Dashboard (`pages/Dashboard.tsx`)

- [ ] **Список ссылок** — `topLinks` (top-5 по визитам) + `recentLinks` (5 последних) отобразить таблицами/карточками
- [ ] **KPI-блоки** — linksCount, visitsTotal, topDomain, activeLinks (4 `StatCard`)
- [ ] **График кликов по дням** (`clicksPerDay`) — `LineChart` (Recharts)
- [ ] **Heatmap активности** по часам/дням недели (`heatmap[]`)
- [ ] **Графики устройств, OS, браузеров** (`PieChart` / `BarChart`)
- [ ] **Список пользователей** (admin-only, секция внизу дашборда)

### 4. Tags (`pages/Tags.tsx`)

- [ ] **Создание тега** — кнопка `+ Тег` → `POST /tags` `{ tag: string }`, обновлять кэш
- [ ] **Переименование тега** — `PATCH /tags/{tag}` `{ tag: newName }`, inline edit или modal
- [ ] **Удаление тега** — `DELETE /tags/{tag}` с confirm

### 5. Audit Logs (`pages/admin/AuditLogs.tsx`)

- [ ] **Скрытые колонки** — отобразить все столбцы: `username`, `role`, `action`, `resource`, `result`, `ipAddress`, `details` — сейчас видны только дата/время
- [ ] **Раскрываемая строка** (expand row) — показывать `details` JSON, `userAgent`
- [ ] **Множественный выбор** записей `checkbox` последней строке + кнопка **Удалить выбранные** → `DELETE /admin/audit` `{ ids: number[] }`

### 6. Settings (`pages/admin/Settings.tsx`)

- [ ] **Статус Shlink** — `connected: boolean` из `/settings` отображается как недоступен. Проверить: либо backend возвращает `connected: false`, либо frontend игнорирует поле — отображать `connected` через `Badge` (зелёный/красный) с дополнительной кнопкой “Проверить соединение” (`GET /settings/health`)

### 7. UI / UX (визуал)

- [ ] **Иконки на KPI-карточках** — `@tabler/icons-react` (`IconLink`, `IconEye`, `IconUsers`, `IconTag`)
- [ ] **Badge-статус ссылки** в списке — зелёный • красный миниатюрный `dot`
- [ ] **Иконки действий** (`IconEdit`, `IconTrash`, `IconBan`, `IconCircleCheck`) во всех action-кнопках — без текста (`ActionIcon`)
- [ ] **Tooltip** на каждой `ActionIcon` (подсказка при hover)
- [ ] **Цвет тегов** — фиксированный palette (hash имени тега → один из 8 цветов Mantine)
- [ ] **Loading skeleton** вместо спиннера на страницах `ShortUrls`, `Tags`, `Dashboard`
- [ ] **Notification toast** после каждого мутабельного действия (`@mantine/notifications`)

---

## Stack (current)

| Package | Version |
|---|---|
| React | 19 |
| Vite | 6 |
| TypeScript | 5 |
| Mantine | 9 |
| React Router | 7 |
| Axios | 1 |
| TanStack Query | 5 |
| Recharts | 3 |
| dayjs | 1 |
| @tabler/icons-react | 3 |
