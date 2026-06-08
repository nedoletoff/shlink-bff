import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import {
  Stack, Group, Title, SegmentedControl, SimpleGrid,
  Paper, Text, Skeleton, Table, Badge, Anchor,
  Box,
} from '@mantine/core'
import {
  ResponsiveContainer, LineChart, Line,
  XAxis, YAxis, Tooltip as RTooltip, CartesianGrid,
  BarChart, Bar, PieChart, Pie, Cell,
} from 'recharts'
import {
  IconLink, IconEye, IconUsers, IconTag,
} from '@tabler/icons-react'
import { useNavigate } from 'react-router-dom'
import { getDashboard } from '@/api/endpoints/dashboard'
import { StatCard } from '@/components/ui/StatCard'
import { EmptyState } from '@/components/ui/EmptyState'
import { formatDate } from '@/utils/date'
import { useAuth } from '@/hooks/useAuth'

const COLORS = ['#0e9488', '#3b82f6', '#f59e0b', '#8b5cf6', '#ec4899']
const WEEKDAYS = ['Вс', 'Пн', 'Вт', 'Ср', 'Чт', 'Пт', 'Сб']
const PERIODS = [
  { label: '7 дней', value: '7' },
  { label: '30 дней', value: '30' },
  { label: '90 дней', value: '90' },
]

const USER_STATUS_COLOR: Record<string, string> = {
  active: 'teal', inactive: 'gray', banned: 'red',
}

