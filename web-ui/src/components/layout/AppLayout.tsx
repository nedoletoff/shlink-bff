import { AppShell, Burger, Group, Text, Box } from '@mantine/core';
import { useDisclosure } from '@mantine/hooks';
import { Outlet, useLocation } from 'react-router-dom';
import { Sidebar } from './Sidebar';
import LogoutButton from '../LogoutButton';
import { useAuth } from '../../contexts/AuthContext';

/** Страницы пользовательской зоны — контент центрируется, max-width 900px */
const USER_ROUTES = ['/dashboard', '/links', '/tags'];

function isUserRoute(pathname: string): boolean {
  return USER_ROUTES.some(r => pathname === r || pathname.startsWith(r + '/'));
}

export function AppLayout() {
  const [opened, { toggle }] = useDisclosure();
  const { user } = useAuth();
  const location = useLocation();

  const userZone = isUserRoute(location.pathname);

  return (
    <AppShell
      header={{ height: 56 }}
      navbar={{ width: 240, breakpoint: 'sm', collapsed: { mobile: !opened } }}
      padding="md"
    >
      <AppShell.Header>
        <Group h="100%" px="md">
          <Burger opened={opened} onClick={toggle} hiddenFrom="sm" size="sm" />
          <Text fw={700} size="lg">Shlink Manager</Text>
          {user && (
            <Group ml="auto" gap="sm">
              <Text size="sm" c="dimmed">
                {user.username} · <b>{user.role}</b>
              </Text>
              <LogoutButton />
            </Group>
          )}
        </Group>
      </AppShell.Header>

      <AppShell.Navbar p="md">
        <Sidebar />
      </AppShell.Navbar>

      <AppShell.Main>
        {userZone ? (
          /* Пользовательская зона: центрируем, добавляем воздух */
          <Box
            maw={900}
            mx="auto"
            px={{ base: 'md', sm: 'xl' }}
            py="md"
          >
            <Outlet />
          </Box>
        ) : (
          /* Админ-зона: full-width, плотный layout */
          <Box px="md" py="md">
            <Outlet />
          </Box>
        )}
      </AppShell.Main>
    </AppShell>
  );
}
