import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import {
  Stack, Group, Title, TextInput, Select, Button,
  Table, Text, ActionIcon, Tooltip, Modal,
  TagsInput, NumberInput, Skeleton, Pagination, Badge,
} from '@mantine/core'
import { DateTimePicker } from '@mantine/dates'
import { useForm } from '@mantine/form'
import { useDisclosure } from '@mantine/hooks'
import { notifications } from '@mantine/notifications'
import {
  IconPlus, IconPencil, IconTrash, IconLink,
  IconExternalLink, IconBan, IconCircleCheck,
} from '@tabler/icons-react'
import { useNavigate } from 'react-router-dom'
import { getLinks, createLink, editLink, deleteLink, deactivateLink, activateLink } from '@/api/endpoints/links'
import type { CreateShortURLPayload, ShortURL } from '@/types/api'
import { CopyButton } from '@/components/ui/CopyButton'
import { ConfirmDialog } from '@/components/ui/ConfirmDialog'
import { EmptyState } from '@/components/ui/EmptyState'
import { formatDate } from '@/utils/date'

// ─── helpers ─────────────────────────────────────────────────────────────────────
const TAG_COLORS = ['teal', 'blue', 'grape', 'orange', 'cyan', 'pink', 'lime', 'indigo']
function tagColor(tag: string) {
  let h = 0
  for (let i = 0; i < tag.length; i++) h = (h * 31 + tag.charCodeAt(i)) & 0xffffffff
  return TAG_COLORS[Math.abs(h) % TAG_COLORS.length]
}

// ─── LinkModal ────────────────────────────────────────────────────────────────────
function LinkModal({ opened, onClose, initial }: { opened: boolean; onClose: () => void; initial?: ShortURL | null }) {
  const qc = useQueryClient()
  const isEdit = Boolean(initial)

  const form = useForm<CreateShortURLPayload>({
    initialValues: {
      longUrl: initial?.longUrl ?? '',
      title: initial?.title ?? '',
      customSlug: initial?.shortCode ?? '',
      tags: initial?.tags ?? [],
      maxVisits: initial?.maxVisits ?? undefined,
      validSince: initial?.validSince ?? undefined,
      validUntil: initial?.validUntil ?? undefined,
    },
  })

  const mutation = useMutation({
    mutationFn: (values: CreateShortURLPayload) =>
      isEdit
        ? editLink(initial!.shortCode, '', {
            longUrl: values.longUrl,
            title: values.title,
            tags: values.tags,
            maxVisits: values.maxVisits,
            validSince: values.validSince,
            validUntil: values.validUntil,
          })
        : createLink(values),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['links'] })
      notifications.show({ color: 'teal', message: isEdit ? 'Ссылка обновлена' : 'Ссылка создана' })
      onClose()
    },
  })

  return (
    <Modal opened={opened} onClose={onClose} title={isEdit ? 'Редактировать ссылку' : 'Создать ссылку'} size="md">
      <form onSubmit={form.onSubmit((v: CreateShortURLPayload) => mutation.mutate(v))}>
        <Stack gap="sm">
          <TextInput label="Длинный URL" required {...form.getInputProps('longUrl')} />
          <TextInput label="Заголовок" {...form.getInputProps('title')} />
          {!isEdit && <TextInput label="Кастомный слаг" {...form.getInputProps('customSlug')} />}
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
            <Button variant="default" onClick={onClose}>Отмена</Button>
            <Button type="submit" loading={mutation.isPending}>{isEdit ? 'Сохранить' : 'Создать'}</Button>
          </Group>
        </Stack>
      </form>
    </Modal>
  )
}

