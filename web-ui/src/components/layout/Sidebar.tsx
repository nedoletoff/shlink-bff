import { Stack, NavLink, Divider } from '@mantine/core';
import { useNavigate, useLocation } from 'react-router-dom';
import {
  IconLayoutDashboard, IconLink, IconTags,
  IconUsers, IconFileText, IconShield, IconSettings,
} from '@tabler/icons-react';
import { useAuth } from '../../contexts/AuthContext';
import { RU } from '../../i18n/ru';

const userNav = [
  { label: RU.nav.dashboard, href: '/dashboard', icon: <IconLayoutDashboard size={18} /> },
  { label: RU.nav.links,     href: '/links',     icon: <IconLink size={18} /> },
  { label: RU.nav.tags,      href: '/tags',      icon: <IconTags size={18} /> },
];

const adminNav = [
  { label: RU.nav.users,    href: '/admin/users',    icon: <IconUsers size={18} /> },
  { label: RU.nav.roles,    href: '/admin/roles',    icon: <IconShield size={18} /> },
  { label: RU.nav.audit,    href: '/admin/logs',     icon: <IconFileText size={18} /> },
  { label: RU.nav.settings, href: '/admin/settings', icon: <IconSettings size={18} /> },
];

export function Sidebar() {
  const { user }  = useAuth();
  const navigate  = useNavigate();
  const location  = useLocation();
  const isAdmin   = user?.role === 'admin';

  return (
    <Stack gap={4}>
      {userNav.map(item => (
        <NavLink
          key={item.href}
          label={item.label}
          leftSection={item.icon}
          active={location.pathname.startsWith(item.href)}
          onClick={() => navigate(item.href)}
        />
      ))}

      {isAdmin && (
        <>
          <Divider my="xs" label="Админ" labelPosition="left" />
          {adminNav.map(item => (
            <NavLink
              key={item.href}
              label={item.label}
              leftSection={item.icon}
              active={location.pathname.startsWith(item.href)}
              onClick={() => navigate(item.href)}
            />
          ))}
        </>
      )}
    </Stack>
  );
}
