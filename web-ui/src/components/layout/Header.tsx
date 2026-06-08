import { Group, Burger, Title, ActionIcon, Menu, Avatar, Text } from '@mantine/core'
import { IconLogout, IconUser } from '@tabler/icons-react'
import { useAuth } from '@/contexts/AuthContext'
import { ThemeToggle } from './ThemeToggle'

interface HeaderProps {
  opened: boolean
  onToggle: () => void
}

export function Header({ opened, onToggle }: HeaderProps) {
  const { me } = useAuth()

  const logout = () => { window.location.href = '/auth/logout' }

  return (
    <Group h="100%" px="md" justify="space-between">
      <Group>
        <Burger opened={opened} onClick={onToggle} hiddenFrom="sm" size="sm" />
        <Title order={4} style={{ letterSpacing: '-0.02em' }}>Shlink</Title>
      </Group>

      <Group gap="xs">
        <ThemeToggle />
        <Menu position="bottom-end" shadow="md">
          <Menu.Target>
            <ActionIcon variant="subtle" size="lg" aria-label="Меню пользователя">
              <Avatar size={30} radius="xl" color="teal">
                {me?.username?.[0]?.toUpperCase() ?? <IconUser size={16} />}
              </Avatar>
            </ActionIcon>
          </Menu.Target>
          <Menu.Dropdown>
            <Menu.Label>
              <Text size="sm" fw={500}>{me?.username}</Text>
              <Text size="xs" c="dimmed">{me?.email}</Text>
            </Menu.Label>
            <Menu.Divider />
            <Menu.Item leftSection={<IconLogout size={14} />} color="red" onClick={logout}>
              Выйти
            </Menu.Item>
          </Menu.Dropdown>
        </Menu>
      </Group>
    </Group>
  )
}
