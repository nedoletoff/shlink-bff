import { Stack, NavLink } from '@mantine/core';
import { useNavigate, useLocation } from 'react-router-dom';
import {
  IconLayoutDashboard, IconLink, IconTags,
  IconUsers, IconFileText,
} from '@tabler/icons-react';
import { useAuth } from '../../contexts/AuthContext';

const userNav = [
  { label: 'Dashboard',  href: '/dashboard', icon: <IconLayoutDashboard size={18} /> },
  { label: 'Ссылки',     href: '/links',     icon: <IconLink size={18} /> },
  { label: 'Теги',       href: '/tags',      icon: <IconTags size={18} /> },
];

const adminNav = [
  { label: 'Пользователи', href: '/admin/users', icon: <IconUsers size={18} /> },
  { label: 'Аудит',        href: '/admin/logs',  icon: <IconFileText size={18} /> },
];

export function Sidebar() {
  const { user }  = useAuth();
  const navigate  = useNavigate();
  const location  = useLocation();

  const nav = user?.role === 'admin' ? [...userNav, ...adminNav] : userNav;

  return (
    <Stack gap={4}>
      {nav.map((item) => (
        <NavLink
          key={item.href}
          label={item.label}
          leftSection={item.icon}
          active={location.pathname.startsWith(item.href)}
          onClick={() => navigate(item.href)}
        />
      ))}
    </Stack>
  );
}
