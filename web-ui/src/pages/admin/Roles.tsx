import { useCallback, useEffect, useState } from 'react';
import {
  Stack, Title, Table, Text, Badge,
  Group, Center, Loader, Divider,
  Card, Alert, Button, Modal,
  SimpleGrid, Switch, Tooltip,
} from '@mantine/core';
import { notifications } from '@mantine/notifications';
import { IconInfoCircle, IconEdit } from '@tabler/icons-react';
import { api } from '../../api/client';
import { RU } from '../../i18n/ru';
import type { RolePermissions, RoleEntry, RoleMapping } from '../../types/api';

interface RolesResponse {
  roles:    RoleEntry[];
  mappings: RoleMapping[];
}

// ─── Permission labels ────────────────────────────────────────────────────────
const PERM_LABELS: Record<keyof Omit<RolePermissions, 'role' | 'updatedAt'>, string> = {
  canViewOwnLinks:              'Просмотр своих ссылок',
  canViewAllLinks:              'Просмотр всех ссылок',
  canCreateLinks:               'Создание ссылок',
  canCreateWithCustomSlug:      'Кастомный слаг при создании',
  canCreateWithoutSlug:         'Создание без явного слага',
  canEditOwnLinks:              'Редактирование своих ссылок',
  canEditAllLinks:              'Редактирование всех ссылок',
  canDeleteOwnLinks:            'Удаление своих ссылок',
  canDeleteAllLinks:            'Удаление всех ссылок',
  canDeactivateOwnLinks:        'Деактивация своих ссылок',
  canDeactivateAllLinks:        'Деактивация всех ссылок',
  canReactivateOwnLinks:        'Активация своих ссылок',
  canReactivateAllLinks:        'Активация всех ссылок',
  canDeleteOwnLinksPermanently: 'Безвозвратное удаление своих ссылок',
  canDeleteAllLinksPermanently: 'Безвозвратное удаление всех ссылок',
  canManageOwnTags:             'Управление своими тегами',
  canManageAllTags:             'Управление всеми тегами',
  canViewOwnStats:              'Просмотр своей статистики',
  canViewAllStats:              'Просмотр всей статистики',
  canViewAuditLogs:             'Просмотр журнала аудита',
  canManageUsers:               'Управление пользователями',
  canManageRoles:               'Управление ролями',
};

const PERM_KEYS = Object.keys(PERM_LABELS) as (keyof typeof PERM_LABELS)[];

// ─── Edit modal ───────────────────────────────────────────────────────────────
function EditRoleModal({
  role, opened, onClose, onSaved,
}: {
  role:    string;
  opened:  boolean;
  onClose: () => void;
  onSaved: () => void;
}) {
  const [perms,   setPerms]   = useState<RolePermissions | null>(null);
  const [loading, setLoading] = useState(false);
  const [saving,  setSaving]  = useState(false);

  useEffect(() => {
    if (!opened) return;
    setLoading(true);
    api.get<RolePermissions>(`/api/admin/roles/${role}`)
      .then(setPerms)
      .catch(() => notifications.show({ message: RU.error, color: 'red' }))
      .finally(() => setLoading(false));
  }, [role, opened]);

  const toggle = (key: keyof typeof PERM_LABELS) => {
    if (!perms) return;
    setPerms({ ...perms, [key]: !perms[key] });
  };

  const handleSave = async () => {
    if (!perms) return;
    setSaving(true);
    try {
      await api.put(`/api/admin/roles/${role}/permissions`, perms);
      notifications.show({ message: RU.roles.saved, color: 'green' });
      onSaved();
      onClose();
    } catch {
      /* shown via interceptor */
    } finally {
      setSaving(false);
    }
  };

  return (
    <Modal
      opened={opened}
      onClose={onClose}
      title={`${RU.roles.editTitle}: ${role}`}
      size="lg"
    >
      {loading || !perms ? (
        <Center h={200}><Loader /></Center>
      ) : (
        <Stack gap="md">
          <SimpleGrid cols={2} spacing="xs">
            {PERM_KEYS.map(key => (
              <Switch
                key={key}
                label={PERM_LABELS[key]}
                checked={!!perms[key]}
                onChange={() => toggle(key)}
                size="sm"
              />
            ))}
          </SimpleGrid>
          <Group justify="flex-end" mt="sm">
            <Button variant="default" onClick={onClose}>{RU.cancel}</Button>
            <Button onClick={handleSave} loading={saving}>{RU.save}</Button>
          </Group>
        </Stack>
      )}
    </Modal>
  );
}

