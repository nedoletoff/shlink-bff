import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import {
  Stack, Group, Title, SegmentedControl,
  SimpleGrid, Paper, Text, Skeleton,
} from '@mantine/core'
import {
  ResponsiveContainer, LineChart, Line,
  XAxis, YAxis, Tooltip as RTooltip, CartesianGrid,
  PieChart, Pie, Cell,
} from 'recharts'
import { IconLink, IconEye } from '@tabler/icons-react'
import { getDashboard } from '@/api/endpoints/dashboard'
import { StatCard } from '@/components/ui/StatCard'
import { EmptyState } from '@/components/ui/EmptyState'
import { formatDate } from '@/utils/date'

const COLORS = ['#0e9488', '#3b82f6', '#f59e0b', '#8b5cf6', '#ec4899']

const PERIODS = [
  { label: '7 дней', value: '7' },
  { label: '30 дней', value: '30' },
  { label: '90 дней', value: '90' },
]

export function Dashboard() {
  const [period, setPeriod] = useState('7')
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

  return (
    <Stack gap="lg">
      <Group justify="space-between" align="center">
        <Title order={2}>Дашборд</Title>
        <SegmentedControl
          size="xs"
          data={PERIODS}
          value={period}
          onChange={setPeriod}
        />
      </Group>

      <SimpleGrid cols={{ base: 1, sm: 2, md: 4 }} spacing="md">
        <Skeleton visible={isLoading} radius="md">
          <StatCard
            label="Всего ссылок"
            value={data?.overview.linksCount ?? 0}
            icon={<IconLink size={18} color="teal" />}
          />
        </Skeleton>
        <Skeleton visible={isLoading} radius="md">
          <StatCard
            label="Всего переходов"
            value={data?.visits.clicksTotal ?? 0}
            icon={<IconEye size={18} color="teal" />}
          />
        </Skeleton>
      </SimpleGrid>

      <Paper withBorder p="md" radius="md">
        <Text fw={600} mb="md">Переходы по дням</Text>
        {isLoading ? (
          <Skeleton height={220} />
        ) : clicksData.length === 0 ? (
          <EmptyState icon={<IconEye size={24} />} title="Нет данных за период" />
        ) : (
          <ResponsiveContainer width="100%" height={220}>
            <LineChart data={clicksData}>
              <CartesianGrid strokeDasharray="3 3" stroke="var(--mantine-color-gray-2)" />
              <XAxis dataKey="date" tick={{ fontSize: 12 }} />
              <YAxis tick={{ fontSize: 12 }} />
              <RTooltip />
              <Line type="monotone" dataKey="clicks" stroke="#0e9488" strokeWidth={2} dot={false} />
            </LineChart>
          </ResponsiveContainer>
        )}
      </Paper>

      <SimpleGrid cols={{ base: 1, md: 2 }} spacing="md">
        <Paper withBorder p="md" radius="md">
          <Text fw={600} mb="md">Браузеры</Text>
          {isLoading ? (
            <Skeleton height={180} />
          ) : (
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
          {isLoading ? (
            <Skeleton height={180} />
          ) : (
            <ResponsiveContainer width="100%" height={180}>
              <PieChart>
                <Pie data={osData} dataKey="count" nameKey="name" cx="50%" cy="50%" outerRadius={70} label={(e) => e.name}>
                  {osData.map((_, i) => <Cell key={i} fill={COLORS[i % COLORS.length]} />)}
                </Pie>
                <RTooltip />
              </PieChart>
            </ResponsiveContainer>
          )}
        </Paper>
      </SimpleGrid>
    </Stack>
  )
}
