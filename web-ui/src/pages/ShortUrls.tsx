import { useState, useCallback } from 'react'
import {
  Title, Stack, Group, TextInput, SegmentedControl, Button,
  Table, Badge, Text, ActionIcon, Skeleton, Pagination, Tooltip, Modal, NumberInput,
  MultiSelect,
} from '@mantine/core'
import { DateTimePicker } from '@mantine/dates'
import { useDisclosure } from '@mantine/hooks'
import {
  IconSearch, IconPlus, IconEdit, IconPlayerPause, IconPlayerPlay,
  IconTrash, IconCopy,
} from '@tabler/icons-react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { notifications } from '@mantine/notifications'
import { useNavigate } from 'react-router-dom'
import {
  listLinks, createLink, updateLink, deleteLink,
  deactivateLink, activateLink,
} from '@/api/endpoints/links'
import { getTags } from '@/api/endpoints/tags'
import { CopyButton } from '@/components/ui/CopyButton'
import { StatusBadge } from '@/components/ui/StatusBadge'
import { ConfirmDialog } from '@/components/ui/ConfirmDialog'
import { EmptyState } from '@/components/ui/EmptyState'
import { useAuth } from '@/contexts/AuthContext'
import { formatDate } from '@/utils/date'
import type { ShortURL, CreateShortURLPayload } from '@/types/api'
import dayjs from 'dayjs'

function useDebounce<T>(value: T, delay: number): T {
  const [debouncedValue, setDebouncedValue] = useState(value)
  useCallback(() => {
    const timer = setTimeout(() => setDebouncedValue(value), delay)
    return () => clearTimeout(timer)
  }, [value, delay])
  return debouncedValue
}

interface LinkFormData {
  longUrl: string
  title: string
  customSlug: string
  tags: string[]
  maxVisits: number | ''
  validSince: Date | null
  validUntil: Date | null
}

const DEFAULT_FORM: LinkFormData = {
  longUrl: '',
  title: '',
  customSlug: '',
  tags: [],
  maxVisits: '',
  validSince: null,
  validUntil: null,
}

