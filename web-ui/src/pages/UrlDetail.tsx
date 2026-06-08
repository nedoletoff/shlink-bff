import { useState } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import {
  Stack, Group, Title, Text, Badge, Button,
  Paper, SimpleGrid, Skeleton, SegmentedControl,
  Table, ActionIcon, Tooltip, Modal, TextInput,
  TagsInput, NumberInput,
} from '@mantine/core'
import { DateTimePicker } from '@mantine/dates'
import { useForm } from '@mantine/form'
import {
  ResponsiveContainer, BarChart, Bar,
  XAxis, YAxis, Tooltip as RTooltip, CartesianGrid,
} from 'recharts'
import {
  IconArrowLeft, IconExternalLink, IconPencil,
  IconTrash, IconBan, IconCircleCheck,
} from '@tabler/icons-react'
import { notifications } from '@mantine/notifications'
import { getLinkDetail } from '@/api/endpoints/linkDetail'
import { editLink, deleteLink, deactivateLink, activateLink } from '@/api/endpoints/links'
import { StatCard } from '@/components/ui/StatCard'
import { CopyButton } from '@/components/ui/CopyButton'
import { ConfirmDialog } from '@/components/ui/ConfirmDialog'
import { formatDate, formatDateTime } from '@/utils/date'
import type { EditShortURLPayload } from '@/types/api'

const PERIODS = [
  { label: '7 дней', value: '7' },
  { label: '30 дней', value: '30' },
  { label: '90 дней', value: '90' },
]

export function UrlDetail() {
  const { shortCode } = useParams<{ shortCode: string }>()
  const navigate = useNavigate()
  const qc = useQueryClient()
  const [period, setPeriod] = useState('30')
  const [editOpened, setEditOpened] = useState(false)
  const [confirmDelete, setConfirmDelete] = useState(false)

  const { data, isLoading } = useQuery({
    queryKey: ['link-detail', shortCode, period],
    queryFn: () => getLinkDetail(shortCode!, Number(period)),
    enabled: Boolean(shortCode),
  })

  const form = useForm<EditShortURLPayload>({
    initialValues: {
      longUrl: data?.longUrl ?? '',
      title: data?.title ?? '',
      tags: [],
      maxVisits: undefined,
      validSince: undefined,
      validUntil: undefined,
    },
  })

  const openEdit = () => {
    form.setValues({
      longUrl: data?.longUrl ?? '',
      title: data?.title ?? '',
      tags: [],
      maxVisits: undefined,
      validSince: undefined,
      validUntil: undefined,
    })
    setEditOpened(true)
  }

  const editMutation = useMutation({
    mutationFn: (values: EditShortURLPayload) => editLink(shortCode!, '', values),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['link-detail', shortCode] })
      notifications.show({ color: 'teal', message: 'Ссылка обновлена' })
      setEditOpened(false)
    },
  })

  const toggleMutation = useMutation({
    mutationFn: () =>
      data?.isActive ? deactivateLink(shortCode!) : activateLink(shortCode!),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['link-detail', shortCode] })
      notifications.show({
        color: data?.isActive ? 'orange' : 'teal',
        message: data?.isActive ? 'Ссылка деактивирована' : 'Ссылка активирована',
      })
    },
  })

  const deleteMutation = useMutation({
    mutationFn: () => deleteLink(shortCode!, ''),
    onSuccess: () => {
      notifications.show({ color: 'teal', message: 'Ссылка удалена' })
      navigate('/links')
    },
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
        <Group justify="space-between" align="flex-start" wrap="wrap" gap="sm">
          <Stack gap={4}>
            <Group gap="xs">
              <Title order={2}>{data?.shortCode}</Title>
              <Badge color={data?.isActive ? 'teal' : 'gray'} variant="dot">
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

          <Group gap="xs">
            <SegmentedControl size="xs" data={PERIODS} value={period} onChange={setPeriod} />
            <Tooltip label="Редактировать">
              <ActionIcon variant="default" size="md" onClick={openEdit} aria-label="Редактировать">
                <IconPencil size={15} />
              </ActionIcon>
            </Tooltip>
            <Tooltip label={data?.isActive ? 'Деактивировать' : 'Активировать'}>
              <ActionIcon
                variant="default" size="md"
                color={data?.isActive ? 'orange' : 'teal'}
                onClick={() => toggleMutation.mutate()}
                loading={toggleMutation.isPending}
                aria-label={data?.isActive ? 'Деактивировать' : 'Активировать'}
              >
                {data?.isActive ? <IconBan size={15} /> : <IconCircleCheck size={15} />}
              </ActionIcon>
            </Tooltip>
            <Tooltip label="Удалить">
              <ActionIcon
                variant="default" size="md" color="red"
                onClick={() => setConfirmDelete(true)}
                aria-label="Удалить"
              >
                <IconTrash size={15} />
              </ActionIcon>
            </Tooltip>
          </Group>
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

      {/* Edit modal */}
      <Modal opened={editOpened} onClose={() => setEditOpened(false)} title="Редактировать ссылку" size="md">
        <form onSubmit={form.onSubmit((v) => editMutation.mutate(v))}>
          <Stack gap="sm">
            <TextInput label="Длинный URL" required {...form.getInputProps('longUrl')} />
            <TextInput label="Заголовок" {...form.getInputProps('title')} />
            <TagsInput
              label="Теги"
              placeholder="Добавить тег → Enter"
              {...form.getInputProps('tags')}
            />
            <NumberInput label="Макс. переходов" min={1} {...form.getInputProps('maxVisits')} />
            <Group grow>
              <DateTimePicker
                label="Действителен с"
                placeholder="Не ограничен"
                clearable
                value={form.values.validSince ? new Date(form.values.validSince) : null}
                onChange={(d) => form.setFieldValue('validSince', d ? d.toISOString() : undefined)}
              />
              <DateTimePicker
                label="Действителен до"
                placeholder="Не ограничен"
                clearable
                value={form.values.validUntil ? new Date(form.values.validUntil) : null}
                onChange={(d) => form.setFieldValue('validUntil', d ? d.toISOString() : undefined)}
              />
            </Group>
            <Group justify="flex-end" mt="xs">
              <Button variant="default" onClick={() => setEditOpened(false)}>Отмена</Button>
              <Button type="submit" loading={editMutation.isPending}>Сохранить</Button>
            </Group>
          </Stack>
        </form>
      </Modal>

      {/* Delete confirm */}
      <ConfirmDialog
        opened={confirmDelete}
        onClose={() => setConfirmDelete(false)}
        onConfirm={() => deleteMutation.mutate()}
        message={`Удалить ссылку «${shortCode}»? Действие необратимо.`}
        loading={deleteMutation.isPending}
      />
    </Stack>
  )
}
