import { NavLink, Stack, Text, Divider, Anchor } from '@mantine/core'
import {
  IconHome,
  IconLink,
  IconTag,
  IconUsers,
  IconShield,
  IconClipboardList,
  IconSettings,
  IconExternalLink,
} from '@tabler/icons-react'
import { useNavigate, useLocation } from 'react-router-dom'
import { useAuth } from '@/contexts/AuthContext'

interface Props {
  onNavigate?: () => void
}

export function Sidebar({ onNavigate }: Props) {
  const navigate = useNavigate()
  const location = useLocation()
  const { can, isAdmin } = useAuth()

  const nav = (path: string) => {
    navigate(path)
    onNavigate?.()
  }

  const isActive = (path: string) =>
    path === '/' ? location.pathname === '/' : location.pathname.startsWith(path)

  return (
    <Stack gap="xs" h="100%">
      <NavLink
        label="Главная"
        leftSection={<IconHome size={16} />}
        active={isActive('/')}
        onClick={() => nav('/')}
      />
      <NavLink
        label="Мои ссылки"
        leftSection={<IconLink size={16} />}
        active={isActive('/links')}
        onClick={() => nav('/links')}
      />
      {can('canManageOwnTags') && (
        <NavLink
          label="Теги"
          leftSection={<IconTag size={16} />}
          active={isActive('/tags')}
          onClick={() => nav('/tags')}
        />
      )}

      {isAdmin() && (
        <>
          <Divider label="Администрирование" labelPosition="center" my="xs" />
          <NavLink
            label="Пользователи"
            leftSection={<IconUsers size={16} />}
            active={isActive('/admin/users')}
            onClick={() => nav('/admin/users')}
          />
          <NavLink
            label="Роли"
            leftSection={<IconShield size={16} />}
            active={isActive('/admin/roles')}
            onClick={() => nav('/admin/roles')}
          />
          <NavLink
            label="Аудит"
            leftSection={<IconClipboardList size={16} />}
            active={isActive('/admin/audit')}
            onClick={() => nav('/admin/audit')}
          />
          <NavLink
            label="Настройки"
            leftSection={<IconSettings size={16} />}
            active={isActive('/admin/settings')}
            onClick={() => nav('/admin/settings')}
          />
        </>
      )}

      <Divider my="xs" />
      <Anchor href="https://shlink.io/documentation" target="_blank" size="sm" c="dimmed">
        <NavLink label="Shlink Docs" leftSection={<IconExternalLink size={14} />} />
      </Anchor>
      <Anchor href="https://api-spec.shlink.io" target="_blank" size="sm" c="dimmed">
        <NavLink label="REST API" leftSection={<IconExternalLink size={14} />} />
      </Anchor>

      <Stack mt="auto" gap={2}>
        <Text size="xs" c="dimmed">
          Shlink BFF UI — управление ссылками с ролевым доступом
        </Text>
      </Stack>
    </Stack>
  )
}