// ─── Main page ────────────────────────────────────────────────────────────────
export function Roles() {
  const [data,        setData]        = useState<RolesResponse | null>(null);
  const [loading,     setLoading]     = useState(true);
  const [editingRole, setEditingRole] = useState<string | null>(null);

  const fetchRoles = useCallback(() => {
    setLoading(true);
    api.get<RolesResponse>('/api/admin/roles')
      .then(setData)
      .catch(() => setData(null))
      .finally(() => setLoading(false));
  }, []);

  useEffect(() => { fetchRoles(); }, [fetchRoles]);

  return (
    <Stack gap="lg">
      <Group justify="space-between">
        <Title order={2}>{RU.roles.title}</Title>
      </Group>

      {loading ? (
        <Center h={200}><Loader /></Center>
      ) : !data ? (
        <Text c="dimmed">{RU.roles.noRoles}</Text>
      ) : (
        <Stack gap="xl">
          {/* Таблица ролей */}
          <Card withBorder radius="md" p={0}>
            <Table striped highlightOnHover>
              <Table.Thead>
                <Table.Tr>
                  <Table.Th>{RU.roles.role}</Table.Th>
                  <Table.Th>{RU.roles.permissions}</Table.Th>
                  <Table.Th style={{ width: 60 }} />
                </Table.Tr>
              </Table.Thead>
              <Table.Tbody>
                {data.roles.map(r => (
                  <Table.Tr key={r.role}>
                    <Table.Td>
                      <Badge color={r.role === 'admin' ? 'red' : 'blue'} variant="light">
                        {r.role}
                      </Badge>
                    </Table.Td>
                    <Table.Td>
                      <Group gap={4} wrap="wrap">
                        {r.permissions.map(p => (
                          <Badge key={p} size="xs" variant="outline" color="gray">{p}</Badge>
                        ))}
                      </Group>
                    </Table.Td>
                    <Table.Td>
                      <Tooltip label={RU.edit}>
                        <Button
                          variant="subtle" size="xs" color="blue"
                          leftSection={<IconEdit size={14} />}
                          onClick={() => setEditingRole(r.role)}
                        >
                          {RU.edit}
                        </Button>
                      </Tooltip>
                    </Table.Td>
                  </Table.Tr>
                ))}
              </Table.Tbody>
            </Table>
          </Card>

          <Divider label="Маппинг групп Keycloak → роль" labelPosition="left" />

          <Alert
            icon={<IconInfoCircle size={16} />}
            color="blue"
            variant="light"
            title="Как настроить маппинги?"
          >
            Маппинги групп Keycloak → роль управляются через переменную окружения{' '}
            <Text component="span" ff="monospace" size="sm" fw={600}>ROLE_GROUPS</Text>{' '}
            в конфигурации сервера. Пример:{' '}
            <Text component="span" ff="monospace" size="sm">ROLE_GROUPS=shlink-admins:admin,shlink-users:user</Text>
          </Alert>

          {data.mappings.length > 0 && (
            <Card withBorder radius="md" p={0}>
              <Table striped highlightOnHover>
                <Table.Thead>
                  <Table.Tr>
                    <Table.Th>{RU.roles.kcGroup}</Table.Th>
                    <Table.Th>{RU.roles.appRole}</Table.Th>
                  </Table.Tr>
                </Table.Thead>
                <Table.Tbody>
                  {data.mappings.map(m => (
                    <Table.Tr key={m.kcGroup}>
                      <Table.Td><Text ff="monospace" size="sm">{m.kcGroup}</Text></Table.Td>
                      <Table.Td>
                        <Badge color={m.appRole === 'admin' ? 'red' : 'blue'} variant="light">
                          {m.appRole}
                        </Badge>
                      </Table.Td>
                    </Table.Tr>
                  ))}
                </Table.Tbody>
              </Table>
            </Card>
          )}
        </Stack>
      )}

      {editingRole && (
        <EditRoleModal
          role={editingRole}
          opened={editingRole !== null}
          onClose={() => setEditingRole(null)}
          onSaved={fetchRoles}
        />
      )}
    </Stack>
  );
}
