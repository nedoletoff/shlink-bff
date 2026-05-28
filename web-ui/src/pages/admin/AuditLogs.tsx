import { useEffect, useState } from 'react';
import {
  Stack, Title, Table, Text, Badge, Group, Select,
  TextInput, Center, Loader, Pagination,
} from '@mantine/core';
import { api } from '../../api/client';
import type { AuditLog, AuditLogsResponse } from '../../types/api';

const PAGE_SIZE = 50;

const ACTION_OPTIONS = [
  { value: '',                  label: 'Все действия' },
  { value: 'create_short_url',  label: 'Создание ссылки' },
  { value: 'update_short_url',  label: 'Обновление ссылки' },
  { value: 'delete_short_url',  label: 'Удаление ссылки' },
  { value: 'list_short_urls',   label: 'Просмотр ссылок' },
  { value: 'create_tag',        label: 'Создание тега' },
  { value: 'delete_tag',        label: 'Удаление тега' },
  { value: 'rename_tag',        label: 'Переименование тега' },
  { value: 'rbac_denied',       label: 'RBAC-отказ' },
  { value: 'admin_update_user', label: 'Изменение пользователя' },
  { value: 'admin_update_apikey', label: 'Обновление API key' },
  { value: 'admin_update_prefix', label: 'Обновление префикса' },
];

const RESULT_OPTIONS = [
  { value: '',        label: 'Все результаты' },
  { value: 'success', label: 'Успех' },
  { value: 'denied',  label: 'Отказ' },
  { value: 'error',   label: 'Ошибка' },
];

export function AuditLogs() {
  const [logs,    setLogs]    = useState<AuditLog[]>([]);
  const [total,   setTotal]   = useState(0);
  const [loading, setLoading] = useState(true);
  const [page,    setPage]    = useState(1);

  const [username, setUsername] = useState('');
  const [action,   setAction]   = useState('');
  const [result,   setResult]   = useState('');
  const [dateFrom, setDateFrom] = useState('');
  const [dateTo,   setDateTo]   = useState('');

  const fetchLogs = () => {
    setLoading(true);
    api.get<AuditLogsResponse>('/api/admin/logs', {
      params: {
        page,
        limit:    PAGE_SIZE,
        username: username || undefined,
        action:   action   || undefined,
        result:   result   || undefined,
        dateFrom: dateFrom || undefined,
        dateTo:   dateTo   || undefined,
      },
    })
      .then(r => { setLogs(r.logs ?? []); setTotal(r.total); })
      .catch(() => { setLogs([]); setTotal(0); })
      .finally(() => setLoading(false));
  };

  useEffect(() => { fetchLogs(); }, [page, username, action, result, dateFrom, dateTo]);

  const resultColor = (r: string) =>
    r === 'success' ? 'green' : r === 'denied' ? 'red' : 'orange';

  return (
    <Stack gap="lg">
      <Title order={2}>Журнал аудита</Title>

      {/* Фильтры */}
      <Group wrap="wrap">
        <TextInput
          placeholder="Фильтр по username"
          value={username}
          onChange={e => { setUsername(e.currentTarget.value); setPage(1); }}
          style={{ flex: 1, minWidth: 160 }}
        />
        <Select
          data={ACTION_OPTIONS} value={action}
          onChange={v => { setAction(v ?? ''); setPage(1); }}
          style={{ minWidth: 200 }}
        />
        <Select
          data={RESULT_OPTIONS} value={result}
          onChange={v => { setResult(v ?? ''); setPage(1); }}
          style={{ minWidth: 140 }}
        />
        <TextInput
          type="date" placeholder="С"
          value={dateFrom}
          onChange={e => { setDateFrom(e.currentTarget.value); setPage(1); }}
          style={{ width: 150 }}
        />
        <TextInput
          type="date" placeholder="По"
          value={dateTo}
          onChange={e => { setDateTo(e.currentTarget.value); setPage(1); }}
          style={{ width: 150 }}
        />
      </Group>

      {loading ? (
        <Center h={200}><Loader /></Center>
      ) : (
        <>
          <Table striped highlightOnHover withTableBorder>
            <Table.Thead>
              <Table.Tr>
                <Table.Th>Время</Table.Th>
                <Table.Th>Пользователь</Table.Th>
                <Table.Th>Роль</Table.Th>
                <Table.Th>Действие</Table.Th>
                <Table.Th>Ресурс</Table.Th>
                <Table.Th>Результат</Table.Th>
                <Table.Th>IP</Table.Th>
              </Table.Tr>
            </Table.Thead>
            <Table.Tbody>
              {logs.map(log => (
                <Table.Tr key={log.id}>
                  <Table.Td>
                    <Text size="xs" ff="monospace">
                      {new Date(log.createdAt).toLocaleString('ru')}
                    </Text>
                  </Table.Td>
                  <Table.Td>
                    <Text size="sm">{log.username}</Text>
                    <Text size="xs" c="dimmed" ff="monospace">{log.userSub?.slice(0, 8)}…</Text>
                  </Table.Td>
                  <Table.Td>
                    <Badge size="sm" color={log.role === 'admin' ? 'red' : 'blue'} variant="light">
                      {log.role}
                    </Badge>
                  </Table.Td>
                  <Table.Td>
                    <Text size="sm" ff="monospace">{log.action}</Text>
                  </Table.Td>
                  <Table.Td>
                    <Text size="sm" c="dimmed" truncate="end" maw={200}>{log.resource}</Text>
                  </Table.Td>
                  <Table.Td>
                    <Badge size="sm" color={resultColor(log.result)} variant="light">
                      {log.result}
                    </Badge>
                  </Table.Td>
                  <Table.Td>
                    <Text size="xs" ff="monospace" c="dimmed">{log.ipAddress}</Text>
                  </Table.Td>
                </Table.Tr>
              ))}
              {logs.length === 0 && (
                <Table.Tr>
                  <Table.Td colSpan={7}>
                    <Center p="xl"><Text c="dimmed">Записей не найдено</Text></Center>
                  </Table.Td>
                </Table.Tr>
              )}
            </Table.Tbody>
          </Table>

          <Pagination
            total={Math.max(1, Math.ceil(total / PAGE_SIZE))}
            value={page}
            onChange={setPage}
            siblings={1}
          />
        </>
      )}
    </Stack>
  );
}
