import { useCallback, useEffect, useState } from 'react';
import {
  Stack, Title, Table, Text, Badge,
  Group, Center, Loader, Divider,
  Card, Alert,
} from '@mantine/core';
import { IconInfoCircle } from '@tabler/icons-react';
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

export function Roles() {
  const [data,    setData]    = useState<RolesResponse | null>(null);
  const [loading, setLoading] = useState(true);

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

          {/* Информационный блок: маппинги конфигурируются через env */}
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

          {/* Текущие маппинги (читаем из ответа API, если есть) */}
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
    </Stack>
  );
}
