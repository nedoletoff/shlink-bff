import { useState, useCallback } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import {
  Stack, Group, Title, Table, Text, Skeleton, TextInput,
  Pagination, Select, ActionIcon, Tooltip,
} from '@mantine/core'
import { useDebouncedValue } from '@mantine/hooks'
import { useNavigate } from 'react-router-dom'
import { IconUsers, IconBan, IconCircleCheck } from '@tabler/icons-react'
import { notifications } from '@mantine/notifications'
import { getAdminUsers, updateAdminUser } from '@/api/endpoints/adminUsers'
import { RoleBadge } from '@/components/ui/RoleBadge'
import { StatusBadge } from '@/components/ui/StatusBadge'
import { EmptyState } from '@/components/ui/EmptyState'
import { formatDate } from '@/utils/date'

const PER_PAGE = 20

export function AdminUsers() {
  const navigate = useNavigate()
  const qc = useQueryClient()
  const [search, setSearch] = useState('')
  const [roleFilter, setRoleFilter] = useState<string | null>(null)
  const [statusFilter, setStatusFilter] = useState<string | null>(null)
  const [page, setPage] = useState(1)

  // Баг #2: поиск без дебоунса перезапрашивал API на каждый символ
  const [debouncedSearch] = useDebouncedValue(search, 300)

  const { data: users = [], isLoading } = useQuery({
    queryKey: ['admin-users', debouncedSearch, roleFilter, statusFilter],
    queryFn: getAdminUsers,
  })

  const toggleStatusMutation = useMutation({
    mutationFn: ({ sub, status }: { sub: string; status: string }) =>
      updateAdminUser(sub, { status }),
    onSuccess: (_, { status }) => {
      qc.invalidateQueries({ queryKey: ['admin-users'] })
      notifications.show({
        color: status === 'banned' ? 'orange' : 'teal',
        message: status === 'banned' ? 'Пользователь заблокирован' : 'Пользователь активирован',
      })
    },
  })

  // фильтрация на фронте по роли и статусу
  const filtered = users.filter((u) => {
    if (roleFilter && u.role !== roleFilter) return false
    if (statusFilter && u.status !== statusFilter) return false
    return true
  })

  const roles = [...new Set(users.map((u) => u.role))]
  const statuses = [...new Set(users.map((u) => u.status))]

  const totalPages = Math.max(1, Math.ceil(filtered.length / PER_PAGE))
  const paginated = filtered.slice((page - 1) * PER_PAGE, page * PER_PAGE)

  const handleSearchChange = useCallback((v: string) => {
    setSearch(v); setPage(1)
  }, [])

  return (
    <Stack gap="md">
      <Group justify="space-between">
        <Title order={2}>Пользователи</Title>
        <Text size="sm" c="dimmed">{filtered.length} чел.</Text>
      </Group>

      <Group gap="sm" wrap="wrap">
        <TextInput
          placeholder="Поиск по имени / email"
          value={search}
          onChange={(e) => handleSearchChange(e.currentTarget.value)}
          w={240}
        />
        <Select
          placeholder="Роль"
          clearable
          data={roles}
          value={roleFilter}
          onChange={(v) => { setRoleFilter(v); setPage(1) }}
          w={140}
        />
        <Select
          placeholder="Статус"
          clearable
          data={statuses}
          value={statusFilter}
          onChange={(v) => { setStatusFilter(v); setPage(1) }}
          w={130}
        />
      </Group>

      {isLoading ? (
        <Stack gap="xs">
          {Array.from({ length: 8 }).map((_, i) => <Skeleton key={i} height={44} radius="sm" />)}
        </Stack>
      ) : paginated.length === 0 ? (
        <EmptyState icon={<IconUsers size={24} />} title="Пользователей нет" />
      ) : (
        <Table.ScrollContainer minWidth={600}>
          <Table striped highlightOnHover>
            <Table.Thead>
              <Table.Tr>
                <Table.Th>Пользователь</Table.Th>
                <Table.Th>Email</Table.Th>
                <Table.Th>Роль</Table.Th>
                <Table.Th>Статус</Table.Th>
                <Table.Th>Создан</Table.Th>
                <Table.Th />
              </Table.Tr>
            </Table.Thead>
            <Table.Tbody>
              {paginated.map((u) => (
                <Table.Tr
                  key={u.sub}
                  style={{ cursor: 'pointer' }}
                  onClick={() => navigate(`/admin/users/${u.sub}`)}
                >
                  <Table.Td>
                    <Text size="sm" fw={500}>{u.username}</Text>
                    <Text size="xs" c="dimmed">{u.sub.slice(0, 8)}…</Text>
                  </Table.Td>
                  <Table.Td><Text size="sm">{u.email}</Text></Table.Td>
                  <Table.Td><RoleBadge role={u.role} /></Table.Td>
                  <Table.Td><StatusBadge status={u.status} /></Table.Td>
                  <Table.Td>
                    <Text size="sm" c="dimmed">{u.createdAt ? formatDate(u.createdAt) : '—'}</Text>
                  </Table.Td>
                  <Table.Td onClick={(e) => e.stopPropagation()}>
                    {u.status === 'banned' ? (
                      <Tooltip label="Активировать">
                        <ActionIcon
                          variant="subtle" color="teal" size="sm"
                          aria-label="Активировать"
                          onClick={() => toggleStatusMutation.mutate({ sub: u.sub, status: 'active' })}
                        >
                          <IconCircleCheck size={14} />
                        </ActionIcon>
                      </Tooltip>
                    ) : (
                      <Tooltip label="Заблокировать">
                        <ActionIcon
                          variant="subtle" color="red" size="sm"
                          aria-label="Заблокировать"
                          onClick={() => toggleStatusMutation.mutate({ sub: u.sub, status: 'banned' })}
                        >
                          <IconBan size={14} />
                        </ActionIcon>
                      </Tooltip>
                    )}
                  </Table.Td>
                </Table.Tr>
              ))}
            </Table.Tbody>
          </Table>
        </Table.ScrollContainer>
      )}

      {totalPages > 1 && (
        <Group justify="center">
          <Pagination total={totalPages} value={page} onChange={setPage} size="sm" />
        </Group>
      )}
    </Stack>
  )
}
