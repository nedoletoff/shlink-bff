import { useState } from 'react'
import {
  Title, Stack, Group, Button, Text, Paper, Badge, Alert,
  SegmentedControl, Table, Anchor, Skeleton, ActionIcon, Tooltip, Pagination,
} from '@mantine/core'
import { DonutChart, LineChart } from '@mantine/charts'
import { IconArrowLeft, IconAlertTriangle, IconEdit, IconPlayerPause, IconPlayerPlay, IconTrash, IconExternalLink } from '@tabler/icons-react'
import { useParams, useNavigate } from 'react-router-dom'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { notifications } from '@mantine/notifications'
import { getLinkDetail } from '@/api/endpoints/linkDetail'
import { deactivateLink, activateLink, deleteLink } from '@/api/endpoints/links'
import { CopyButton } from '@/components/ui/CopyButton'
import { StatusBadge } from '@/components/ui/StatusBadge'
import { ConfirmDialog } from '@/components/ui/ConfirmDialog'
import { EmptyState } from '@/components/ui/EmptyState'
import { useAuth } from '@/contexts/AuthContext'
import { formatDate, formatDateTime, formatDateLong } from '@/utils/date'

const PERIODS = [{ label: '7 д', value: '7' }, { label: '30 д', value: '30' }, { label: '90 д', value: '90' }]
const DEVICE_ICON: Record<string, string> = { desktop: '💻', mobile: '📱', tablet: '📙' }

