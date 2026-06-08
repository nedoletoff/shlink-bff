import { Group, Text, Avatar, Menu, UnstyledButton, Burger } from '@mantine/core'
import { IconLogout, IconUser } from '@tabler/icons-react'
import { useAuth } from '@/hooks/useAuth'

interface Props {
  opened: boolean
  onToggle: () => void
}

export function Header({ opened, onToggle }: Props) {
  const { me, isAdmin } = useAuth()

  return (
    <Group h="100%" px="md" justify="space-between">
      <Group>
        <Burger opened={opened} onClick={onToggle} hiddenFrom="sm" size="sm" />
        <Text fw={700} size="lg">Shlink</Text>
      </Group>
      {me && (
        <Menu shadow="md" width={180}>
          <Menu.Target>
            <UnstyledButton>
              <Group gap="xs">
                <Avatar size="sm" color="teal" radius="xl">
                  {me.username?.[0]?.toUpperCase()}
                </Avatar>
                <Text size="sm" fw={500} visibleFrom="sm">{me.username}</Text>
              </Group>
            </UnstyledButton>
          </Menu.Target>
          <Menu.Dropdown>
            <Menu.Label>{isAdmin() ? 'Администратор' : 'Пользователь'}</Menu.Label>
            <Menu.Item leftSection={<IconUser size={14} />}>
              {me.email}
            </Menu.Item>
            <Menu.Divider />
            <Menu.Item
              color="red"
              leftSection={<IconLogout size={14} />}
              onClick={() => { window.location.href = '/oauth2/sign_out' }}
            >
              Выйти
            </Menu.Item>
          </Menu.Dropdown>
        </Menu>
      )}
    </Group>
  )
}
