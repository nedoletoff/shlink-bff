import { AppShell, Burger, Group, Text } from '@mantine/core';
import { useDisclosure } from '@mantine/hooks';
import { Outlet } from 'react-router-dom';
import { Sidebar } from './Sidebar';
import LogoutButton from '../LogoutButton';
import { useAuth } from '../../contexts/AuthContext';

export function AppLayout() {
  const [opened, { toggle }] = useDisclosure();
  const { user } = useAuth();

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
              {/* Кнопка выхода подключена (#25): ранее LogoutButton был реализован, но не использовался */}
              <LogoutButton />
            </Group>
          )}
        </Group>
      </AppShell.Header>

      <AppShell.Navbar p="md">
        <Sidebar />
      </AppShell.Navbar>

      <AppShell.Main>
        <Outlet />
      </AppShell.Main>
    </AppShell>
  );
}
