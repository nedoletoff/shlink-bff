import { useEffect, useState } from 'react';
import {
  Stack, Title, Table, Text, Badge, ActionIcon,
  Center, Loader, Modal, TextInput, Group, Button,
  Select, Tooltip,
} from '@mantine/core';
import { useDisclosure } from '@mantine/hooks';
import { notifications } from '@mantine/notifications';
import { IconEdit, IconKey, IconUserCog } from '@tabler/icons-react';
import { api } from '../../api/client';
import { useAuth } from '../../contexts/AuthContext';
import type { AdminUser, UserRole } from '../../types/api';
import { RU } from '../../i18n/ru';

const ROLE_OPTIONS: { value: UserRole; label: string }[] = [
  { value: 'admin', label: 'admin' },
  { value: 'user',  label: 'user'  },
];

export function AdminUsers() {
  const { user: currentUser } = useAuth();
  const [users,   setUsers]   = useState<AdminUser[]>([]);
  const [loading, setLoading] = useState(true);
  const [editUser,       setEditUser]       = useState<AdminUser | null>(null);
  const [roleTarget,     setRoleTarget]     = useState<AdminUser | null>(null);
  const [editOpen,       { open: openEdit,       close: closeEdit       }] = useDisclosure(false);
  const [roleOpen,       { open: openRole,       close: closeRole       }] = useDisclosure(false);
  const [confirmDemote,  { open: openConfirm,    close: closeConfirm    }] = useDisclosure(false);

  const [selectedRole, setSelectedRole] = useState<UserRole>('user');
  const [roleSaving,   setRoleSaving]   = useState(false);

  const fetchUsers = () => {
    api.get<AdminUser[]>('/api/admin/users')
      .then(setUsers)
      .catch(() => setUsers([]))
      .finally(() => setLoading(false));
  };

  useEffect(() => { fetchUsers(); }, []);

  const handleEdit = (u: AdminUser) => { setEditUser(u); openEdit(); };

  const openRoleModal = (u: AdminUser) => {
    setRoleTarget(u);
    setSelectedRole(u.role);
    openRole();
  };

  const applyRoleChange = async () => {
    if (!roleTarget) return;
    // Защита от снятия роли с себя
    if (roleTarget.sub === currentUser?.sub) {
      notifications.show({ message: RU.roles.selfGuard, color: 'red' });
      return;
    }
    // Предупреждение при понижении admin → user
    if (roleTarget.role === 'admin' && selectedRole === 'user') {
      openConfirm();
      return;
    }
    await doRoleChange();
  };

  const doRoleChange = async () => {
    if (!roleTarget) return;
    setRoleSaving(true);
    try {
      // Используем PUT /api/admin/users/:sub (общий update endpoint)
      await api.put(`/api/admin/users/${encodeURIComponent(roleTarget.sub)}`, {
        role: selectedRole,
      });
      notifications.show({ message: 'Роль обновлена', color: 'green' });
      closeRole();
      closeConfirm();
      fetchUsers();
    } catch {
      // handled
    } finally {
      setRoleSaving(false);
    }
  };

  return (
    <Stack gap="lg">
      <Title order={2}>{RU.users.title}</Title>

      {loading ? (
        <Center h={200}><Loader /></Center>
      ) : (
        <Table striped highlightOnHover withTableBorder>
          <Table.Thead>
            <Table.Tr>
              <Table.Th>{RU.users.user}</Table.Th>
              <Table.Th>{RU.users.email}</Table.Th>
              <Table.Th>{RU.users.role}</Table.Th>
              <Table.Th>{RU.users.prefix}</Table.Th>
              <Table.Th>{RU.users.status}</Table.Th>
              <Table.Th>{RU.users.apiKey}</Table.Th>
              <Table.Th />
            </Table.Tr>
          </Table.Thead>
          <Table.Tbody>
            {users.map(u => {
              const isSelf = u.sub === currentUser?.sub;
              return (
                <Table.Tr key={u.sub}>
                  <Table.Td>
                    <Text fw={500}>{u.username}</Text>
                    <Text size="xs" c="dimmed" ff="monospace">{u.sub.slice(0, 8)}…</Text>
                  </Table.Td>
                  <Table.Td>{u.email}</Table.Td>
                  <Table.Td>
                    <Badge color={u.role === 'admin' ? 'red' : 'blue'} variant="light">
                      {u.role}
                    </Badge>
                  </Table.Td>
                  <Table.Td>
                    <Text ff="monospace" size="sm">{u.slugPrefix || '—'}</Text>
                  </Table.Td>
                  <Table.Td>
                    <Badge
                      color={u.status === 'active' ? 'green' : u.status === 'disabled' ? 'red' : 'gray'}
                      variant="dot"
                    >
                      {u.status}
                    </Badge>
                  </Table.Td>
                  <Table.Td>
                    <Badge color={u.hasApiKey ? 'green' : 'orange'} variant="light" size="sm">
                      {u.hasApiKey ? 'Задан' : 'Отсутствует'}
                    </Badge>
                  </Table.Td>
                  <Table.Td>
                    <Group gap={4}>
                      {/* Кнопка смены роли */}
                      <Tooltip
                        label={isSelf ? RU.roles.selfGuard : RU.roles.changeRole}
                        withArrow
                      >
                        <ActionIcon
                          variant="subtle"
                          color="blue"
                          disabled={isSelf}
                          onClick={() => openRoleModal(u)}
                        >
                          <IconUserCog size={16} />
                        </ActionIcon>
                      </Tooltip>
                      {/* Кнопка редактирования (prefix/apikey) */}
                      <ActionIcon variant="subtle" onClick={() => handleEdit(u)}>
                        <IconEdit size={16} />
                      </ActionIcon>
                    </Group>
                  </Table.Td>
                </Table.Tr>
              );
            })}
            {users.length === 0 && (
              <Table.Tr>
                <Table.Td colSpan={7}>
                  <Center p="xl"><Text c="dimmed">{RU.users.notFound}</Text></Center>
                </Table.Td>
              </Table.Tr>
            )}
          </Table.Tbody>
        </Table>
      )}

      {/* Модальное окно смены роли */}
      <Modal
        opened={roleOpen}
        onClose={closeRole}
        title={`${RU.roles.changeRole}: ${roleTarget?.username}`}
        size="sm"
      >
        <Stack gap="sm">
          <Select
            label={RU.roles.role}
            data={ROLE_OPTIONS}
            value={selectedRole}
            onChange={v => setSelectedRole((v ?? 'user') as UserRole)}
          />
          <Group justify="flex-end" mt="md">
            <Button variant="default" onClick={closeRole}>{RU.cancel}</Button>
            <Button
              onClick={applyRoleChange}
              loading={roleSaving}
              disabled={selectedRole === roleTarget?.role}
            >
              {RU.save}
            </Button>
          </Group>
        </Stack>
      </Modal>

      {/* Подтверждение понижения admin → user */}
      <Modal
        opened={confirmDemote}
        onClose={closeConfirm}
        title="Подтвердите смену роли"
        size="sm"
      >
        <Stack gap="md">
          <Text size="sm">{RU.roles.confirmDemotion}</Text>
          <Group justify="flex-end">
            <Button variant="default" onClick={closeConfirm}>{RU.cancel}</Button>
            <Button color="orange" onClick={doRoleChange} loading={roleSaving}>
              {RU.confirm}
            </Button>
          </Group>
        </Stack>
      </Modal>

      {/* Модаль: prefix + API key */}
      <EditUserModal
        opened={editOpen}
        user={editUser}
        onClose={closeEdit}
        onSaved={() => { closeEdit(); fetchUsers(); }}
      />
    </Stack>
  );
}

