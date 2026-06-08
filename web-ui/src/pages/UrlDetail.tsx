import { useState } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import {
  Stack, Group, Title, Text, Badge, Button,
  Paper, SimpleGrid, Skeleton, SegmentedControl,
  Table,
} from '@mantine/core'
import {
  ResponsiveContainer, BarChart, Bar,
  XAxis, YAxis, Tooltip as RTooltip, CartesianGrid,
} from 'recharts'
import { IconArrowLeft, IconExternalLink } from '@tabler/icons-react'
import { getLinkDetail } from '@/api/endpoints/linkDetail'
import { StatCard } from '@/components/ui/StatCard'
import { CopyButton } from '@/components/ui/CopyButton'
import { formatDate, formatDateTime } from '@/utils/date'

const PERIODS = [
  { label: '7 дней', value: '7' },
  { label: '30 дней', value: '30' },
  { label: '90 дней', value: '90' },
]

export function UrlDetail() {
  const { shortCode } = useParams<{ shortCode: string }>()
  const navigate = useNavigate()
  const [period, setPeriod] = useState('30')

  const { data, isLoading } = useQuery({
    queryKey: ['link-detail', shortCode, period],
    queryFn: () => getLinkDetail(shortCode!, Number(period)),
    enabled: Boolean(shortCode),
  })

  const clicksData = (data?.clicksPerDay ?? []).map((p) => ({
    ...p,
    date: formatDate(p.date),
  }))

  return (
    <Stack gap="lg">
      <Group>
        <Button variant="subtle" size="sm" leftSection={<IconArrowLeft size={14} />} onClick={() => navigate('/links')}>
          Ссылки
        </Button>
      </Group>

      {isLoading ? (
        <Skeleton height={60} />
      ) : (
        <Group justify="space-between" align="flex-start">
          <Stack gap={4}>
            <Group gap="xs">
              <Title order={2}>{data?.shortCode}</Title>
              <Badge color={data?.isActive ? 'teal' : 'gray'} variant="light">
                {data?.isActive ? 'Активна' : 'Неактивна'}
              </Badge>
            </Group>
            <Group gap={6}>
              <Text size="sm" c="dimmed">{data?.shortUrl}</Text>
              <CopyButton value={data?.shortUrl ?? ''} />
              <Button
                component="a"
                href={data?.longUrl}
                target="_blank"
                variant="subtle"
                size="xs"
                rightSection={<IconExternalLink size={12} />}
              >
                {data?.title || data?.longUrl}
              </Button>
            </Group>
          </Stack>

          <SegmentedControl size="xs" data={PERIODS} value={period} onChange={setPeriod} />
        </Group>
      )}

      <SimpleGrid cols={{ base: 1, sm: 3 }} spacing="md">
        <Skeleton visible={isLoading} radius="md">
          <StatCard label="Всего переходов" value={data?.visitsTotal ?? 0} />
        </Skeleton>
        <Skeleton visible={isLoading} radius="md">
          <StatCard label="Создана" value={data ? formatDate(data.dateCreated) : '—'} />
        </Skeleton>
        <Skeleton visible={isLoading} radius="md">
          <StatCard label="Desktop / Mobile / Tablet" value={
            data ? `${data.devices.desktop} / ${data.devices.mobile} / ${data.devices.tablet}` : '—'
          } />
        </Skeleton>
      </SimpleGrid>

      <Paper withBorder p="md" radius="md">
        <Text fw={600} mb="md">Переходы по дням</Text>
        {isLoading ? (
          <Skeleton height={220} />
        ) : (
          <ResponsiveContainer width="100%" height={220}>
            <BarChart data={clicksData}>
              <CartesianGrid strokeDasharray="3 3" stroke="var(--mantine-color-gray-2)" />
              <XAxis dataKey="date" tick={{ fontSize: 12 }} />
              <YAxis tick={{ fontSize: 12 }} />
              <RTooltip />
              <Bar dataKey="clicks" fill="#0e9488" radius={[3, 3, 0, 0]} />
            </BarChart>
          </ResponsiveContainer>
        )}
      </Paper>

      <Paper withBorder p="md" radius="md">
        <Text fw={600} mb="md">Последние визиты</Text>
        {isLoading ? (
          <Skeleton height={200} />
        ) : (
          <Table.ScrollContainer minWidth={500}>
            <Table striped highlightOnHover>
              <Table.Thead>
                <Table.Tr>
                  <Table.Th>Дата</Table.Th>
                  <Table.Th>Страна</Table.Th>
                  <Table.Th>Браузер</Table.Th>
                  <Table.Th>ОС</Table.Th>
                  <Table.Th>Устройство</Table.Th>
                  <Table.Th>Реферер</Table.Th>
                </Table.Tr>
              </Table.Thead>
              <Table.Tbody>
                {(data?.visits ?? []).slice(0, 50).map((v, i) => (
                  <Table.Tr key={i}>
                    <Table.Td><Text size="xs">{formatDateTime(v.date)}</Text></Table.Td>
                    <Table.Td><Text size="xs">{v.country ?? '—'}</Text></Table.Td>
                    <Table.Td><Text size="xs">{v.browser}</Text></Table.Td>
                    <Table.Td><Text size="xs">{v.os}</Text></Table.Td>
                    <Table.Td><Text size="xs">{v.device}</Text></Table.Td>
                    <Table.Td><Text size="xs" c="dimmed">{v.referer ?? '—'}</Text></Table.Td>
                  </Table.Tr>
                ))}
              </Table.Tbody>
            </Table>
          </Table.ScrollContainer>
        )}
      </Paper>
    </Stack>
  )
}
