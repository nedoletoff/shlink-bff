import { useCallback, useEffect, useState } from 'react';
import {
  Stack, Title, Table, Text, Badge, Button, Modal,
  Select, TextInput, Group, Center, Loader, Divider,
  Card,
} from '@mantine/core';
import { useDisclosure } from '@mantine/hooks';
import { notifications } from '@mantine/notifications';
import { IconPlus } from '@tabler/icons-react';
import { api } from '../../api/client';
import { RU } from '../../i18n/ru';

interface RoleEntry {
  role:        string;
  permissions: string[];
  usersCount:  number;
}

interface RoleMapping {
  kcGroup: string;
  appRole: string;
}

interface RolesResponse {
  roles:    RoleEntry[];
  mappings: RoleMapping[];
}

const AVAILABLE_ROLES = [
  { value: 'admin', label: 'admin' },
  { value: 'user',  label: 'user'  },
];

export function Roles() {
  const [data,    setData]    = useState<RolesResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [addOpen, { open: openAdd, close: closeAdd }] = useDisclosure(false);

  const [kcGroup,  setKcGroup]  = useState('');
  const [appRole,  setAppRole]  = useState<string | null>('user');
  const [saving,   setSaving]   = useState(false);

  const fetchRoles = useCallback(() => {
    setLoading(true);
    api.get<RolesResponse>('/api/admin/roles')
      .then(setData)
      .catch(() => setData(null))
      .finally(() => setLoading(false));
  }, []);

  useEffect(() => { fetchRoles(); }, [fetchRoles]);

  const handleAddMapping = async () => {
    if (!kcGroup.trim() || !appRole) return;
    setSaving(true);
    try {
      await api.post('/api/admin/roles/mappings', {
        kcGroup: kcGroup.trim(),
        appRole,
      });
      notifications.show({ message: 'Маппинг добавлен', color: 'green' });
      closeAdd();
      setKcGroup('');
      setAppRole('user');
      fetchRoles();
    } catch {
      // handled
    } finally {
      setSaving(false);
    }
  };

  return (
    <Stack gap="lg">
      <Group justify="space-between">
        <Title order={2}>{RU.roles.title}</Title>
        <Button leftSection={<IconPlus size={16} />} onClick={openAdd}>
          {RU.roles.addMapping}
        </Button>
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
                  <Table.Th>{RU.roles.usersCount}</Table.Th>
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
                      <Text size="sm">{r.usersCount}</Text>
                    </Table.Td>
                  </Table.Tr>
                ))}
              </Table.Tbody>
            </Table>
          </Card>

          <Divider label="Маппинг групп Keycloak → роль" labelPosition="left" />

          {/* Текущие маппинги */}
          <Card withBorder radius="md" p={0}>
            <Table striped highlightOnHover>
              <Table.Thead>
                <Table.Tr>
                  <Table.Th>{RU.roles.kcGroup}</Table.Th>
                  <Table.Th>{RU.roles.appRole}</Table.Th>
                </Table.Tr>
              </Table.Thead>
              <Table.Tbody>
                {data.mappings.length === 0 ? (
                  <Table.Tr>
                    <Table.Td colSpan={2}>
                      <Center p="md"><Text c="dimmed">Маппинги не настроены</Text></Center>
                    </Table.Td>
                  </Table.Tr>
                ) : data.mappings.map(m => (
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
        </Stack>
      )}

      {/* Модалка добавления маппинга */}
      <Modal opened={addOpen} onClose={closeAdd} title={RU.roles.addMapping} size="sm">
        <Stack gap="sm">
          <TextInput
            label={RU.roles.kcGroup}
            placeholder="shlink-admins"
            value={kcGroup}
            onChange={e => setKcGroup(e.currentTarget.value)}
          />
          <Select
            label={RU.roles.appRole}
            data={AVAILABLE_ROLES}
            value={appRole}
            onChange={setAppRole}
          />
          <Group justify="flex-end" mt="md">
            <Button variant="default" onClick={closeAdd}>{RU.cancel}</Button>
            <Button
              onClick={handleAddMapping}
              loading={saving}
              disabled={!kcGroup.trim() || !appRole}
            >
              {RU.create}
            </Button>
          </Group>
        </Stack>
      </Modal>
    </Stack>
  );
}
