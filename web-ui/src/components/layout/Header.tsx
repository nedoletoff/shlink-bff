import { Group, Text, Avatar, Menu, UnstyledButton } from '@mantine/core'
import { IconLogout, IconUser } from '@tabler/icons-react'
import { useAuth } from '@/hooks/useAuth'
import type { MeResponse } from '@/types/api'

interface Props {
  me: MeResponse | null
}

export function Header({ me }: Props) {
  const { isAdmin } = useAuth()

  return (
    <Group gap="sm">
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
