import { Stack, NavLink, Divider, Text, Box, Badge } from '@mantine/core';
import { useNavigate, useLocation } from 'react-router-dom';
import {
  IconLayoutDashboard, IconLink, IconTags,
  IconUsers, IconFileText, IconShield, IconSettings,
} from '@tabler/icons-react';
import { useAuth } from '../../contexts/AuthContext';
import { RU } from '../../i18n/ru';

const userNav = [
  { label: RU.nav.dashboard, href: '/dashboard', icon: <IconLayoutDashboard size={16} /> },
  { label: RU.nav.links,     href: '/links',     icon: <IconLink size={16} /> },
  { label: RU.nav.tags,      href: '/tags',      icon: <IconTags size={16} /> },
];

const adminNav = [
  { label: RU.nav.users,    href: '/admin/users',    icon: <IconUsers size={16} /> },
  { label: RU.nav.roles,    href: '/admin/roles',    icon: <IconShield size={16} /> },
  { label: RU.nav.audit,    href: '/admin/logs',     icon: <IconFileText size={16} /> },
  { label: RU.nav.settings, href: '/admin/settings', icon: <IconSettings size={16} /> },
];

export function Sidebar() {
  const { user }  = useAuth();
  const navigate  = useNavigate();
  const location  = useLocation();
  const isAdmin   = user?.role === 'admin';

  return (
    <Stack gap={2} style={{ height: '100%' }}>
      {userNav.map(item => (
        <NavLink
          key={item.href}
          label={item.label}
          leftSection={item.icon}
          active={location.pathname === item.href || location.pathname.startsWith(item.href + '/')}
          onClick={() => navigate(item.href)}
          styles={{ root: { borderRadius: 6 } }}
        />
      ))}

      {isAdmin && (
        <>
          <Divider my="xs" />
          <Text size="xs" c="dimmed" px="sm" pb={2} tt="uppercase" fw={600} style={{ letterSpacing: '0.05em' }}>
            Администрирование
          </Text>
          {adminNav.map(item => (
            <NavLink
              key={item.href}
              label={item.label}
              leftSection={item.icon}
              active={location.pathname.startsWith(item.href)}
              onClick={() => navigate(item.href)}
              styles={{ root: { borderRadius: 6 } }}
              rightSection={
                item.href === '/admin/users'
                  ? <Badge size="xs" variant="light" color="red">admin</Badge>
                  : undefined
              }
            />
          ))}
        </>
      )}

      <Box style={{ flexGrow: 1 }} />
      <Box px="xs" pb="xs">
        <Text size="xs" c="dimmed">v{__APP_VERSION__ ?? '—'}</Text>
      </Box>
    </Stack>
  );
}

declare const __APP_VERSION__: string | undefined;
