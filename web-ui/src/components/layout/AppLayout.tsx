import { AppShell, Burger, Group, Text, Box, Avatar, Menu, ActionIcon, useMantineColorScheme, Tooltip } from '@mantine/core';
import { useDisclosure } from '@mantine/hooks';
import { Outlet } from 'react-router-dom';
import { IconSun, IconMoon, IconLogout } from '@tabler/icons-react';
import { Sidebar } from './Sidebar';
import { useAuth } from '../../contexts/AuthContext';

export function AppLayout() {
  const [opened, { toggle }] = useDisclosure();
  const { user } = useAuth();
  const { colorScheme, toggleColorScheme } = useMantineColorScheme();

  const initials = user?.username
    ? user.username.slice(0, 2).toUpperCase()
    : '??';

  return (
    <AppShell
      header={{ height: 56 }}
      navbar={{ width: 220, breakpoint: 'sm', collapsed: { mobile: !opened } }}
      padding="lg"
    >
      <AppShell.Header>
        <Group h="100%" px="md" justify="space-between">
          <Group gap="sm">
            <Burger opened={opened} onClick={toggle} hiddenFrom="sm" size="sm" />
            <Group gap={6}>
              <Box
                style={{
                  width: 28, height: 28, borderRadius: 6,
                  background: 'linear-gradient(135deg, #4dabf7 0%, #228be6 100%)',
                  display: 'flex', alignItems: 'center', justifyContent: 'center',
                }}
              >
                <Text size="xs" fw={800} c="white" ff="monospace">SL</Text>
              </Box>
              <Text fw={700} size="sm" visibleFrom="xs">shlink-bff</Text>
            </Group>
          </Group>

          <Group gap="xs">
            <Tooltip label={colorScheme === 'dark' ? 'Светлая тема' : 'Тёмная тема'}>
              <ActionIcon variant="subtle" color="gray" onClick={() => toggleColorScheme()}>
                {colorScheme === 'dark' ? <IconSun size={18} /> : <IconMoon size={18} />}
              </ActionIcon>
            </Tooltip>

            <Menu position="bottom-end" offset={4}>
              <Menu.Target>
                <Avatar
                  size="sm" radius="xl"
                  style={{ cursor: 'pointer' }}
                  color={user?.role === 'admin' ? 'red' : 'blue'}
                >
                  {initials}
                </Avatar>
              </Menu.Target>
              <Menu.Dropdown>
                <Menu.Label>
                  {user?.email ?? user?.username ?? '—'}
                </Menu.Label>
                <Menu.Item
                  color="red"
                  leftSection={<IconLogout size={14} />}
                  component="a"
                  href="/oauth2/sign_out"
                >
                  Выйти
                </Menu.Item>
              </Menu.Dropdown>
            </Menu>
          </Group>
        </Group>
      </AppShell.Header>

      <AppShell.Navbar p="sm">
        <Sidebar />
      </AppShell.Navbar>

      <AppShell.Main>
        <Box maw={1200} mx="auto">
          <Outlet />
        </Box>
      </AppShell.Main>
    </AppShell>
  );
}
