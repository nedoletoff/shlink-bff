import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import {
  Stack, Group, Title, TextInput, Select, Table,
  Text, Badge, Skeleton, Pagination, Paper,
} from '@mantine/core'
import { DateInput } from '@mantine/dates'
import { IconFileText } from '@tabler/icons-react'
import { getAuditLogs } from '@/api/endpoints/adminAudit'
import { EmptyState } from '@/components/ui/EmptyState'
import { formatDateTime } from '@/utils/date'
import dayjs from 'dayjs'

const RESULT_COLORS: Record<string, string> = {
  success: 'teal',
  error: 'red',
  denied: 'orange',
}

export function AdminAuditLogs() {
  const [page, setPage] = useState(1)
  const [username, setUsername] = useState('')
  const [action, setAction] = useState<string | null>(null)
  const [result, setResult] = useState<string | null>(null)
  const [dateFrom, setDateFrom] = useState<Date | null>(null)
  const [dateTo, setDateTo] = useState<Date | null>(null)

  const { data, isLoading } = useQuery({
    queryKey: ['audit-logs', page, username, action, result, dateFrom, dateTo],
    queryFn: () =>
      getAuditLogs({
        page,
        perPage: 25,
        username: username || undefined,
        action: action || undefined,
        result: result || undefined,
        dateFrom: dateFrom ? dayjs(dateFrom).format('YYYY-MM-DD') : undefined,
        dateTo: dateTo ? dayjs(dateTo).format('YYYY-MM-DD') : undefined,
      }),
  })

  const logs = data?.items ?? []
  const total = data?.total ?? 0
  const totalPages = Math.ceil(total / 25)

  return (
    <Stack gap="md">
      <Title order={2}>Аудит-логи</Title>

      <Group gap="sm" wrap="wrap">
        <TextInput
          placeholder="Пользователь"
          value={username}
          onChange={(e) => { setUsername(e.currentTarget.value); setPage(1) }}
          w={180}
        />
        <Select
          placeholder="Действие"
          clearable
          data={['create', 'edit', 'delete', 'deactivate', 'activate', 'login', 'logout']}
          value={action}
          onChange={(v: string | null) => { setAction(v); setPage(1) }}
          w={180}
        />
        <Select
          placeholder="Результат"
          clearable
          data={['success', 'error', 'denied']}
          value={result}
          onChange={(v: string | null) => { setResult(v); setPage(1) }}
          w={150}
        />
        <DateInput
          placeholder="От"
          value={dateFrom}
          onChange={(v: Date | null) => { setDateFrom(v); setPage(1) }}
          w={140}
          clearable
        />
        <DateInput
          placeholder="До"
          value={dateTo}
          onChange={(v: Date | null) => { setDateTo(v); setPage(1) }}
          w={140}
          clearable
        />
      </Group>

      {isLoading ? (
        <Skeleton height={400} />
      ) : logs.length === 0 ? (
        <EmptyState icon={<IconFileText size={24} />} title="Записей нет" description="Попробуйте снять фильтры" />
      ) : (
        <Paper withBorder radius="md">
          <Table.ScrollContainer minWidth={700}>
            <Table striped highlightOnHover>
              <Table.Thead>
                <Table.Tr>
                  <Table.Th>Дата</Table.Th>
                  <Table.Th>Пользователь</Table.Th>
                  <Table.Th>Действие</Table.Th>
                  <Table.Th>Ресурс</Table.Th>
                  <Table.Th>Результат</Table.Th>
                  <Table.Th>IP</Table.Th>
                </Table.Tr>
              </Table.Thead>
              <Table.Tbody>
                {logs.map((log, i) => (
                  <Table.Tr key={log.id ?? i}>
                    <Table.Td><Text size="xs">{formatDateTime(log.createdAt)}</Text></Table.Td>
                    <Table.Td>
                      <Text size="sm" fw={500}>{log.username}</Text>
                      <Text size="xs" c="dimmed">{log.role}</Text>
                    </Table.Td>
                    <Table.Td><Badge variant="light" size="sm">{log.action}</Badge></Table.Td>
                    <Table.Td><Text size="xs" c="dimmed" lineClamp={1}>{log.resource}</Text></Table.Td>
                    <Table.Td>
                      <Badge color={RESULT_COLORS[log.result] ?? 'gray'} variant="light" size="sm">
                        {log.result}
                      </Badge>
                    </Table.Td>
                    <Table.Td><Text size="xs" c="dimmed">{log.ipAddress}</Text></Table.Td>
                  </Table.Tr>
                ))}
              </Table.Tbody>
            </Table>
          </Table.ScrollContainer>
        </Paper>
      )}

      {totalPages > 1 && (
        <Group justify="center">
          <Pagination total={totalPages} value={page} onChange={setPage} size="sm" />
        </Group>
      )}
    </Stack>
  )
}
