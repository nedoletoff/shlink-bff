import { useQuery } from '@tanstack/react-query'
import {
  Stack, Group, Title, Table, Text, Skeleton, TextInput,
} from '@mantine/core'
import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { IconUsers } from '@tabler/icons-react'
import { getAdminUsers } from '@/api/endpoints/adminUsers'
import { RoleBadge } from '@/components/ui/RoleBadge'
import { StatusBadge } from '@/components/ui/StatusBadge'
import { EmptyState } from '@/components/ui/EmptyState'
import { formatDate } from '@/utils/date'

export function AdminUsers() {
  const navigate = useNavigate()
  const [search, setSearch] = useState('')

  const { data: users = [], isLoading } = useQuery({
    queryKey: ['admin-users'],
    queryFn: getAdminUsers,
  })

  const filtered = users.filter(
    (u) =>
      u.username.toLowerCase().includes(search.toLowerCase()) ||
      u.email.toLowerCase().includes(search.toLowerCase()),
  )

  return (
    <Stack gap="md">
      <Group justify="space-between">
        <Title order={2}>Пользователи</Title>
        <TextInput
          placeholder="Поиск..."
          value={search}
          onChange={(e) => setSearch(e.currentTarget.value)}
          w={240}
        />
      </Group>

      {isLoading ? (
        <Skeleton height={300} />
      ) : filtered.length === 0 ? (
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
              </Table.Tr>
            </Table.Thead>
            <Table.Tbody>
              {filtered.map((u) => (
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
                  <Table.Td><Text size="sm" c="dimmed">{u.createdAt ? formatDate(u.createdAt) : '—'}</Text></Table.Td>
                </Table.Tr>
              ))}
            </Table.Tbody>
          </Table>
        </Table.ScrollContainer>
      )}
    </Stack>
  )
}
