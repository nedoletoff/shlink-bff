import { useCallback, useEffect, useState } from 'react';
import {
  Stack, Title, Button, TextInput, Table, ActionIcon, Group,
  Badge, Text, Loader, Center, Modal, Tooltip, Pagination,
  Anchor, CopyButton, Box, Card, SegmentedControl,
} from '@mantine/core';
import { useDebouncedValue, useDisclosure } from '@mantine/hooks';
import { notifications } from '@mantine/notifications';
import {
  IconPlus, IconTrash, IconSearch, IconEdit,
  IconCopy, IconCheck, IconBan, IconExternalLink,
  IconPlayerPlay, IconAlertTriangle,
} from '@tabler/icons-react';
import { api } from '../api/client';
import { useAuth } from '../contexts/AuthContext';
import { ConfirmDialog } from '../components/ui/ConfirmDialog';
import { formatDate } from '../utils/date';
import { useIsAdmin } from '../contexts/AuthContext';
import { useNavigate } from 'react-router-dom';
import type { Pagination as PaginationInfo, ShortURL, ShortURLsListResponse } from '../types/api';

const ITEMS_PER_PAGE = 20;

type StatusFilter = 'active' | 'inactive' | 'all';

// ─── Create / Edit modal ─────────────────────────────────────────────────────
function CreateEditModal({
  opened, onClose, onSaved, editTarget,
}: {
  opened:     boolean;
  onClose:    () => void;
  onSaved:    () => void;
  editTarget: ShortURL | null;
}) {
  const { user } = useAuth();
  const canCustomSlug = user?.permissions.canCreateWithCustomSlug ?? false;

  const [longUrl,    setLongUrl]  = useState('');
  const [title,      setTitle]    = useState('');
  const [customSlug, setSlug]     = useState('');
  const [saving,     setSaving]   = useState(false);

  useEffect(() => {
    if (editTarget) {
      setLongUrl(editTarget.longUrl);
      setTitle(editTarget.title ?? '');
      setSlug('');
    } else {
      setLongUrl(''); setTitle(''); setSlug('');
    }
  }, [editTarget, opened]);

  const handleSubmit = async () => {
    if (!longUrl.trim()) return;
    setSaving(true);
    try {
      if (editTarget) {
        await api.patch(`/api/shlink/short-urls/${editTarget.shortCode}`, {
          longUrl: longUrl.trim(),
          title:   title.trim() || undefined,
        });
        notifications.show({ message: 'Ссылка обновлена', color: 'green' });
      } else {
        await api.post('/api/shlink/short-urls', {
          longUrl:    longUrl.trim(),
          title:      title.trim() || undefined,
          ...(canCustomSlug && customSlug.trim() ? { customSlug: customSlug.trim() } : {}),
        });
        notifications.show({ message: 'Ссылка создана', color: 'green' });
      }
      onSaved();
      onClose();
    } catch {
      /* APIError уже показан через notifications */
    } finally {
      setSaving(false);
    }
  };

  return (
    <Modal
      opened={opened}
      onClose={onClose}
      title={editTarget ? 'Редактировать ссылку' : 'Создать ссылку'}
      size="md"
    >
      <Stack gap="sm">
        <TextInput
          label="Длинная ссылка"
          placeholder="https://example.com/very/long/path"
          value={longUrl}
          onChange={e => setLongUrl(e.currentTarget.value)}
          required
          data-autofocus
        />
        <TextInput
          label="Название (опционально)"
          placeholder="Как вы будете её узнавать"
          value={title}
          onChange={e => setTitle(e.currentTarget.value)}
        />
        {!editTarget && canCustomSlug && (
          <TextInput
            label="Кастомный слаг (опционально)"
            placeholder="my-link"
            value={customSlug}
            onChange={e => setSlug(e.currentTarget.value)}
            description="Оставьте пустым — сгенерируется автоматически"
          />
        )}
        <Group justify="flex-end" mt="sm">
          <Button variant="default" onClick={onClose}>Отмена</Button>
          <Button onClick={handleSubmit} loading={saving}>
            {editTarget ? 'Сохранить' : 'Создать'}
          </Button>
        </Group>
      </Stack>
    </Modal>
  );
}

