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

  if (!user && !error) {
    // Нет пользователя и нет ошибки — редирект на oauth2
    window.location.href = '/oauth2/start?rd=' + encodeURIComponent(window.location.pathname);
    return null;
  }

  if (error) {
    // Ошибка от /api/me (403 = не провизионирован, 500 = внутренняя)
    // Не редиректим в бесконечный цикл — показываем страницу ошибки
    return (
      <Center h="100vh">
        <Stack align="center" gap="md">
          <Text size="xl" fw={700} c="red">Ошибка авторизации</Text>
          <Text c="dimmed" ta="center" maw={400}>
            {error.includes('403') || error.includes('Forbidden')
              ? 'Ваш аккаунт не провизионирован администратором. Обратитесь к администратору системы.'
              : `Не удалось получить данные профиля: ${error}`
            }
          </Text>
          <Button variant="light" onClick={() => { window.location.href = '/oauth2/sign_out'; }}>
            Выйти
          </Button>
        </Stack>
      </Center>
    );
  }

  if (requiredRole && user!.role !== requiredRole) {
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
