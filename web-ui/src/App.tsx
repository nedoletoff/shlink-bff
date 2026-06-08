import { Navigate, Route, Routes, useLocation } from 'react-router-dom'
import { Center, Loader, Title, Text, Button, Stack } from '@mantine/core'
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

function RequireAuth({ children }: { children: React.ReactNode }) {
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

function RequireAdmin({ children }: { children: React.ReactNode }) {
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

export default function App() {
  return (
    <RequireAuth>
      <AppShellWrapper>
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
      </AppShellWrapper>
    </RequireAuth>
  )
}