// ─── Main page ────────────────────────────────────────────────────────────────
export function ShortUrls() {
  const { user } = useAuth();
  const isAdmin  = useIsAdmin();
  const navigate = useNavigate();

  const [urls,           setUrls]           = useState<ShortURL[]>([]);
  const [pagination,     setPagination]     = useState<PaginationInfo | null>(null);
  const [loading,        setLoading]        = useState(true);
  const [search,         setSearch]         = useState('');
  const [page,           setPage]           = useState(1);
  const [statusFilter,   setStatusFilter]   = useState<StatusFilter>('active');
  const [deactivateTarget, setDeactivateTarget] = useState<ShortURL | null>(null);
  const [activateTarget,   setActivateTarget]   = useState<ShortURL | null>(null);
  const [deleteTarget,     setDeleteTarget]     = useState<ShortURL | null>(null);
  const [editTarget,       setEditTarget]       = useState<ShortURL | null>(null);
  const [createOpen, { open: openCreate, close: closeCreate }] = useDisclosure(false);

  const [debouncedSearch] = useDebouncedValue(search, 350);

  const perms = user?.permissions;
  const canDeactivate  = perms?.canDeactivateOwnLinks  || perms?.canDeactivateAllLinks;
  const canReactivate  = perms?.canReactivateOwnLinks  || perms?.canReactivateAllLinks;
  const canPermDelete  = perms?.canDeleteOwnLinksPermanently || perms?.canDeleteAllLinksPermanently;

  const fetchUrls = useCallback(() => {
    setLoading(true);
    api.get<ShortURLsListResponse>('/api/shlink/short-urls', {
      params: {
        searchTerm:   debouncedSearch || undefined,
        page,
        itemsPerPage: ITEMS_PER_PAGE,
        status:       statusFilter,
      },
    })
      .then(r => {
        setUrls(r.shortUrls.data);
        setPagination(r.shortUrls.pagination);
      })
      .catch(() => { setUrls([]); setPagination(null); })
      .finally(() => setLoading(false));
  }, [debouncedSearch, page, statusFilter]);

  useEffect(() => { fetchUrls(); }, [fetchUrls]);
  useEffect(() => { setPage(1); }, [debouncedSearch, statusFilter]);

  const handleDeactivate = async () => {
    if (!deactivateTarget) return;
    try {
      await api.post(`/api/shlink/short-urls/${deactivateTarget.shortCode}/deactivate`, {});
      notifications.show({ message: 'Ссылка деактивирована', color: 'orange' });
      setDeactivateTarget(null);
      fetchUrls();
    } catch { /* already shown */ }
  };

  const handleActivate = async () => {
    if (!activateTarget) return;
    try {
      await api.post(`/api/shlink/short-urls/${activateTarget.shortCode}/activate`, {});
      notifications.show({ message: 'Ссылка активирована', color: 'green' });
      setActivateTarget(null);
      fetchUrls();
    } catch { /* already shown */ }
  };

  const handlePermanentDelete = async () => {
    if (!deleteTarget) return;
    try {
      await api.delete(`/api/shlink/short-urls/${deleteTarget.shortCode}/permanent`);
      notifications.show({ message: 'Ссылка удалена безвозвратно', color: 'red' });
      setDeleteTarget(null);
      fetchUrls();
    } catch { /* already shown */ }
  };

  const totalPages = pagination ? pagination.pagesCount : 1;

  return (
    <Stack gap="lg">
      <Group justify="space-between" wrap="nowrap">
        <div>
          <Title order={2} fw={700}>Мои ссылки</Title>
          {pagination && (
            <Text size="sm" c="dimmed">{pagination.totalItems} ссылок</Text>
          )}
        </div>
        {user?.permissions.canCreateShortUrl && (
          <Button size="sm" leftSection={<IconPlus size={14} />} onClick={openCreate}>
            Создать
          </Button>
        )}
      </Group>

      {/* Filters */}
      <Group gap="sm" wrap="nowrap">
        <TextInput
          placeholder="Поиск по ссылке или коду…"
          leftSection={<IconSearch size={14} />}
          value={search}
          onChange={e => setSearch(e.currentTarget.value)}
          size="sm"
          style={{ flex: 1 }}
        />
        <SegmentedControl
          size="sm"
          value={statusFilter}
          onChange={v => setStatusFilter(v as StatusFilter)}
          data={[
            { label: 'Активные',       value: 'active'   },
            { label: 'Деактивиров.',   value: 'inactive' },
            { label: 'Все',            value: 'all'      },
          ]}
        />
      </Group>

      {/* Table */}
      <Card withBorder p={0} radius="md" style={{ overflow: 'hidden' }}>
        {loading ? (
          <Center h={200}><Loader size="sm" /></Center>
        ) : urls.length === 0 ? (
          <Center h={120}>
            <Text c="dimmed" size="sm">Ничего не найдено</Text>
          </Center>
        ) : (
          <Box style={{ overflowX: 'auto' }}>
            <Table highlightOnHover withRowBorders stickyHeader>
              <Table.Thead>
                <Table.Tr>
                  <Table.Th style={{ width: 120 }}>Код</Table.Th>
                  <Table.Th>Назначение</Table.Th>
                  <Table.Th style={{ width: 100 }}>Создана</Table.Th>
                  <Table.Th style={{ width: 80 }} ta="right">Клики</Table.Th>
                  {isAdmin && <Table.Th style={{ width: 110 }}>Владелец</Table.Th>}
                  <Table.Th style={{ width: 90 }}>Статус</Table.Th>
                  <Table.Th style={{ width: 100 }} />
                </Table.Tr>
              </Table.Thead>
              <Table.Tbody>
                {urls.map(url => {
                  const inactive = url.isActive === false;
                  return (
                    <Table.Tr
                      key={url.shortCode}
                      style={{ opacity: inactive ? 0.6 : 1 }}
                    >
                      {/* Code + copy */}
                      <Table.Td>
                        <Group gap={4} wrap="nowrap">
                          <CopyButton value={url.shortUrl}>
                            {({ copy, copied }) => (
                              <Tooltip label={copied ? 'Скопировано!' : 'Копировать'}>
                                <ActionIcon
                                  variant="subtle" size="xs" color={copied ? 'teal' : 'gray'}
                                  onClick={e => { e.stopPropagation(); copy(); }}
                                >
                                  {copied ? <IconCheck size={12} /> : <IconCopy size={12} />}
                                </ActionIcon>
                              </Tooltip>
                            )}
                          </CopyButton>
                          <Anchor
                            size="sm" ff="monospace"
                            onClick={e => { e.stopPropagation(); navigate(`/links/${url.shortCode}`); }}
                          >
                            {url.shortCode}
                          </Anchor>
                        </Group>
                      </Table.Td>

                      {/* Destination */}
                      <Table.Td>
                        <Group gap={4} wrap="nowrap">
                          <Text size="sm" truncate maw={280} title={url.longUrl}>
                            {url.title || url.longUrl}
                          </Text>
                          <Tooltip label={url.longUrl}>
                            <ActionIcon
                              component="a" href={url.longUrl} target="_blank"
                              variant="subtle" size="xs" color="gray"
                              onClick={e => e.stopPropagation()}
                            >
                              <IconExternalLink size={12} />
                            </ActionIcon>
                          </Tooltip>
                        </Group>
                      </Table.Td>

                      {/* Created */}
                      <Table.Td>
                        <Text size="xs" c="dimmed">
                          {url.dateCreated ? formatDate(url.dateCreated) : '—'}
                        </Text>
                      </Table.Td>

                      {/* Clicks */}
                      <Table.Td ta="right">
                        <Text size="sm" fw={600} ff="monospace">
                          {url.visitsSummary?.total ?? 0}
                        </Text>
                      </Table.Td>

                      {/* Owner (admin) */}
                      {isAdmin && (
                        <Table.Td>
                          <Text size="xs" c="dimmed">{url.ownerUsername ?? '—'}</Text>
                        </Table.Td>
                      )}

                      {/* Status */}
                      <Table.Td>
                        {inactive ? (
                          <Badge size="xs" variant="light" color="orange" leftSection={<IconBan size={10} />}>
                            Деактив.
                          </Badge>
                        ) : (
                          <Badge size="xs" variant="light" color="green">Активна</Badge>
                        )}
                      </Table.Td>

                      {/* Actions */}
                      <Table.Td>
                        <Group gap={4} justify="flex-end" wrap="nowrap">
                          {user?.permissions.canEditOwnLinks && !inactive && (
                            <Tooltip label="Редактировать">
                              <ActionIcon
                                variant="subtle" size="sm" color="blue"
                                onClick={e => { e.stopPropagation(); setEditTarget(url); }}
                              >
                                <IconEdit size={14} />
                              </ActionIcon>
                            </Tooltip>
                          )}
                          {canDeactivate && !inactive && (
                            <Tooltip label="Деактивировать">
                              <ActionIcon
                                variant="subtle" size="sm" color="orange"
                                onClick={e => { e.stopPropagation(); setDeactivateTarget(url); }}
                              >
                                <IconBan size={14} />
                              </ActionIcon>
                            </Tooltip>
                          )}
                          {canReactivate && inactive && (
                            <Tooltip label="Активировать">
                              <ActionIcon
                                variant="subtle" size="sm" color="green"
                                onClick={e => { e.stopPropagation(); setActivateTarget(url); }}
                              >
                                <IconPlayerPlay size={14} />
                              </ActionIcon>
                            </Tooltip>
                          )}
                          {canPermDelete && (
                            <Tooltip label="Удалить безвозвратно">
                              <ActionIcon
                                variant="subtle" size="sm" color="red"
                                onClick={e => { e.stopPropagation(); setDeleteTarget(url); }}
                              >
                                <IconTrash size={14} />
                              </ActionIcon>
                            </Tooltip>
                          )}
                        </Group>
                      </Table.Td>
                    </Table.Tr>
                  );
                })}
              </Table.Tbody>
            </Table>
          </Box>
        )}
      </Card>

      {/* Pagination */}
      {totalPages > 1 && (
        <Group justify="center">
          <Pagination value={page} onChange={setPage} total={totalPages} size="sm" />
        </Group>
      )}

      {/* Modals */}
      <CreateEditModal
        opened={createOpen || editTarget !== null}
        onClose={() => { closeCreate(); setEditTarget(null); }}
        onSaved={fetchUrls}
        editTarget={editTarget}
      />

      <ConfirmDialog
        opened={deactivateTarget !== null}
        title="Деактивировать ссылку?"
        message={`Ссылка «${deactivateTarget?.shortCode}» перестанет работать. Историю переходов можно восстановить активацией.`}
        onConfirm={handleDeactivate}
        onCancel={() => setDeactivateTarget(null)}
      />

      <ConfirmDialog
        opened={activateTarget !== null}
        title="Активировать ссылку?"
        message={`Ссылка «${activateTarget?.shortCode}» снова начнёт работать.`}
        onConfirm={handleActivate}
        onCancel={() => setActivateTarget(null)}
      />

      <ConfirmDialog
        opened={deleteTarget !== null}
        title="Удалить ссылку безвозвратно?"
        message={`Ссылка «${deleteTarget?.shortCode}» и все её переходы будут удалены навсегда. Это действие нельзя отменить.`}
        confirmColor="red"
        onConfirm={handlePermanentDelete}
        onCancel={() => setDeleteTarget(null)}
      />
    </Stack>
  );
}
