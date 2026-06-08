import { useState } from 'react'
import {
  Title, Grid, SimpleGrid, SegmentedControl, Text, Skeleton,
  Table, Group, Anchor, Paper, Stack, Badge,
} from '@mantine/core'
import { LineChart, BarChart, DonutChart } from '@mantine/charts'
import { IconLink, IconEye, IconChartBar, IconUsers } from '@tabler/icons-react'
import { useQuery } from '@tanstack/react-query'
import { getDashboard } from '@/api/endpoints/dashboard'
import { StatCard } from '@/components/ui/StatCard'
import { CopyButton } from '@/components/ui/CopyButton'
import { RoleBadge } from '@/components/ui/RoleBadge'
import { StatusBadge } from '@/components/ui/StatusBadge'
import { EmptyState } from '@/components/ui/EmptyState'
import { useAuth } from '@/contexts/AuthContext'
import { formatDate } from '@/utils/date'
import type { HeatCell } from '@/types/api'

const PERIODS = [
  { label: '7 д', value: '7' },
  { label: '30 д', value: '30' },
  { label: '90 д', value: '90' },
]

const WEEKDAYS = ['Пн', 'Вт', 'Ср', 'Чт', 'Пт', 'Сб', 'Вс']

function Heatmap({ data }: { data: HeatCell[] }) {
  if (!data.length) return null
  const max = Math.max(...data.map((d) => d.value), 1)
  const cells = Array.from({ length: 7 }, (_, wd) =>
    Array.from({ length: 24 }, (_, h) => {
      const cell = data.find((d) => d.weekday === wd && d.hour === h)
      return cell?.value ?? 0
    })
  )

  return (
    <Paper withBorder p="md">
      <Text fw={600} mb="sm">Активность по часам</Text>
      <div style={{ overflowX: 'auto' }}>
        <table style={{ borderCollapse: 'collapse', fontSize: 11 }}>
          <thead>
            <tr>
              <th style={{ width: 30 }} />
              {Array.from({ length: 24 }, (_, h) => (
                <th key={h} style={{ width: 18, textAlign: 'center', color: 'var(--mantine-color-dimmed)' }}>{h}</th>
              ))}
            </tr>
          </thead>
          <tbody>
            {cells.map((hours, wd) => (
              <tr key={wd}>
                <td style={{ paddingRight: 6, color: 'var(--mantine-color-dimmed)' }}>{WEEKDAYS[wd]}</td>
                {hours.map((val, h) => {
                  const opacity = val === 0 ? 0.05 : 0.1 + (val / max) * 0.9
                  return (
                    <td
                      key={h}
                      title={`${WEEKDAYS[wd]}, ${h}:00 — ${val} переходов`}
                      style={{
                        width: 18, height: 18,
                        background: `rgba(34, 139, 230, ${opacity})`,
                        borderRadius: 2,
                        cursor: 'default',
                      }}
                    />
                  )
                })}
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </Paper>
  )
}

export function Dashboard() {
  const [period, setPeriod] = useState('30')
  const { isAdmin } = useAuth()

  const { data, isLoading } = useQuery({
    queryKey: ['dashboard', period],
    queryFn: () => getDashboard(Number(period)),
  })

  const deviceChartData = data
    ? [
        { name: 'Десктоп', value: data.devices.devices.desktop, color: 'blue' },
        { name: 'Мобильный', value: data.devices.devices.mobile, color: 'teal' },
        { name: 'Планшет', value: data.devices.devices.tablet, color: 'orange' },
      ]
    : []

  const browsersData = data?.devices.browsers.map((b) => ({ name: b.name, value: b.count, color: 'blue' })) ?? []
  const osData = data?.devices.os.map((o) => ({ name: o.name, value: o.count, color: 'violet' })) ?? []

  return (
    <Stack gap="lg">
      <Group justify="space-between">
        <Title order={2}>Главная</Title>
        <SegmentedControl
          data={PERIODS}
          value={period}
          onChange={setPeriod}
          size="xs"
        />
      </Group>

      {/* Метрики */}
      <SimpleGrid cols={{ base: 1, sm: 2, lg: 4 }}>
        {isLoading ? (
          Array.from({ length: isAdmin() ? 4 : 3 }).map((_, i) => <Skeleton key={i} h={100} radius="md" />)
        ) : (
          <>
            <StatCard title="Активных ссылок" value={data?.overview.linksCount ?? 0} icon={<IconLink size={20} />} />
            <StatCard title="Переходов за период" value={data?.visits.clicksTotal ?? 0} icon={<IconEye size={20} />} />
            <StatCard title="Всего переходов" value={data?.overview.visitsTotal ?? 0} icon={<IconChartBar size={20} />} />
            {isAdmin() && <StatCard title="Пользователей" value={data?.users?.length ?? 0} icon={<IconUsers size={20} />} color="violet" />}
          </>
        )}
      </SimpleGrid>

      {/* График */}
      <Paper withBorder p="md">
        <Text fw={600} mb="sm">Переходы по дням</Text>
        {isLoading ? <Skeleton h={200} /> : (
          <LineChart
            h={200}
            data={data?.visits.clicksPerDay ?? []}
            dataKey="date"
            series={[{ name: 'clicks', color: 'blue', label: 'Переходы' }]}
            withTooltip
          />
        )}
      </Paper>

      {/* Donut чарты */}
      <SimpleGrid cols={{ base: 1, sm: 3 }}>
        <Paper withBorder p="md">
          <Text fw={600} mb="sm">Устройства</Text>
          {isLoading ? <Skeleton h={160} /> : <DonutChart data={deviceChartData} withLabelsLine withLabels />}
        </Paper>
        <Paper withBorder p="md">
          <Text fw={600} mb="sm">Браузеры</Text>
          {isLoading ? <Skeleton h={160} /> : <DonutChart data={browsersData} withLabelsLine withLabels />}
        </Paper>
        <Paper withBorder p="md">
          <Text fw={600} mb="sm">Операционные системы</Text>
          {isLoading ? <Skeleton h={160} /> : <DonutChart data={osData} withLabelsLine withLabels />}
        </Paper>
      </SimpleGrid>

      {/* Heatmap */}
      {!isLoading && data?.devices.heatmap?.length ? <Heatmap data={data.devices.heatmap} /> : null}

      {/* Топ ссылок */}
      <Paper withBorder p="md">
        <Text fw={600} mb="sm">Топ ссылок</Text>
        {isLoading ? <Skeleton h={200} /> : (
          <Table striped highlightOnHover>
            <Table.Thead>
              <Table.Tr>
                <Table.Th>Код</Table.Th>
                <Table.Th>Название / URL</Table.Th>
                <Table.Th>Переходов</Table.Th>
              </Table.Tr>
            </Table.Thead>
            <Table.Tbody>
              {data?.overview.topLinks.map((link) => (
                <Table.Tr key={link.shortCode}>
                  <Table.Td>
                    <Group gap={4}>
                      <Anchor href={`/links/${link.shortCode}`} size="sm">{link.shortCode}</Anchor>
                      <CopyButton value={link.shortUrl} />
                    </Group>
                  </Table.Td>
                  <Table.Td>
                    <Text size="sm" truncate maw={300}>{link.title || link.longUrl}</Text>
                  </Table.Td>
                  <Table.Td>{link.visitsTotal}</Table.Td>
                </Table.Tr>
              ))}
            </Table.Tbody>
          </Table>
        )}
      </Paper>

      {/* Последние ссылки */}
      <Paper withBorder p="md">
        <Text fw={600} mb="sm">Последние ссылки</Text>
        {isLoading ? <Skeleton h={150} /> : (
          <Table striped highlightOnHover>
            <Table.Thead>
              <Table.Tr>
                <Table.Th>Код</Table.Th>
                <Table.Th>URL</Table.Th>
                <Table.Th>Переходов</Table.Th>
              </Table.Tr>
            </Table.Thead>
            <Table.Tbody>
              {data?.overview.recentLinks.map((link) => (
                <Table.Tr key={link.shortCode}>
                  <Table.Td>
                    <Anchor href={`/links/${link.shortCode}`} size="sm">{link.shortCode}</Anchor>
                  </Table.Td>
                  <Table.Td>
                    <Text size="sm" truncate maw={300}>{link.title || link.longUrl}</Text>
                  </Table.Td>
                  <Table.Td>{link.visitsTotal}</Table.Td>
                </Table.Tr>
              ))}
            </Table.Tbody>
          </Table>
        )}
      </Paper>

      {/* Теги (admin) */}
      {isAdmin() && data?.tags && (
        <Paper withBorder p="md">
          <Text fw={600} mb="sm">Популярные теги</Text>
          <BarChart
            h={200}
            data={data.tags.slice(0, 20)}
            dataKey="tag"
            series={[{ name: 'visits', color: 'blue', label: 'Переходы' }]}
            orientation="vertical"
          />
        </Paper>
      )}

      {/* Пользователи (admin) */}
      {isAdmin() && data?.users && (
        <Paper withBorder p="md">
          <Text fw={600} mb="sm">Пользователи</Text>
          <Table striped highlightOnHover>
            <Table.Thead>
              <Table.Tr>
                <Table.Th>Пользователь</Table.Th>
                <Table.Th>Роль</Table.Th>
                <Table.Th>Статус</Table.Th>
                <Table.Th />
              </Table.Tr>
            </Table.Thead>
            <Table.Tbody>
              {data.users.map((u) => (
                <Table.Tr key={u.sub}>
                  <Table.Td>{u.username}</Table.Td>
                  <Table.Td><RoleBadge role={u.role} /></Table.Td>
                  <Table.Td><StatusBadge active={u.status === 'active'} activeLabel={u.status} inactiveLabel={u.status} /></Table.Td>
                  <Table.Td>
                    <Anchor href={`/admin/users/${u.sub}`} size="sm">Открыть</Anchor>
                  </Table.Td>
                </Table.Tr>
              ))}
            </Table.Tbody>
          </Table>
        </Paper>
      )}
    </Stack>
  )
}