export function UrlDetail() {
  const { shortCode } = useParams<{ shortCode: string }>()
  const navigate = useNavigate()
  const qc = useQueryClient()
  const { can } = useAuth()
  const [period, setPeriod] = useState('30')
  const [visitsPage, setVisitsPage] = useState(1)
  const [visitsSortAsc, setVisitsSortAsc] = useState(false)
  const [confirmDelete, setConfirmDelete] = useState(false)
  const [confirmDeactivate, setConfirmDeactivate] = useState(false)

  const { data, isLoading } = useQuery({
    queryKey: ['urlDetail', shortCode, period],
    queryFn: () => getLinkDetail(shortCode!, Number(period)),
    enabled: !!shortCode,
  })

  const deactivateMutation = useMutation({
    mutationFn: () => deactivateLink(shortCode!),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['urlDetail', shortCode] }); setConfirmDeactivate(false) },
  })

  const activateMutation = useMutation({
    mutationFn: () => activateLink(shortCode!),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['urlDetail', shortCode] }),
  })

  const deleteMutation = useMutation({
    mutationFn: () => deleteLink(shortCode!),
    onSuccess: () => { notifications.show({ color: 'teal', message: 'Ссылка удалена' }); navigate('/links') },
  })

  const deviceData = data
    ? [
        { name: 'Десктоп', value: data.devices.desktop, color: 'blue' },
        { name: 'Мобильный', value: data.devices.mobile, color: 'teal' },
        { name: 'Планшет', value: data.devices.tablet, color: 'orange' },
      ]
    : []

  const sortedVisits = [...(data?.visits ?? [])].sort((a, b) =>
    visitsSortAsc ? a.date.localeCompare(b.date) : b.date.localeCompare(a.date)
  )
  const perPage = 20
  const totalVisitPages = Math.ceil(sortedVisits.length / perPage)
  const pagedVisits = sortedVisits.slice((visitsPage - 1) * perPage, visitsPage * perPage)

  return (
    <Stack gap="md">
      <Group>
        <Button variant="subtle" leftSection={<IconArrowLeft size={16} />} onClick={() => navigate('/links')}>
          Назад
        </Button>
        <Group gap="xs">
          {isLoading ? <Skeleton h={24} w={100} /> : (
            <>
              <Title order={3}>{data?.shortCode}</Title>
              <StatusBadge active={data?.isActive} />
            </>
          )}
        </Group>
      </Group>

      {!isLoading && data && (
        <Group gap="xs">
          <Text size="sm" c="dimmed">{data.shortUrl}</Text>
          <CopyButton value={data.shortUrl} />
        </Group>
      )}

      {/* Деактивирована */}
      {!isLoading && data && !data.isActive && (
        <Alert color="orange" icon={<IconAlertTriangle />} title="Ссылка деактивирована">
          <Text size="sm">Деактивирована: {formatDateTime(data.deactivatedAt)}</Text>
          {data.deactivatedBy && <Text size="sm">Кем: {data.deactivatedBy}</Text>}
        </Alert>
      )}

      {/* Акции */}
      <Group>
        {(can('canEditOwnLinks') || can('canEditAllLinks')) && (
          <Button size="xs" leftSection={<IconEdit size={14} />} variant="default">Редактировать</Button>
        )}
        {data?.isActive && (can('canDeactivateOwnLinks') || can('canDeactivateAllLinks')) && (
          <Button size="xs" color="orange" leftSection={<IconPlayerPause size={14} />} onClick={() => setConfirmDeactivate(true)}>
            Деактивировать
          </Button>
        )}
        {data && !data.isActive && (can('canReactivateOwnLinks') || can('canReactivateAllLinks')) && (
          <Button size="xs" color="teal" leftSection={<IconPlayerPlay size={14} />} loading={activateMutation.isPending} onClick={() => activateMutation.mutate()}>
            Активировать
          </Button>
        )}
        {(can('canDeleteOwnLinksPermanently') || can('canDeleteAllLinksPermanently')) && (
          <Button size="xs" color="red" leftSection={<IconTrash size={14} />} onClick={() => setConfirmDelete(true)}>
            Удалить
          </Button>
        )}
      </Group>

      {/* Информация */}
      {isLoading ? <Skeleton h={150} radius="md" /> : data && (
        <Paper withBorder p="md">
          <Stack gap="xs">
            <Group><Text size="sm" fw={500} w={160}>Назначение:</Text>
              <Anchor href={data.longUrl} target="_blank" size="sm">{data.longUrl} <IconExternalLink size={12} /></Anchor>
            </Group>
            <Group><Text size="sm" fw={500} w={160}>Создана:</Text><Text size="sm">{formatDateLong(data.dateCreated)}</Text></Group>
            <Group><Text size="sm" fw={500} w={160}>Всего переходов:</Text><Text size="sm">{data.visitsTotal}</Text></Group>
          </Stack>
        </Paper>
      )}

      <Group justify="space-between">
        <Text fw={600}>Статистика за период</Text>
        <SegmentedControl data={PERIODS} value={period} onChange={setPeriod} size="xs" />
      </Group>

      {/* График */}
      <Paper withBorder p="md">
        <Text fw={600} mb="sm">Переходы по дням</Text>
        {isLoading ? <Skeleton h={200} /> : (
          <LineChart
            h={200}
            data={data?.clicksPerDay ?? []}
            dataKey="date"
            series={[{ name: 'clicks', color: 'blue', label: 'Переходы' }]}
            withTooltip
          />
        )}
      </Paper>

      {/* Donut чарты */}
      <Group grow>
        <Paper withBorder p="md">
          <Text fw={600} mb="sm">Устройства</Text>
          {isLoading ? <Skeleton h={160} /> : <DonutChart data={deviceData} withLabelsLine withLabels />}
        </Paper>
        <Paper withBorder p="md">
          <Text fw={600} mb="sm">Браузеры</Text>
          {isLoading ? <Skeleton h={160} /> : (
            <DonutChart
              data={(data?.browsers ?? []).map((b, i) => ({ name: b.name, value: b.count, color: ['blue', 'teal', 'violet', 'orange', 'pink'][i % 5] }))}
              withLabelsLine withLabels
            />
          )}
        </Paper>
        <Paper withBorder p="md">
          <Text fw={600} mb="sm">Операционные системы</Text>
          {isLoading ? <Skeleton h={160} /> : (
            <DonutChart
              data={(data?.os ?? []).map((o, i) => ({ name: o.name, value: o.count, color: ['green', 'cyan', 'lime', 'yellow', 'red'][i % 5] }))}
              withLabelsLine withLabels
            />
          )}
        </Paper>
      </Group>

      {/* Таблица переходов */}
      <Paper withBorder p="md">
        <Text fw={600} mb="sm">Переходы</Text>
        {isLoading ? <Skeleton h={300} /> : pagedVisits.length === 0 ? (
          <EmptyState title="Переходов за этот период нет" />
        ) : (
          <>
            <Table striped highlightOnHover>
              <Table.Thead>
                <Table.Tr>
                  <Table.Th
                    style={{ cursor: 'pointer' }}
                    onClick={() => { setVisitsSortAsc(!visitsSortAsc); setVisitsPage(1) }}
                  >
                    Дата {visitsSortAsc ? '↑' : '↓'}
                  </Table.Th>
                  <Table.Th>Браузер</Table.Th>
                  <Table.Th>ОС</Table.Th>
                  <Table.Th>Устройство</Table.Th>
                  <Table.Th>Страна</Table.Th>
                  <Table.Th>Реферер</Table.Th>
                </Table.Tr>
              </Table.Thead>
              <Table.Tbody>
                {pagedVisits.map((v, i) => (
                  <Table.Tr key={i}>
                    <Table.Td><Text size="sm">{formatDateTime(v.date)}</Text></Table.Td>
                    <Table.Td>{v.browser}</Table.Td>
                    <Table.Td>{v.os}</Table.Td>
                    <Table.Td>{DEVICE_ICON[v.device] ?? ''} {v.device}</Table.Td>
                    <Table.Td>{v.country ?? '—'}</Table.Td>
                    <Table.Td>
                      {v.referer ? (
                        <Tooltip label={v.referer} multiline maw={300}>
                          <Text size="sm" truncate maw={160}>{v.referer}</Text>
                        </Tooltip>
                      ) : '—'}
                    </Table.Td>
                  </Table.Tr>
                ))}
              </Table.Tbody>
            </Table>
            {totalVisitPages > 1 && (
              <Group justify="center" mt="sm">
                <Pagination total={totalVisitPages} value={visitsPage} onChange={setVisitsPage} size="sm" />
              </Group>
            )}
          </>
        )}
      </Paper>

      <ConfirmDialog
        opened={confirmDeactivate}
        onClose={() => setConfirmDeactivate(false)}
        onConfirm={() => deactivateMutation.mutate()}
        title="Деактивировать ссылку?"
        message={`Ссылка ${shortCode} будет деактивирована.`}
        confirmLabel="Деактивировать"
        loading={deactivateMutation.isPending}
      />
      <ConfirmDialog
        opened={confirmDelete}
        onClose={() => setConfirmDelete(false)}
        onConfirm={() => deleteMutation.mutate()}
        title="Удалить ссылку?"
        message={`Ссылка ${shortCode} будет удалена безвозвратно.`}
        confirmLabel="Удалить"
        loading={deleteMutation.isPending}
        danger
      />
    </Stack>
  )
}
