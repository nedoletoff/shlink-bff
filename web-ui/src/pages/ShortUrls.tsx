import { useCallback, useEffect, useState } from 'react';
import {
  Stack, Title, Button, TextInput, Table, ActionIcon, Group,
  Badge, Text, Loader, Center, Modal, Tooltip, Pagination,
  Anchor, CopyButton,
} from '@mantine/core';
import { useDebouncedValue, useDisclosure } from '@mantine/hooks';
import { notifications } from '@mantine/notifications';
import {
  IconPlus, IconTrash, IconSearch, IconEdit,
  IconCopy, IconCheck,
} from '@tabler/icons-react';
import { api } from '../api/client';
import { useAuth } from '../contexts/AuthContext';
import { ConfirmDialog } from '../components/ui/ConfirmDialog';
import { formatDate } from '../utils/date';
import { useIsAdmin } from '../contexts/AuthContext';
import { useNavigate } from 'react-router-dom';
import type { Pagination as PaginationInfo, ShortURL, ShortURLsListResponse } from '../types/api';

const ITEMS_PER_PAGE = 20;

export function ShortUrls() {
  const { user }  = useAuth();
  const isAdmin   = useIsAdmin();
  const navigate  = useNavigate();
  const [urls,         setUrls]         = useState<ShortURL[]>([]);
  const [pagination,   setPagination]   = useState<PaginationInfo | null>(null);
  const [loading,      setLoading]      = useState(true);
  const [search,       setSearch]       = useState('');
  const [page,         setPage]         = useState(1);
  const [deleteTarget, setDeleteTarget] = useState<ShortURL | null>(null);
  const [editTarget,   setEditTarget]   = useState<ShortURL | null>(null);
  const [createOpen, { open: openCreate, close: closeCreate }] = useDisclosure(false);

  const [debouncedSearch] = useDebouncedValue(search, 350);

  const fetchUrls = useCallback(() => {
    setLoading(true);
    api.get<ShortURLsListResponse>('/api/shlink/short-urls', {
      params: {
        searchTerm:   debouncedSearch || undefined,
        page,
        itemsPerPage: ITEMS_PER_PAGE,
      },
    })
      .then(r => {
        setUrls(r.shortUrls.data);
        setPagination(r.shortUrls.pagination);
      })
      .catch(() => { setUrls([]); setPagination(null); })
      .finally(() => setLoading(false));
  }, [debouncedSearch, page]);

  useEffect(() => { fetchUrls(); }, [fetchUrls]);
  useEffect(() => { setPage(1); }, [debouncedSearch]);

  const handleDelete = async () => {
    if (!deleteTarget) return;
    try {
      await api.delete(`/api/shlink/short-urls/${deleteTarget.shortCode}`);
      notifications.show({ message: 'Ссылка удалена', color: 'green' });
      setDeleteTarget(null);
      fetchUrls();
    } catch {
      /* APIError уже показан через notifications */
    }
  };

  return (
    <Stack gap="lg">
      <Group justify="space-between">
        <Title order={2}>Мои ссылки</Title>
        {user?.permissions.canCreateShortUrl && (
          <Button leftSection={<IconPlus size={16} />} onClick={openCreate}>
            Создать ссылку
          </Button>
        )}
      </Group>

      <TextInput
        placeholder="Поиск по коду, URL, названию..."
        leftSection={<IconSearch size={16} />}
        value={search}
        onChange={e => setSearch(e.currentTarget.value)}
        style={{ maxWidth: 400 }}
      />

      {loading ? (
        <Center h={200}><Loader /></Center>
      ) : (
        <Table highlightOnHover verticalSpacing="xs">
          <Table.Thead>
            <Table.Tr>
              <Table.Th>Код</Table.Th>
              <Table.Th>Куда ведёт</Table.Th>
              <Table.Th>Название</Table.Th>
              {isAdmin && <Table.Th>Создатель</Table.Th>}
              <Table.Th>Статус</Table.Th>
              <Table.Th style={{ width: 80 }}>Переходы</Table.Th>
              <Table.Th>Создана</Table.Th>
              <Table.Th style={{ width: 80 }}>Действия</Table.Th>
            </Table.Tr>
          </Table.Thead>
          <Table.Tbody>
            {urls.map(url => (
              <Table.Tr
                key={url.shortCode}
                style={{ cursor: 'pointer', opacity: url.isActive === false ? 0.55 : 1 }}
                onClick={e => {
                  // не открывать детали при клике на кнопку
                  if ((e.target as HTMLElement).closest('button,a')) return;
                  navigate(`/urls/${url.shortCode}`);
                }}
              >
                <Table.Td>
                  <Group gap={4} wrap="nowrap">
                    <Text ff="monospace" size="sm">{url.shortCode}</Text>
                    <CopyButton value={url.shortUrl}>
                      {({ copied, copy }) => (
                        <Tooltip label={copied ? 'Скопировано' : 'Копировать'} withArrow>
                          <ActionIcon
                            size="xs" variant="subtle"
                            color={copied ? 'teal' : 'gray'}
                            onClick={e => { e.stopPropagation(); copy(); }}
                          >
                            {copied ? <IconCheck size={12} /> : <IconCopy size={12} />}
                          </ActionIcon>
                        </Tooltip>
                      )}
                    </CopyButton>
                  </Group>
                </Table.Td>

                <Table.Td style={{ maxWidth: 220, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                  <Anchor
                    href={url.longUrl}
                    target="_blank"
                    rel="noopener noreferrer"
                    fz="sm"
                    onClick={e => e.stopPropagation()}
                  >
                    {url.longUrl}
                  </Anchor>
                </Table.Td>

                <Table.Td>
                  <Text size="sm" c={url.title ? undefined : 'dimmed'}>
                    {url.title || '—'}
                  </Text>
                </Table.Td>

                {isAdmin && (
                  <Table.Td>
                    <Text size="sm" c="dimmed">{url.ownerUsername ?? '—'}</Text>
                  </Table.Td>
                )}

                <Table.Td>
                  <Badge
                    size="xs"
                    variant="light"
                    color={url.isActive === false ? 'red' : 'green'}
                  >
                    {url.isActive === false ? 'неактивна' : 'активна'}
                  </Badge>
                </Table.Td>

                <Table.Td>{url.visitsSummary.total.toLocaleString('ru')}</Table.Td>

                <Table.Td>
                  <Text size="sm">{formatDate(url.dateCreated)}</Text>
                </Table.Td>

                <Table.Td>
                  <Group gap={4}>
                    {user?.permissions.canEditOwnLinks && (
                      <ActionIcon
                        variant="subtle"
                        onClick={e => { e.stopPropagation(); setEditTarget(url); }}
                      >
                        <IconEdit size={16} />
                      </ActionIcon>
                    )}
                    {user?.permissions.canDeleteOwnLinks && (
                      <ActionIcon
                        color="red" variant="subtle"
                        onClick={e => { e.stopPropagation(); setDeleteTarget(url); }}
                      >
                        <IconTrash size={16} />
                      </ActionIcon>
                    )}
                  </Group>
                </Table.Td>
              </Table.Tr>
            ))}
            {urls.length === 0 && (
              <Table.Tr>
                <Table.Td colSpan={isAdmin ? 8 : 7}>
                  <Center p="xl">
                    <Text c="dimmed">Ничего не найдено</Text>
                  </Center>
                </Table.Td>
              </Table.Tr>
            )}
          </Table.Tbody>
        </Table>
      )}

      {pagination && pagination.pagesCount > 1 && (
        <Group justify="space-between" align="center">
          <Text size="sm" c="dimmed">
            Всего: {pagination.totalItems}
          </Text>
          <Pagination
            total={pagination.pagesCount}
            value={page}
            onChange={setPage}
            siblings={1}
          />
        </Group>
      )}

      <CreateShortUrlModal
        opened={createOpen}
        onClose={closeCreate}
        onCreated={() => { closeCreate(); fetchUrls(); }}
        slugPrefix={user?.features.userSlugPrefixEnabled ? user.slugPrefix : undefined}
      />

      {editTarget && (
        <EditShortUrlModal
          url={editTarget}
          onClose={() => setEditTarget(null)}
          onSaved={() => { setEditTarget(null); fetchUrls(); }}
        />
      )}

      <ConfirmDialog
        opened={!!deleteTarget}
        title="Удалить ссылку?"
        message={`Ссылка "${deleteTarget?.shortCode}" будет удалена безвозвратно. Все переходы перестанут работать.`}
        confirmLabel="Удалить"
        confirmColor="red"
        onConfirm={handleDelete}
        onCancel={() => setDeleteTarget(null)}
      />
    </Stack>
  );
}

// ── Форма создания ────────────────────────────────────────────────────────────
function CreateShortUrlModal({
  opened, onClose, onCreated, slugPrefix,
}: {
  opened: boolean; onClose: () => void; onCreated: () => void; slugPrefix?: string;
}) {
  const [longUrl,    setLongUrl]    = useState('');
  const [customSlug, setCustomSlug] = useState(slugPrefix ?? '');
  const [title,      setTitle]      = useState('');
  const [loading,    setLoading]    = useState(false);

  const handleSubmit = async () => {
    if (!longUrl.trim()) return;
    setLoading(true);
    try {
      await api.post('/api/shlink/short-urls', {
        longUrl:    longUrl.trim(),
        customSlug: customSlug.trim() || undefined,
        title:      title.trim() || undefined,
      });
      notifications.show({ message: 'Ссылка создана', color: 'green' });
      onCreated();
    } catch {
      // ошибка уже отображена через api.post → notifications
    } finally {
      setLoading(false);
    }
  };

  return (
    <Modal opened={opened} onClose={onClose} title="Создать ссылку" size="md">
      <Stack gap="sm">
        <TextInput
          label="Длинная ссылка"
          description="Адрес страницы, на которую будет вести короткая ссылка"
          required
          placeholder="https://example.com/long-path"
          value={longUrl}
          onChange={e => setLongUrl(e.currentTarget.value)}
        />
        <TextInput
          label="Название"
          description="Как вы будете её узнавать в списке"
          placeholder="Например: Презентация Q3"
          value={title}
          onChange={e => setTitle(e.currentTarget.value)}
        />
        <TextInput
          label={slugPrefix ? `Кастомная ссылка (префикс: ${slugPrefix})` : 'Кастомная ссылка'}
          description="Придумайте короткое слово, например: konferencia"
          placeholder={slugPrefix ? `${slugPrefix}-...` : 'авто-генерация'}
          value={customSlug}
          onChange={e => setCustomSlug(e.currentTarget.value)}
        />
        <Group justify="flex-end" mt="md">
          <Button variant="default" onClick={onClose}>Отмена</Button>
          <Button onClick={handleSubmit} loading={loading} disabled={!longUrl.trim()}>
            Создать ссылку
          </Button>
        </Group>
      </Stack>
    </Modal>
  );
}

// ── Форма редактирования ──────────────────────────────────────────────────────
function EditShortUrlModal({
  url, onClose, onSaved,
}: {
  url: ShortURL; onClose: () => void; onSaved: () => void;
}) {
  const [longUrl, setLongUrl] = useState(url.longUrl);
  const [title,   setTitle]   = useState(url.title ?? '');
  const [loading, setLoading] = useState(false);

  const handleSave = async () => {
    setLoading(true);
    try {
      await api.patch(`/api/shlink/short-urls/${url.shortCode}`, {
        longUrl: longUrl.trim(),
        title:   title.trim() || null,
      });
      notifications.show({ message: 'Ссылка обновлена', color: 'green' });
      onSaved();
    } catch {
      // handled by api client
    } finally {
      setLoading(false);
    }
  };

  return (
    <Modal
      opened={true}
      onClose={onClose}
      title={`Редактировать: ${url.shortCode}`}
      size="md"
    >
      <Stack gap="sm">
        <TextInput
          label="Длинная ссылка"
          required
          value={longUrl}
          onChange={e => setLongUrl(e.currentTarget.value)}
        />
        <TextInput
          label="Название"
          value={title}
          onChange={e => setTitle(e.currentTarget.value)}
        />
        <Group justify="flex-end" mt="md">
          <Button variant="default" onClick={onClose}>Отмена</Button>
          <Button onClick={handleSave} loading={loading}>Сохранить</Button>
        </Group>
      </Stack>
    </Modal>
  );
}