// ─── ShortUrls page ─────────────────────────────────────────────────────────────────
export function ShortUrls() {
  const navigate = useNavigate()
  const qc = useQueryClient()
  const [page, setPage] = useState(1)
  const [search, setSearch] = useState('')
  const [status, setStatus] = useState<string | null>('all')
  const [editing, setEditing] = useState<ShortURL | null>(null)
  const [deleting, setDeleting] = useState<ShortURL | null>(null)
  const [createOpened, { open: openCreate, close: closeCreate }] = useDisclosure()

  const { data, isLoading } = useQuery({
    queryKey: ['links', page, search, status],
    queryFn: () => getLinks({ page, itemsPerPage: 20, searchTerm: search || undefined, status: (status as 'active' | 'inactive' | 'all') ?? 'all' }),
  })

  const deleteMutation = useMutation({
    mutationFn: () => deleteLink(deleting!.shortCode, ''),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['links'] })
      notifications.show({ color: 'teal', message: 'Ссылка удалена' })
      setDeleting(null)
    },
  })

  const toggleMutation = useMutation({
    mutationFn: (link: ShortURL) =>
      link.isActive ? deactivateLink(link.shortCode) : activateLink(link.shortCode),
    onSuccess: (_data, link) => {
      qc.invalidateQueries({ queryKey: ['links'] })
      notifications.show({
        color: link.isActive ? 'orange' : 'teal',
        message: link.isActive ? 'Ссылка деактивирована' : 'Ссылка активирована',
      })
    },
  })

  const links = data?.shortUrls.data ?? []
  const pagination = data?.shortUrls.pagination

  return (
    <Stack gap="md">
      <Group justify="space-between">
        <Title order={2}>Ссылки</Title>
        <Button leftSection={<IconPlus size={16} />} onClick={openCreate}>Создать</Button>
      </Group>

      <Group gap="sm">
        <TextInput
          placeholder="Поиск..."
          value={search}
          onChange={(e) => { setSearch(e.currentTarget.value); setPage(1) }}
          style={{ flex: 1 }}
        />
        <Select
          data={[
            { value: 'all', label: 'Все' },
            { value: 'active', label: 'Активные' },
            { value: 'inactive', label: 'Неактивные' },
          ]}
          value={status}
          onChange={(v: string | null) => { setStatus(v); setPage(1) }}
          w={150}
        />
      </Group>

      {isLoading ? (
        <Stack gap="xs">
          {Array.from({ length: 6 }).map((_, i) => <Skeleton key={i} height={44} radius="sm" />)}
        </Stack>
      ) : links.length === 0 ? (
        <EmptyState
          icon={<IconLink size={24} />}
          title="Ссылок нет"
          description="Создайте первую короткую ссылку"
          action={<Button onClick={openCreate}>Создать</Button>}
        />
      ) : (
        <Table.ScrollContainer minWidth={640}>
          <Table striped highlightOnHover>
            <Table.Thead>
              <Table.Tr>
                <Table.Th>Ссылка</Table.Th>
                <Table.Th>Назначение</Table.Th>
                <Table.Th>Теги</Table.Th>
                <Table.Th>Переходы</Table.Th>
                <Table.Th>Создана</Table.Th>
                <Table.Th>Статус</Table.Th>
                <Table.Th />
              </Table.Tr>
            </Table.Thead>
            <Table.Tbody>
              {links.map((link) => (
                <Table.Tr
                  key={link.shortCode}
                  style={{ opacity: link.isActive === false ? 0.55 : 1 }}
                >
                  <Table.Td>
                    <Group gap={4} wrap="nowrap">
                      <Text
                        size="sm" fw={500}
                        style={{ cursor: 'pointer', color: 'var(--mantine-color-teal-6)' }}
                        onClick={() => navigate(`/links/${link.shortCode}`)}
                      >
                        {link.shortCode}
                      </Text>
                      <CopyButton value={link.shortUrl} />
                      <ActionIcon variant="subtle" size="xs" component="a" href={link.shortUrl} target="_blank" aria-label="Открыть">
                        <IconExternalLink size={12} />
                      </ActionIcon>
                    </Group>
                  </Table.Td>
                  <Table.Td>
                    <Text size="sm" lineClamp={1} title={link.longUrl}>{link.title || link.longUrl}</Text>
                  </Table.Td>
                  <Table.Td>
                    <Group gap={4} wrap="wrap">
                      {link.tags?.map((t) => (
                        <Badge key={t} size="xs" variant="light" color={tagColor(t)}>{t}</Badge>
                      ))}
                    </Group>
                  </Table.Td>
                  <Table.Td>
                    <Text size="sm" style={{ fontVariantNumeric: 'tabular-nums' }}>
                      {link.visitsSummary.total.toLocaleString('ru')}
                    </Text>
                  </Table.Td>
                  <Table.Td>
                    <Text size="sm" c="dimmed">{formatDate(link.dateCreated)}</Text>
                  </Table.Td>
                  <Table.Td>
                    <Badge
                      size="sm"
                      variant="dot"
                      color={link.isActive !== false ? 'teal' : 'gray'}
                    >
                      {link.isActive !== false ? 'Активна' : 'Неактивна'}
                    </Badge>
                  </Table.Td>
                  <Table.Td>
                    <Group gap={4} wrap="nowrap" justify="flex-end">
                      <Tooltip label="Редактировать">
                        <ActionIcon variant="subtle" size="sm" onClick={() => setEditing(link)} aria-label="Редактировать">
                          <IconPencil size={14} />
                        </ActionIcon>
                      </Tooltip>
                      <Tooltip label={link.isActive !== false ? 'Деактивировать' : 'Активировать'}>
                        <ActionIcon
                          variant="subtle" size="sm"
                          color={link.isActive !== false ? 'orange' : 'teal'}
                          onClick={() => toggleMutation.mutate(link)}
                          loading={toggleMutation.isPending}
                          aria-label={link.isActive !== false ? 'Деактивировать' : 'Активировать'}
                        >
                          {link.isActive !== false ? <IconBan size={14} /> : <IconCircleCheck size={14} />}
                        </ActionIcon>
                      </Tooltip>
                      <Tooltip label="Удалить">
                        <ActionIcon variant="subtle" size="sm" color="red" onClick={() => setDeleting(link)} aria-label="Удалить">
                          <IconTrash size={14} />
                        </ActionIcon>
                      </Tooltip>
                    </Group>
                  </Table.Td>
                </Table.Tr>
              ))}
            </Table.Tbody>
          </Table>
        </Table.ScrollContainer>
      )}

      {pagination && pagination.pagesCount > 1 && (
        <Group justify="center">
          <Pagination total={pagination.pagesCount} value={page} onChange={setPage} size="sm" />
        </Group>
      )}

      <LinkModal opened={createOpened} onClose={closeCreate} />
      {editing && <LinkModal opened onClose={() => setEditing(null)} initial={editing} />}

      <ConfirmDialog
        opened={Boolean(deleting)}
        onClose={() => setDeleting(null)}
        onConfirm={() => deleteMutation.mutate()}
        message={`Удалить ссылку «${deleting?.shortCode}»? Действие необратимо.`}
        loading={deleteMutation.isPending}
      />
    </Stack>
  )
}
