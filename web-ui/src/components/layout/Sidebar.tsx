import { NavLink, Stack, Text } from '@mantine/core'
import { useNavigate, useLocation } from 'react-router-dom'
import {
  IconHome, IconLink, IconTags, IconUsers,
  IconShield, IconClipboardList, IconSettings,
} from '@tabler/icons-react'
import { useAuth } from '@/hooks/useAuth'

interface Props {
  onClose: () => void
}

const userLinks = [
  { to: '/', label: 'Дашборд', icon: IconHome },
  { to: '/links', label: 'Ссылки', icon: IconLink },
  { to: '/tags', label: 'Теги', icon: IconTags },
]

const adminLinks = [
  { to: '/admin/users', label: 'Пользователи', icon: IconUsers },
  { to: '/admin/roles', label: 'Роли', icon: IconShield },
  { to: '/admin/audit', label: 'Аудит', icon: IconClipboardList },
  { to: '/admin/settings', label: 'Настройки', icon: IconSettings },
]

export function Sidebar({ onClose }: Props) {
  const navigate = useNavigate()
  const location = useLocation()
  const { me, isAdmin } = useAuth()

  const handleNav = (to: string) => {
    navigate(to)
    onClose()
  }

  return (
    <Stack gap={4} p="xs">
      {userLinks.map(({ to, label, icon: Icon }) => (
        <NavLink
          key={to}
          label={label}
          leftSection={<Icon size={16} />}
          active={location.pathname === to}
          onClick={() => handleNav(to)}
        />
      ))}

      {isAdmin() && (
        <>
          <Text size="xs" c="dimmed" px="sm" mt="sm" mb={2} tt="uppercase" fw={600}>
            Администрирование
          </Text>
          {adminLinks.map(({ to, label, icon: Icon }) => (
            <NavLink
              key={to}
              label={label}
              leftSection={<Icon size={16} />}
              active={location.pathname.startsWith(to)}
              onClick={() => handleNav(to)}
            />
          ))}
        </>
      )}

      {me && (
        <Text size="xs" c="dimmed" px="sm" mt="auto" pt="md">
          {me.email}
        </Text>
      )}
    </Stack>
  )
}
