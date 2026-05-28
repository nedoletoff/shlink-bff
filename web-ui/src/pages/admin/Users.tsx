import { useEffect, useState } from 'react';
import {
  Stack, Title, Table, Text, Badge, ActionIcon,
  Center, Loader, Modal, TextInput, Group, Button,
} from '@mantine/core';
import { useDisclosure } from '@mantine/hooks';
import { notifications } from '@mantine/notifications';
import { IconEdit, IconKey } from '@tabler/icons-react';
import { api } from '../../api/client';
import type { AdminUser } from '../../types/api';

export function AdminUsers() {
  const [users,   setUsers]   = useState<AdminUser[]>([]);
  const [loading, setLoading] = useState(true);
  const [editUser, setEditUser] = useState<AdminUser | null>(null);
  const [editOpen, { open: openEdit, close: closeEdit }] = useDisclosure(false);

  const fetchUsers = () => {
    api.get<AdminUser[]>('/api/admin/users')
      .then(setUsers)
      .catch(() => setUsers([]))
      .finally(() => setLoading(false));
  };

  useEffect(() => { fetchUsers(); }, []);

  const handleEdit = (u: AdminUser) => { setEditUser(u); openEdit(); };

  return (
    <Stack gap="lg">
      <Title order={2}>Управление пользователями</Title>

      {loading ? (
        <Center h={200}><Loader /></Center>
      ) : (
        <Table striped highlightOnHover withTableBorder>
          <Table.Thead>
            <Table.Tr>
              <Table.Th>Пользователь</Table.Th>
              <Table.Th>Email</Table.Th>
              <Table.Th>Роль</Table.Th>
              <Table.Th>Префикс</Table.Th>
              <Table.Th>Статус</Table.Th>
              <Table.Th>API Key</Table.Th>
              <Table.Th />
            </Table.Tr>
          </Table.Thead>
          <Table.Tbody>
            {users.map(u => (
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
                  <ActionIcon variant="subtle" onClick={() => handleEdit(u)}>
                    <IconEdit size={16} />
                  </ActionIcon>
                </Table.Td>
              </Table.Tr>
            ))}
            {users.length === 0 && (
              <Table.Tr>
                <Table.Td colSpan={7}>
                  <Center p="xl"><Text c="dimmed">Пользователи не найдены</Text></Center>
                </Table.Td>
              </Table.Tr>
            )}
          </Table.Tbody>
        </Table>
      )}

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
  const [prefix,  setPrefix]  = useState('');
  const [newKey,  setNewKey]  = useState('');
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    if (user) {
      setPrefix(user.slugPrefix ?? '');
      setNewKey('');
    }
  }, [user]);

  if (!user) return null;

  const savePrefix = async () => {
    setLoading(true);
    try {
      await api.put(`/api/admin/users/${encodeURIComponent(user.sub)}/prefix`, { prefix });
      notifications.show({ message: 'Префикс обновлён', color: 'green' });
      onSaved();
    } catch {
      // handled
    } finally {
      setLoading(false);
    }
  };

  const saveApiKey = async () => {
    if (!newKey.trim()) return;
    setLoading(true);
    try {
      await api.put(`/api/admin/users/${encodeURIComponent(user.sub)}/apikey`, { apiKey: newKey });
      notifications.show({ message: 'API-ключ обновлён', color: 'green' });
      setNewKey('');
      onSaved();
    } catch {
      // handled
    } finally {
      setLoading(false);
    }
  };

  return (
    <Modal
      opened={opened} onClose={onClose}
      title={`Редактировать: ${user.username}`} size="md"
    >
      <Stack gap="lg">
        {/* Slug-префикс */}
        <Stack gap="xs">
          <Text fw={600} size="sm">Slug-префикс</Text>
          <Group>
            <TextInput
              placeholder="u123"
              value={prefix}
              onChange={e => setPrefix(e.currentTarget.value)}
              style={{ flex: 1 }}
            />
            <Button
              onClick={savePrefix} loading={loading}
              leftSection={<IconEdit size={14} />}
            >
              Сохранить
            </Button>
          </Group>
        </Stack>

        {/* Обновление API key — реальный ключ не отображается */}
        <Stack gap="xs">
          <Text fw={600} size="sm">Shlink API Key</Text>
          <Text size="xs" c="dimmed">
            Текущий ключ не отображается из соображений безопасности.
            Введите новый значение чтобы обновить.
          </Text>
          <Group>
            <TextInput
              type="password"
              placeholder="Новый Shlink API ключ"
              value={newKey}
              onChange={e => setNewKey(e.currentTarget.value)}
              style={{ flex: 1 }}
            />
            <Button
              color="orange"
              onClick={saveApiKey}
              loading={loading}
              disabled={!newKey.trim()}
              leftSection={<IconKey size={14} />}
            >
              Обновить
            </Button>
          </Group>
        </Stack>
      </Stack>
    </Modal>
  );
}