export function ShortUrls() {
  const { can } = useAuth()
  const qc = useQueryClient()
  const navigate = useNavigate()
  const [page, setPage] = useState(1)
  const [search, setSearch] = useState('')
  const [status, setStatus] = useState('all')
  const [createOpen, { open: openCreate, close: closeCreate }] = useDisclosure()
  const [editLink, setEditLink] = useState<ShortURL | null>(null)
  const [confirmDelete, setConfirmDelete] = useState<ShortURL | null>(null)
  const [confirmDeactivate, setConfirmDeactivate] = useState<ShortURL | null>(null)
  const [form, setForm] = useState<LinkFormData>(DEFAULT_FORM)
  const [formError, setFormError] = useState('')

  const debouncedSearch = search

  const { data, isLoading } = useQuery({
    queryKey: ['links', page, debouncedSearch, status],
    queryFn: () => listLinks({ page, itemsPerPage: 20, searchTerm: debouncedSearch || undefined, status: status !== 'all' ? status : undefined }),
  })

  const { data: tagsData } = useQuery({ queryKey: ['tags'], queryFn: getTags })
  const tagOptions = tagsData?.map((t) => ({ value: t.tag, label: t.tag })) ?? []

  const createMutation = useMutation({
    mutationFn: createLink,
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['links'] })
      closeCreate()
      setForm(DEFAULT_FORM)
      notifications.show({ color: 'teal', message: 'Ссылка создана' })
    },
  })

  const updateMutation = useMutation({
    mutationFn: ({ shortCode, payload }: { shortCode: string; payload: CreateShortURLPayload }) =>
      updateLink(shortCode, payload),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['links'] })
      setEditLink(null)
      notifications.show({ color: 'teal', message: 'Ссылка обновлена' })
    },
  })

  const deleteMutation = useMutation({
    mutationFn: deleteLink,
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['links'] })
      setConfirmDelete(null)
      notifications.show({ color: 'teal', message: 'Ссылка удалена' })
    },
  })

  const deactivateMutation = useMutation({
    mutationFn: deactivateLink,
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['links'] })
      setConfirmDeactivate(null)
    },
  })

  const activateMutation = useMutation({
    mutationFn: activateLink,
    onSuccess: () => qc.invalidateQueries({ queryKey: ['links'] }),
  })

  const validateAndSubmit = () => {
    if (!form.longUrl.match(/^https?:\/\//)) {
      setFormError('Введите корректный URL (http:// или https://)')
      return
    }
    if (form.validSince && form.validUntil && form.validUntil <= form.validSince) {
      setFormError('validUntil должен быть позже validSince')
      return
    }
    setFormError('')
    const payload: CreateShortURLPayload = {
      longUrl: form.longUrl,
      ...(form.title && { title: form.title }),
      ...(form.customSlug && { customSlug: form.customSlug }),
      ...(form.tags.length && { tags: form.tags }),
      ...(form.maxVisits !== '' && { maxVisits: Number(form.maxVisits) }),
      ...(form.validSince && { validSince: form.validSince.toISOString() }),
      ...(form.validUntil && { validUntil: form.validUntil.toISOString() }),
    }
    if (editLink) {
      updateMutation.mutate({ shortCode: editLink.shortCode, payload })
    } else {
      createMutation.mutate(payload)
    }
  }

  const openEdit = (link: ShortURL) => {
    setEditLink(link)
    setForm({
      longUrl: link.longUrl,
      title: link.title ?? '',
      customSlug: '',
      tags: link.tags ?? [],
      maxVisits: link.maxVisits ?? '',
      validSince: link.validSince ? new Date(link.validSince) : null,
      validUntil: link.validUntil ? new Date(link.validUntil) : null,
    })
    setFormError('')
  }

  const pagination = data?.shortUrls.pagination
  const links = data?.shortUrls.data ?? []

  return (
    <Stack gap="md">
      <Group justify="space-between">
        <Title order={2}>Мои ссылки</Title>
        {can('canCreateLinks') && (
          <Button leftSection={<IconPlus size={16} />} onClick={openCreate}>
            Создать ссылку
          </Button>
        )}
      </Group>

      <Group>
        <TextInput
          placeholder="Поиск..."
          leftSection={<IconSearch size={16} />}
          value={search}
          onChange={(e) => { setSearch(e.currentTarget.value); setPage(1) }}
          style={{ flex: 1 }}
        />
        <SegmentedControl
          data={[
            { label: 'Все', value: 'all' },
            { label: 'Активные', value: 'active' },
            { label: 'Неактивные', value: 'inactive' },
          ]}
          value={status}
          onChange={(v) => { setStatus(v); setPage(1) }}
          size="sm"
        />
      </Group>

      {isLoading ? (
        <Skeleton h={400} radius="md" />
      ) : links.length === 0 ? (
        <EmptyState
          title="Ссылок не найдено"
          action={
            can('canCreateLinks') ? (
              <Button size="sm" onClick={openCreate}>Создать ссылку</Button>
            ) : undefined
          }
        />
      ) : (
        <>
          <Table striped highlightOnHover withTableBorder withColumnBorders>
            <Table.Thead>
              <Table.Tr>
                <Table.Th>Ссылка</Table.Th>
                <Table.Th>Назначение</Table.Th>
                <Table.Th>Теги</Table.Th>
                <Table.Th>Создана</Table.Th>
                <Table.Th>Лимит</Table.Th>
                <Table.Th>Переходов</Table.Th>
                <Table.Th>Статус</Table.Th>
                <Table.Th>Действия</Table.Th>
              </Table.Tr>
            </Table.Thead>
            <Table.Tbody>
              {links.map((link) => (
                <Table.Tr key={link.shortCode}>
                  <Table.Td>
                    <Group gap={4}>
                      <Text
                        size="sm"
                        fw={500}
                        c="blue"
                        style={{ cursor: 'pointer' }}
                        onClick={() => navigate(`/links/${link.shortCode}`)}
                      >
                        {link.shortCode}
                      </Text>
                      <CopyButton value={link.shortUrl} />
                    </Group>
                    <Text size="xs" c="dimmed">{link.shortUrl}</Text>
                  </Table.Td>
                  <Table.Td maw={240}>
                    <Tooltip label={link.longUrl} multiline maw={300}>
                      <Text size="sm" truncate>{link.title || link.longUrl.slice(0, 60)}</Text>
                    </Tooltip>
                  </Table.Td>
                  <Table.Td>
                    <Group gap={4}>
                      {link.tags.slice(0, 3).map((t) => <Badge key={t} size="xs" variant="outline">{t}</Badge>)}
                      {link.tags.length > 3 && <Badge size="xs" color="gray">+{link.tags.length - 3}</Badge>}
                    </Group>
                  </Table.Td>
                  <Table.Td><Text size="sm">{formatDate(link.dateCreated)}</Text></Table.Td>
                  <Table.Td><Text size="sm">{link.maxVisits ?? '∞'}</Text></Table.Td>
                  <Table.Td>{link.visitsSummary.total}</Table.Td>
                  <Table.Td><StatusBadge active={link.isActive !== false} /></Table.Td>
                  <Table.Td>
                    <Group gap={4}>
                      {(can('canEditOwnLinks') || can('canEditAllLinks')) && (
                        <Tooltip label="Редактировать">
                          <ActionIcon variant="subtle" size="sm" onClick={() => openEdit(link)}>
                            <IconEdit size={14} />
                          </ActionIcon>
                        </Tooltip>
                      )}
                      {link.isActive !== false && (can('canDeactivateOwnLinks') || can('canDeactivateAllLinks')) && (
                        <Tooltip label="Деактивировать">
                          <ActionIcon variant="subtle" size="sm" color="orange" onClick={() => setConfirmDeactivate(link)}>
                            <IconPlayerPause size={14} />
                          </ActionIcon>
                        </Tooltip>
                      )}
                      {link.isActive === false && (can('canReactivateOwnLinks') || can('canReactivateAllLinks')) && (
                        <Tooltip label="Активировать">
                          <ActionIcon variant="subtle" size="sm" color="teal" onClick={() => activateMutation.mutate(link.shortCode)}>
                            <IconPlayerPlay size={14} />
                          </ActionIcon>
                        </Tooltip>
                      )}
                      {(can('canDeleteOwnLinksPermanently') || can('canDeleteAllLinksPermanently')) && (
                        <Tooltip label="Удалить">
                          <ActionIcon variant="subtle" size="sm" color="red" onClick={() => setConfirmDelete(link)}>
                            <IconTrash size={14} />
                          </ActionIcon>
                        </Tooltip>
                      )}
                    </Group>
                  </Table.Td>
                </Table.Tr>
              ))}
            </Table.Tbody>
          </Table>
          {pagination && pagination.pagesCount > 1 && (
            <Group justify="space-between">
              <Text size="sm" c="dimmed">
                Записи {(page - 1) * 20 + 1}–{Math.min(page * 20, pagination.totalItems)} из {pagination.totalItems}
              </Text>
              <Pagination total={pagination.pagesCount} value={page} onChange={setPage} size="sm" />
            </Group>
          )}
        </>
      )}

      {/* Модалка создания / редактирования */}
      <Modal
        opened={createOpen || editLink !== null}
        onClose={() => { closeCreate(); setEditLink(null); setForm(DEFAULT_FORM); setFormError('') }}
        title={editLink ? 'Редактировать ссылку' : 'Создать ссылку'}
        size="lg"
      >
        <Stack>
          <TextInput
            label="Long URL"
            placeholder="https://example.com/..."
            required
            value={form.longUrl}
            onChange={(e) => setForm((f) => ({ ...f, longUrl: e.currentTarget.value }))}
          />
          <TextInput
            label="Заголовок"
            value={form.title}
            onChange={(e) => setForm((f) => ({ ...f, title: e.currentTarget.value }))}
          />
          {!editLink && can('canCreateWithCustomSlug') && (
            <TextInput
              label="Кастомный слаг"
              value={form.customSlug}
              onChange={(e) => setForm((f) => ({ ...f, customSlug: e.currentTarget.value }))}
            />
          )}
          <MultiSelect
            label="Теги"
            data={tagOptions}
            value={form.tags}
            onChange={(v) => setForm((f) => ({ ...f, tags: v }))}
            searchable
            creatable
          />
          <NumberInput
            label="Макс. переходов"
            min={1}
            value={form.maxVisits}
            onChange={(v) => setForm((f) => ({ ...f, maxVisits: v === '' ? '' : Number(v) }))}
            placeholder="Без ограничения"
          />
          <DateTimePicker
            label="Действительна с"
            value={form.validSince}
            onChange={(v) => setForm((f) => ({ ...f, validSince: v }))}
            clearable
          />
          <DateTimePicker
            label="Действительна по"
            value={form.validUntil}
            onChange={(v) => setForm((f) => ({ ...f, validUntil: v }))}
            minDate={form.validSince ?? undefined}
            clearable
          />
          {formError && <Text c="red" size="sm">{formError}</Text>}
          <Group justify="flex-end">
            <Button variant="default" onClick={() => { closeCreate(); setEditLink(null) }}>Отмена</Button>
            <Button
              loading={createMutation.isPending || updateMutation.isPending}
              onClick={validateAndSubmit}
            >
              Сохранить
            </Button>
          </Group>
        </Stack>
      </Modal>

      <ConfirmDialog
        opened={confirmDelete !== null}
        onClose={() => setConfirmDelete(null)}
        onConfirm={() => confirmDelete && deleteMutation.mutate(confirmDelete.shortCode)}
        title="Удалить ссылку?"
        message={`Ссылка ${confirmDelete?.shortCode} будет удалена безвозвратно.`}
        confirmLabel="Удалить"
        loading={deleteMutation.isPending}
        danger
      />

      <ConfirmDialog
        opened={confirmDeactivate !== null}
        onClose={() => setConfirmDeactivate(null)}
        onConfirm={() => confirmDeactivate && deactivateMutation.mutate(confirmDeactivate.shortCode)}
        title="Деактивировать ссылку?"
        message={`Ссылка ${confirmDeactivate?.shortCode} будет деактивирована. Пользователи получат ошибку при переходе.`}
        confirmLabel="Деактивировать"
        loading={deactivateMutation.isPending}
      />
    </Stack>
  )
}