export function Dashboard() {
  const [period, setPeriod] = useState('7')
  const navigate = useNavigate()
  const { isAdmin } = useAuth()

  const { data, isLoading } = useQuery({
    queryKey: ['dashboard', period],
    queryFn: () => getDashboard(Number(period)),
  })

  const clicksData = (data?.visits.clicksPerDay ?? []).map((p) => ({
    ...p,
    date: formatDate(p.date),
  }))
  const browsersData = data?.devices.browsers.slice(0, 5) ?? []
  const osData = data?.devices.os.slice(0, 5) ?? []
  const heatmap = data?.devices.heatmap ?? []
  const topLinks = data?.overview.topLinks ?? []
  const recentLinks = data?.overview.recentLinks ?? []
  const users = data?.users ?? []
  const tags = data?.tags ?? []

  // build heatmap grid: 7 weekdays × 24 hours
  const heatMax = Math.max(1, ...heatmap.map((c) => c.value))
  const heatGrid: number[][] = Array.from({ length: 7 }, () => Array(24).fill(0))
  heatmap.forEach(({ weekday, hour, value }) => { heatGrid[weekday][hour] = value })

  return (
    <Stack gap="lg">
      <Group justify="space-between" align="center">
        <Title order={2}>Дашборд</Title>
        <SegmentedControl size="xs" data={PERIODS} value={period} onChange={setPeriod} />
      </Group>

      {/* KPI */}
      <SimpleGrid cols={{ base: 2, sm: 4 }} spacing="md">
        <Skeleton visible={isLoading} radius="md">
          <StatCard label="Всего ссылок" value={data?.overview.linksCount ?? 0} icon={<IconLink size={18} />} />
        </Skeleton>
        <Skeleton visible={isLoading} radius="md">
          <StatCard label="Переходов" value={data?.visits.clicksTotal ?? 0} icon={<IconEye size={18} />} />
        </Skeleton>
        <Skeleton visible={isLoading} radius="md">
          <StatCard label="Тегов" value={tags.length} icon={<IconTag size={18} />} />
        </Skeleton>
        <Skeleton visible={isLoading} radius="md">
          <StatCard label="Пользователей" value={users.length} icon={<IconUsers size={18} />} />
        </Skeleton>
      </SimpleGrid>

      {/* Clicks chart */}
      <Paper withBorder p="md" radius="md">
        <Text fw={600} mb="md">Переходы по дням</Text>
        {isLoading ? <Skeleton height={220} /> : clicksData.length === 0 ? (
          <EmptyState icon={<IconEye size={24} />} title="Нет данных за период" />
        ) : (
          <ResponsiveContainer width="100%" height={220}>
            <LineChart data={clicksData}>
              <CartesianGrid strokeDasharray="3 3" stroke="var(--mantine-color-gray-2)" />
              <XAxis dataKey="date" tick={{ fontSize: 11 }} />
              <YAxis tick={{ fontSize: 11 }} />
              <RTooltip />
              <Line type="monotone" dataKey="clicks" stroke="#0e9488" strokeWidth={2} dot={false} />
            </LineChart>
          </ResponsiveContainer>
        )}
      </Paper>

      {/* Heatmap */}
      {heatmap.length > 0 && (
        <Paper withBorder p="md" radius="md">
          <Text fw={600} mb="sm">Активность по часам</Text>
          <Box style={{ overflowX: 'auto' }}>
            <Box style={{ display: 'grid', gridTemplateColumns: '36px repeat(24, 1fr)', gap: 2, minWidth: 600 }}>
              {/* header */}
              <Box />
              {Array.from({ length: 24 }, (_, h) => (
                <Text key={h} size="xs" c="dimmed" ta="center">{h}</Text>
              ))}
              {/* rows */}
              {WEEKDAYS.map((day, wi) => (
                <>
                  <Text key={`d${wi}`} size="xs" c="dimmed" style={{ lineHeight: '16px' }}>{day}</Text>
                  {heatGrid[wi].map((val, hi) => (
                    <Box
                      key={hi}
                      style={{
                        height: 16,
                        borderRadius: 3,
                        background: val === 0
                          ? 'var(--mantine-color-gray-1)'
                          : `rgba(14,148,136,${0.15 + 0.85 * (val / heatMax)})`,
                      }}
                      title={`${day} ${hi}:00 — ${val}`}
                    />
                  ))}
                </>
              ))}
            </Box>
          </Box>
        </Paper>
      )}

      {/* Devices / OS */}
      <SimpleGrid cols={{ base: 1, md: 2 }} spacing="md">
        <Paper withBorder p="md" radius="md">
          <Text fw={600} mb="md">Браузеры</Text>
          {isLoading ? <Skeleton height={180} /> : (
            <ResponsiveContainer width="100%" height={180}>
              <PieChart>
                <Pie data={browsersData} dataKey="count" nameKey="name" cx="50%" cy="50%" outerRadius={70} label={(e) => e.name}>
                  {browsersData.map((_, i) => <Cell key={i} fill={COLORS[i % COLORS.length]} />)}
                </Pie>
                <RTooltip />
              </PieChart>
            </ResponsiveContainer>
          )}
        </Paper>
        <Paper withBorder p="md" radius="md">
          <Text fw={600} mb="md">ОС</Text>
          {isLoading ? <Skeleton height={180} /> : (
            <ResponsiveContainer width="100%" height={180}>
              <BarChart data={osData} layout="vertical">
                <CartesianGrid strokeDasharray="3 3" />
                <XAxis type="number" tick={{ fontSize: 11 }} />
                <YAxis type="category" dataKey="name" tick={{ fontSize: 11 }} width={60} />
                <RTooltip />
                <Bar dataKey="count" fill="#3b82f6" radius={[0, 3, 3, 0]} />
              </BarChart>
            </ResponsiveContainer>
          )}
        </Paper>
      </SimpleGrid>

      {/* Top links */}
      {(isLoading || topLinks.length > 0) && (
        <Paper withBorder p="md" radius="md">
          <Text fw={600} mb="md">Топ-5 по переходам</Text>
          {isLoading ? <Skeleton height={160} /> : (
            <Table highlightOnHover>
              <Table.Thead>
                <Table.Tr>
                  <Table.Th>Ссылка</Table.Th>
                  <Table.Th>Назначение</Table.Th>
                  <Table.Th style={{ textAlign: 'right' }}>Переходы</Table.Th>
                </Table.Tr>
              </Table.Thead>
              <Table.Tbody>
                {topLinks.map((l) => (
                  <Table.Tr key={l.shortCode}>
                    <Table.Td>
                      <Anchor size="sm" onClick={() => navigate(`/links/${l.shortCode}`)}>{l.shortCode}</Anchor>
                    </Table.Td>
                    <Table.Td><Text size="sm" lineClamp={1}>{l.title || l.longUrl}</Text></Table.Td>
                    <Table.Td style={{ textAlign: 'right' }}>
                      <Text size="sm" fw={600} style={{ fontVariantNumeric: 'tabular-nums' }}>
                        {l.visitsTotal.toLocaleString('ru')}
                      </Text>
                    </Table.Td>
                  </Table.Tr>
                ))}
              </Table.Tbody>
            </Table>
          )}
        </Paper>
      )}

      {/* Recent links */}
      {(isLoading || recentLinks.length > 0) && (
        <Paper withBorder p="md" radius="md">
          <Text fw={600} mb="md">Последние ссылки</Text>
          {isLoading ? <Skeleton height={160} /> : (
            <Table highlightOnHover>
              <Table.Thead>
                <Table.Tr>
                  <Table.Th>Ссылка</Table.Th>
                  <Table.Th>Назначение</Table.Th>
                  <Table.Th style={{ textAlign: 'right' }}>Переходы</Table.Th>
                </Table.Tr>
              </Table.Thead>
              <Table.Tbody>
                {recentLinks.map((l) => (
                  <Table.Tr key={l.shortCode}>
                    <Table.Td>
                      <Anchor size="sm" onClick={() => navigate(`/links/${l.shortCode}`)}>{l.shortCode}</Anchor>
                    </Table.Td>
                    <Table.Td><Text size="sm" lineClamp={1}>{l.title || l.longUrl}</Text></Table.Td>
                    <Table.Td style={{ textAlign: 'right' }}>
                      <Text size="sm" style={{ fontVariantNumeric: 'tabular-nums' }}>
                        {l.visitsTotal.toLocaleString('ru')}
                      </Text>
                    </Table.Td>
                  </Table.Tr>
                ))}
              </Table.Tbody>
            </Table>
          )}
        </Paper>
      )}

      {/* Users (admin only) */}
      {isAdmin() && (isLoading || users.length > 0) && (
        <Paper withBorder p="md" radius="md">
          <Text fw={600} mb="md">Пользователи</Text>
          {isLoading ? <Skeleton height={140} /> : (
            <Table highlightOnHover>
              <Table.Thead>
                <Table.Tr>
                  <Table.Th>Имя</Table.Th>
                  <Table.Th>Email</Table.Th>
                  <Table.Th>Роль</Table.Th>
                  <Table.Th>Статус</Table.Th>
                </Table.Tr>
              </Table.Thead>
              <Table.Tbody>
                {users.map((u) => (
                  <Table.Tr key={u.sub}>
                    <Table.Td><Text size="sm" fw={500}>{u.username}</Text></Table.Td>
                    <Table.Td><Text size="sm" c="dimmed">{u.email}</Text></Table.Td>
                    <Table.Td><Badge size="xs" variant="outline">{u.role}</Badge></Table.Td>
                    <Table.Td>
                      <Badge size="xs" color={USER_STATUS_COLOR[u.status] ?? 'gray'} variant="dot">
                        {u.status}
                      </Badge>
                    </Table.Td>
                  </Table.Tr>
                ))}
              </Table.Tbody>
            </Table>
          )}
        </Paper>
      )}
    </Stack>
  )
}
