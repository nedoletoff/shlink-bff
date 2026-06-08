import { Center, Loader, Stack, Text, Button } from '@mantine/core'
import { useAuth } from '../hooks/useAuth'

interface Props {
  children: React.ReactNode
  requiredRole?: string
}

export function ProtectedRoute({ children, requiredRole }: Props) {
  const { me: user, isLoading: loading } = useAuth()

  if (loading) {
    return (
      <Center h="100vh">
        <Loader size="xl" />
      </Center>
    )
  }

  if (!user) {
    const returnTo =
      window.location.pathname + window.location.search + window.location.hash
    window.location.href = '/oauth2/start?rd=' + encodeURIComponent(returnTo)
    return null
  }

  if (requiredRole && user.role !== requiredRole) {
    return (
      <Center h="100vh">
        <Stack align="center" gap="md">
          <Text size="xl" fw={700} c="red">403 — Доступ запрещён</Text>
          <Text c="dimmed">У вас нет прав для просмотра этой страницы.</Text>
          <Button variant="light" onClick={() => window.history.back()}>
            Назад
          </Button>
        </Stack>
      </Center>
    )
  }

  return <>{children}</>
}
