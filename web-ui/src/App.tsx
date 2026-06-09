import { Navigate, Route, Routes, useLocation } from 'react-router-dom'
import { Center, Loader, Title, Text, Button, Stack } from '@mantine/core'
import { Component, type ReactNode, type ErrorInfo } from 'react'
import { AppShellWrapper } from '@/components/layout/AppShellWrapper'
import { useAuth } from '@/hooks/useAuth'
import { Dashboard } from '@/pages/Dashboard'
import { ShortUrls } from '@/pages/ShortUrls'
import { UrlDetail } from '@/pages/UrlDetail'
import { Tags } from '@/pages/Tags'
import { AdminUsers } from '@/pages/admin/Users'
import { AdminUserDetail } from '@/pages/admin/UserDetail'
import { AdminRoles } from '@/pages/admin/Roles'
import { AdminAuditLogs } from '@/pages/admin/AuditLogs'
import { AdminSettings } from '@/pages/admin/Settings'

// ─── ErrorBoundary ────────────────────────────────────────────────────────────
// Без него любой необработанный рендер-exception размонтирует всё дерево React
// и оставляет белый экран без возможности восстановления без F5.
interface EBState { hasError: boolean; message: string }

class ErrorBoundary extends Component<{ children: ReactNode }, EBState> {
  state: EBState = { hasError: false, message: '' }

  static getDerivedStateFromError(err: unknown): EBState {
    return {
      hasError: true,
      message: err instanceof Error ? err.message : String(err),
    }
  }

  componentDidCatch(err: Error, info: ErrorInfo) {
    console.error('[ErrorBoundary]', err, info.componentStack)
  }

  render() {
    if (this.state.hasError) {
      return (
        <Center h="100vh">
          <Stack align="center" gap="sm">
            <Title order={2}>Что-то пошло не так</Title>
            <Text c="dimmed" size="sm">{this.state.message}</Text>
            <Button
              variant="default"
              onClick={() => {
                this.setState({ hasError: false, message: '' })
                window.location.href = '/'
              }}
            >
              На главную
            </Button>
          </Stack>
        </Center>
      )
    }
    return this.props.children
  }
}

// ─── Guards ───────────────────────────────────────────────────────────────────
function RequireAuth({ children }: { children: ReactNode }) {
  const { me, isLoading } = useAuth()
  const location = useLocation()

  if (isLoading) {
    return (
      <Center h="100vh">
        <Loader size="lg" />
      </Center>
    )
  }

  if (!me) {
    window.location.href = `/auth/login?rd=${encodeURIComponent(location.pathname)}`
    return null
  }

  return <>{children}</>
}

function RequireAdmin({ children }: { children: ReactNode }) {
  const { isAdmin } = useAuth()

  if (!isAdmin()) {
    return (
      <Center h="60vh">
        <Stack align="center">
          <Title order={2}>403</Title>
          <Text c="dimmed">Недостаточно прав доступа</Text>
          <Button variant="default" onClick={() => (window.location.href = '/')}>На главную</Button>
        </Stack>
      </Center>
    )
  }

  return <>{children}</>
}

// ─── App ──────────────────────────────────────────────────────────────────────
export default function App() {
  return (
    <ErrorBoundary>
      <RequireAuth>
        <AppShellWrapper>
          <ErrorBoundary>
            <Routes>
              <Route path="/" element={<Dashboard />} />
              <Route path="/links" element={<ShortUrls />} />
              <Route path="/links/:shortCode" element={<UrlDetail />} />
              <Route path="/tags" element={<Tags />} />
              <Route path="/admin/users" element={<RequireAdmin><AdminUsers /></RequireAdmin>} />
              <Route path="/admin/users/:sub" element={<RequireAdmin><AdminUserDetail /></RequireAdmin>} />
              <Route path="/admin/roles" element={<RequireAdmin><AdminRoles /></RequireAdmin>} />
              <Route path="/admin/audit" element={<RequireAdmin><AdminAuditLogs /></RequireAdmin>} />
              <Route path="/admin/settings" element={<RequireAdmin><AdminSettings /></RequireAdmin>} />
              <Route path="*" element={<Navigate to="/" replace />} />
            </Routes>
          </ErrorBoundary>
        </AppShellWrapper>
      </RequireAuth>
    </ErrorBoundary>
  )
}
