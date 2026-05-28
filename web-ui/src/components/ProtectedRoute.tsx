import { Center, Loader, Stack, Text, Button } from '@mantine/core';
import { useAuth } from '../contexts/AuthContext';
import type { UserRole } from '../types/api';

interface Props {
  children:      React.ReactNode;
  requiredRole?: UserRole;
}

export function ProtectedRoute({ children, requiredRole }: Props) {
  const { user, loading, error } = useAuth();

  if (loading) {
    return (
      <Center h="100vh">
        <Loader size="xl" />
      </Center>
    );
  }

  if (error || !user) {
    // Редирект на oauth2-proxy login
    window.location.href = '/oauth2/sign_in';
    return null;
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
    );
  }

  return <>{children}</>;
}
