import {
  NavLink, Stack, Divider, Text, Box,
} from '@mantine/core'
import {
  IconLayoutDashboard, IconLink, IconTags,
  IconUsers, IconShield, IconFileText,
  IconSettings,
} from '@tabler/icons-react'
import { useLocation, useNavigate } from 'react-router-dom'
import { useAuth } from '@/contexts/AuthContext'

interface NavItem {
  label: string
  path: string
  icon: React.ReactNode
  adminOnly?: boolean
}

const navItems: NavItem[] = [
  { label: 'Дашборд', path: '/', icon: <IconLayoutDashboard size={18} /> },
  { label: 'Ссылки', path: '/links', icon: <IconLink size={18} /> },
  { label: 'Теги', path: '/tags', icon: <IconTags size={18} /> },
]

const adminItems: NavItem[] = [
  { label: 'Пользователи', path: '/admin/users', icon: <IconUsers size={18} /> },
  { label: 'Роли', path: '/admin/roles', icon: <IconShield size={18} /> },
  { label: 'Аудит', path: '/admin/audit', icon: <IconFileText size={18} /> },
  { label: 'Настройки', path: '/admin/settings', icon: <IconSettings size={18} /> },
]

export function Sidebar({ onClose }: { onClose?: () => void }) {
  const { pathname } = useLocation()
  const navigate = useNavigate()
  const { isAdmin } = useAuth()

  const go = (path: string) => {
    navigate(path)
    onClose?.()
  }

  return (
    <Box h="100%" py="sm">
      <Stack gap={2}>
        {navItems.map((item) => (
          <NavLink
            key={item.path}
            label={item.label}
            leftSection={item.icon}
            active={pathname === item.path}
            onClick={() => go(item.path)}
          />
        ))}

        {isAdmin() && (
          <>
            <Divider my="xs" label={
              <Text size="xs" c="dimmed" fw={600} tt="uppercase" style={{ letterSpacing: '0.05em' }}>Админ</Text>
            } />
            {adminItems.map((item) => (
              <NavLink
                key={item.path}
                label={item.label}
                leftSection={item.icon}
                active={pathname === item.path}
                onClick={() => go(item.path)}
              />
            ))}
          </>
        )}
      </Stack>
    </Box>
  )
}
