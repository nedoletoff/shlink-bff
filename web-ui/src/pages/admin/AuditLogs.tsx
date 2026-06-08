import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import {
  Stack, Group, Title, TextInput, Select, Table,
  Text, Badge, Skeleton, Pagination, Paper,
  Checkbox, ActionIcon, Tooltip, Collapse, Box,
  Button, Code,
} from '@mantine/core'
import { DateInput } from '@mantine/dates'
import { IconFileText, IconTrash, IconChevronDown, IconChevronRight } from '@tabler/icons-react'
import { notifications } from '@mantine/notifications'
import { getAuditLogs, deleteAuditLogs } from '@/api/endpoints/adminAudit'
import { EmptyState } from '@/components/ui/EmptyState'
import { ConfirmDialog } from '@/components/ui/ConfirmDialog'
import { formatDateTime } from '@/utils/date'
import dayjs from 'dayjs'

const RESULT_COLORS: Record<string, string> = {
  success: 'teal', error: 'red', denied: 'orange',
}

export function AdminAuditLogs() {
  const qc = useQueryClient()
  const [page, setPage] = useState(1)
  const [username, setUsername] = useState('')
  const [action, setAction] = useState<string | null>(null)
  const [result, setResult] = useState<string | null>(null)
  const [dateFrom, setDateFrom] = useState<string | null>(null)
  const [dateTo, setDateTo] = useState<string | null>(null)
  const [selected, setSelected] = useState<Set<number>>(new Set())
  const [expanded, setExpanded] = useState<Set<number>>(new Set())
  const [confirmDelete, setConfirmDelete] = useState(false)

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

  const deleteMutation = useMutation({
    mutationFn: () => deleteAuditLogs(Array.from(selected)),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['audit-logs'] })
      notifications.show({ color: 'teal', message: `Удалено записей: ${selected.size}` })
      setSelected(new Set())
      setConfirmDelete(false)
    },
  })

  const logs = data?.items ?? []
  const total = data?.total ?? 0
  const totalPages = Math.ceil(total / 25)

  const allIds = logs.map((l) => l.id!).filter(Boolean)
  const allSelected = allIds.length > 0 && allIds.every((id) => selected.has(id))
  const someSelected = allIds.some((id) => selected.has(id))

  const toggleAll = () => {
    if (allSelected) {
      setSelected((s) => { const n = new Set(s); allIds.forEach((id) => n.delete(id)); return n })
    } else {
      setSelected((s) => { const n = new Set(s); allIds.forEach((id) => n.add(id)); return n })
    }
  }

  const toggleOne = (id: number) =>
    setSelected((s) => { const n = new Set(s); n.has(id) ? n.delete(id) : n.add(id); return n })

  const toggleExpand = (id: number) =>
    setExpanded((s) => { const n = new Set(s); n.has(id) ? n.delete(id) : n.add(id); return n })

  return (
    <Stack gap="md">
      <Group justify="space-between">
        <Title order={2}>Аудит-логи</Title>
        {selected.size > 0 && (
          <Tooltip label={`Удалить выбранные (${selected.size})`}>
            <ActionIcon color="red" variant="light" size="lg" onClick={() => setConfirmDelete(true)} aria-label="Удалить выбранные">
              <IconTrash size={16} />
            </ActionIcon>
          </Tooltip>
        )}
      </Group>

      <Group gap="sm" wrap="wrap">
        <TextInput
          placeholder="Пользователь"
          value={username}
          onChange={(e) => { setUsername(e.currentTarget.value); setPage(1) }}
          w={160}
        />
        <Select
          placeholder="Действие"
          clearable
          data={['create', 'edit', 'delete', 'deactivate', 'activate', 'login', 'logout']}
          value={action}
          onChange={(v: string | null) => { setAction(v); setPage(1) }}
          w={160}
        />
        <Select
          placeholder="Результат"
          clearable
          data={['success', 'error', 'denied']}
          value={result}
          onChange={(v: string | null) => { setResult(v); setPage(1) }}
          w={130}
        />
        <DateInput
          placeholder="От"
          value={dateFrom}
          onChange={(v: string | null) => { setDateFrom(v); setPage(1) }}
          w={130}
          clearable
        />
        <DateInput
          placeholder="До"
          value={dateTo}
          onChange={(v: string | null) => { setDateTo(v); setPage(1) }}
          w={130}
          clearable
        />
        {selected.size > 0 && (
          <Button
            size="xs"
            variant="subtle"
            color="dimmed"
            onClick={() => setSelected(new Set())}
          >
            Снять выбор ({selected.size})
          </Button>
        )}
      </Group>

      {isLoading ? (
        <Stack gap="xs">
          {Array.from({ length: 8 }).map((_, i) => <Skeleton key={i} height={44} radius="sm" />)}
        </Stack>
      ) : logs.length === 0 ? (
        <EmptyState icon={<IconFileText size={24} />} title="Записей нет" description="Попробуйте снять фильтры" />
      ) : (
        <Paper withBorder radius="md">
          <Table.ScrollContainer minWidth={800}>
            <Table striped highlightOnHover>
              <Table.Thead>
                <Table.Tr>
                  <Table.Th w={36}>
                    <Checkbox
                      checked={allSelected}
                      indeterminate={someSelected && !allSelected}
                      onChange={toggleAll}
                      aria-label="Выбрать все"
                    />
                  </Table.Th>
                  <Table.Th w={24} />
                  <Table.Th>Дата</Table.Th>
                  <Table.Th>Пользователь</Table.Th>
                  <Table.Th>Действие</Table.Th>
                  <Table.Th>Ресурс</Table.Th>
                  <Table.Th>Результат</Table.Th>
                  <Table.Th>IP</Table.Th>
                </Table.Tr>
              </Table.Thead>
              <Table.Tbody>
                {logs.map((log, i) => {
                  const id = log.id ?? i
                  const isExpanded = expanded.has(id)
                  const isSelected = typeof log.id === 'number' && selected.has(log.id)
                  const hasDetails = log.details || log.userAgent

                  return (
                    <>
                      <Table.Tr
                        key={`row-${id}`}
                        bg={isSelected ? 'var(--mantine-color-red-0)' : undefined}
                      >
                        <Table.Td>
                          {typeof log.id === 'number' && (
                            <Checkbox
                              checked={isSelected}
                              onChange={() => toggleOne(log.id!)}
                              aria-label="Выбрать"
                            />
                          )}
                        </Table.Td>
                        <Table.Td>
                          {hasDetails && (
                            <ActionIcon
                              variant="subtle" size="xs"
                              onClick={() => toggleExpand(id)}
                              aria-label="Развернуть"
                            >
                              {isExpanded ? <IconChevronDown size={12} /> : <IconChevronRight size={12} />}
                            </ActionIcon>
                          )}
                        </Table.Td>
                        <Table.Td><Text size="xs">{formatDateTime(log.createdAt)}</Text></Table.Td>
                        <Table.Td>
                          <Text size="sm" fw={500}>{log.username}</Text>
                          <Text size="xs" c="dimmed">{log.role}</Text>
                        </Table.Td>
                        <Table.Td>
                          <Badge variant="light" size="sm">{log.action}</Badge>
                        </Table.Td>
                        <Table.Td>
                          <Text size="xs" c="dimmed" lineClamp={1} maw={200}>{log.resource}</Text>
                        </Table.Td>
                        <Table.Td>
                          <Badge color={RESULT_COLORS[log.result] ?? 'gray'} variant="light" size="sm">
                            {log.result}
                          </Badge>
                        </Table.Td>
                        <Table.Td><Text size="xs" c="dimmed">{log.ipAddress}</Text></Table.Td>
                      </Table.Tr>

                      {hasDetails && (
                        <Table.Tr key={`exp-${id}`}>
                          <Table.Td colSpan={8} p={0}>
                            <Collapse in={isExpanded}>
                              <Box p="md" bg="var(--mantine-color-gray-0)">
                                {log.userAgent && (
                                  <Text size="xs" c="dimmed" mb="xs">
                                    <b>User-Agent:</b> {log.userAgent}
                                  </Text>
                                )}
                                {log.details && (
                                  <>
                                    <Text size="xs" c="dimmed" mb={4}><b>Details:</b></Text>
                                    <Code block style={{ fontSize: 11 }}>
                                      {JSON.stringify(log.details, null, 2)}
                                    </Code>
                                  </>
                                )}
                              </Box>
                            </Collapse>
                          </Table.Td>
                        </Table.Tr>
                      )}
                    </>
                  )
                })}
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

      <ConfirmDialog
        opened={confirmDelete}
        onClose={() => setConfirmDelete(false)}
        onConfirm={() => deleteMutation.mutate()}
        message={`Удалить ${selected.size} записей? Действие необратимо.`}
        loading={deleteMutation.isPending}
      />
    </Stack>
  )
}