function EditUserModal({
  opened, user, onClose, onSaved,
}: {
  opened: boolean; user: AdminUser | null;
  onClose: () => void; onSaved: () => void;
}) {
  const [prefix,     setPrefix]     = useState('');
  const [newKey,     setNewKey]     = useState('');
  const [confirmKey, setConfirmKey] = useState('');
  const [loading,    setLoading]    = useState(false);

  useEffect(() => {
    if (user) { setPrefix(user.slugPrefix ?? ''); setNewKey(''); setConfirmKey(''); }
  }, [user]);

  if (!user) return null;

  const savePrefix = async () => {
    setLoading(true);
    try {
      await api.put(`/api/admin/users/${encodeURIComponent(user.sub)}/prefix`, { prefix });
      notifications.show({ message: 'Префикс обновлён', color: 'green' });
      onSaved();
    } catch { /* handled */ } finally { setLoading(false); }
  };

  const keysMatch = newKey.trim() !== '' && newKey === confirmKey;

  const saveApiKey = async () => {
    if (!newKey.trim()) return;
    if (newKey !== confirmKey) {
      notifications.show({ message: 'Ключи не совпадают — проверьте ввод', color: 'red' });
      return;
    }
    setLoading(true);
    try {
      await api.put(`/api/admin/users/${encodeURIComponent(user.sub)}/apikey`, { apiKey: newKey });
      notifications.show({ message: 'API-ключ обновлён', color: 'green' });
      setNewKey(''); setConfirmKey('');
      onSaved();
    } catch { /* handled */ } finally { setLoading(false); }
  };

  return (
    <Modal opened={opened} onClose={onClose} title={`Редактировать: ${user.username}`} size="md">
      <Stack gap="lg">
        <Stack gap="xs">
          <Text fw={600} size="sm">Slug-префикс</Text>
          <Group>
            <TextInput placeholder="u123" value={prefix} onChange={e => setPrefix(e.currentTarget.value)} style={{ flex: 1 }} />
            <Button onClick={savePrefix} loading={loading} leftSection={<IconEdit size={14} />}>{RU.save}</Button>
          </Group>
        </Stack>
        <Stack gap="xs">
          <Text fw={600} size="sm">Shlink API Key</Text>
          <Text size="xs" c="dimmed">Текущий ключ не отображается из соображений безопасности.</Text>
          <TextInput type="password" placeholder="Новый Shlink API ключ" value={newKey} onChange={e => setNewKey(e.currentTarget.value)} />
          <Group align="flex-end">
            <TextInput
              type="password"
              placeholder="Подтвердите ключ"
              value={confirmKey}
              onChange={e => setConfirmKey(e.currentTarget.value)}
              style={{ flex: 1 }}
            />
            <Button
              onClick={saveApiKey}
              loading={loading}
              disabled={!keysMatch}
              leftSection={<IconKey size={14} />}
            >
              {RU.save}
            </Button>
          </Group>
        </Stack>
      </Stack>
    </Modal>
  );
}
