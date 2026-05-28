import { useEffect, useState } from 'react';
import {
  Stack, Title, Button, TextInput, Table, ActionIcon, Group,
  Badge, Text, Loader, Center, Modal, Tooltip,
} from '@mantine/core';
import { useDisclosure } from '@mantine/hooks';
import { notifications } from '@mantine/notifications';
import { IconPlus, IconTrash, IconSearch, IconExternalLink, IconEdit } from '@tabler/icons-react';
import { api } from '../api/client';
import { useAuth } from '../contexts/AuthContext';
import { ConfirmDialog } from '../components/ui/ConfirmDialog';
import type { ShortURL, ShortURLsListResponse } from '../types/api';

export function ShortUrls() {
  const { user }  = useAuth();
  const [urls,          setUrls]          = useState<ShortURL[]>([]);
  const [loading,       setLoading]       = useState(true);
  const [search,        setSearch]        = useState('');
  const [deleteTarget,  setDeleteTarget]  = useState<ShortURL | null>(null);
  const [editTarget,    setEditTarget]    = useState<ShortURL | null>(null);
  const [createOpen, { open: openCreate, close: closeCreate }] = useDisclosure(false);

  const fetchUrls = () => {
    setLoading(true);
    api.get<ShortURLsListResponse>('/api/shlink/short-urls')
      .then(r => setUrls(r.shortUrls.data))
      .catch(() => setUrls([]))
      .finally(() => setLoading(false));
  };

  useEffect(() => { fetchUrls(); }, []);

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

  const filtered = urls.filter(u =>
    u.shortCode.toLowerCase().includes(search.toLowerCase()) ||
    u.longUrl.toLowerCase().includes(search.toLowerCase()) ||
    (u.title ?? '').toLowerCase().includes(search.toLowerCase()),
  );

  return (
    <Stack gap="lg">
      <Group justify="space-between">
        <Title order={2}>Короткие ссылки</Title>
        {user?.permissions.canCreateShortUrl && (
          <Button leftSection={<IconPlus size={16} />} onClick={openCreate}>
            Создать
          </Button>
        )}
      </Group>

      <TextInput
        placeholder="Поиск по коду, URL, заголовку..."
        leftSection={<IconSearch size={16} />}
        value={search}
        onChange={e => setSearch(e.currentTarget.value)}
      />

      {loading ? (
        <Center h={200}><Loader /></Center>
      ) : (
        <Table striped highlightOnHover withTableBorder>
          <Table.Thead>
            <Table.Tr>
              <Table.Th>Код</Table.Th>
              <Table.Th>Целевой URL</Table.Th>
              <Table.Th>Теги</Table.Th>
              <Table.Th>Клики</Table.Th>
              <Table.Th>Создана</Table.Th>
              <Table.Th />
            </Table.Tr>
          </Table.Thead>
          <Table.Tbody>
            {filtered.map(url => (
              <Table.Tr key={url.shortCode}>
                <Table.Td>
                  <Group gap={4}>
                    <Text fw={500} ff="monospace">{url.shortCode}</Text>
                    <Tooltip label="Открыть">
                      <ActionIcon
                        size="xs" variant="subtle"
                        component="a" href={url.shortUrl} target="_blank"
                        rel="noopener noreferrer"
                      >
                        <IconExternalLink size={12} />
                      </ActionIcon>
                    </Tooltip>
                  </Group>
                </Table.Td>
                <Table.Td>
                  <Text size="sm" truncate="end" maw={280} title={url.longUrl}>
                    {url.longUrl}
                  </Text>
                </Table.Td>
                <Table.Td>
                  <Group gap={4}>
                    {url.tags.map(t => (
                      <Badge key={t} size="sm" variant="light">{t}</Badge>
                    ))}
                  </Group>
                </Table.Td>
                <Table.Td>{url.visitsSummary.total.toLocaleString('ru')}</Table.Td>
                <Table.Td>
                  <Text size="sm">
                    {new Date(url.dateCreated).toLocaleDateString('ru')}
                  </Text>
                </Table.Td>
                <Table.Td>
                  <Group gap={4}>
                    {user?.permissions.canEditOwnLinks && (
                      <ActionIcon variant="subtle" onClick={() => setEditTarget(url)}>
                        <IconEdit size={16} />
                      </ActionIcon>
                    )}
                    {user?.permissions.canDeleteOwnLinks && (
                      <ActionIcon
                        color="red" variant="subtle"
                        onClick={() => setDeleteTarget(url)}
                      >
                        <IconTrash size={16} />
                      </ActionIcon>
                    )}
                  </Group>
                </Table.Td>
              </Table.Tr>
            ))}
            {filtered.length === 0 && (
              <Table.Tr>
                <Table.Td colSpan={6}>
                  <Center p="xl">
                    <Text c="dimmed">Ничего не найдено</Text>
                  </Center>
                </Table.Td>
              </Table.Tr>
            )}
          </Table.Tbody>
        </Table>
      )}

      {/* Форма создания */}
      <CreateShortUrlModal
        opened={createOpen}
        onClose={closeCreate}
        onCreated={() => { closeCreate(); fetchUrls(); }}
        slugPrefix={user?.features.userSlugPrefixEnabled ? user.slugPrefix : undefined}
      />

      {/* Форма редактирования */}
      {editTarget && (
        <EditShortUrlModal
          url={editTarget}
          onClose={() => setEditTarget(null)}
          onSaved={() => { setEditTarget(null); fetchUrls(); }}
        />
      )}

      {/* Деструктивное подтверждение удаления */}
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
    <Modal opened={opened} onClose={onClose} title="Создать короткую ссылку" size="md">
      <Stack gap="sm">
        <TextInput
          label="Целевой URL" required
          placeholder="https://example.com/long-path"
          value={longUrl} onChange={e => setLongUrl(e.currentTarget.value)}
        />
        <TextInput
          label={slugPrefix ? `Slug (префикс: ${slugPrefix})` : 'Custom Slug (необязательно)'}
          placeholder={slugPrefix ? `${slugPrefix}-...` : 'авто-генерация'}
          value={customSlug} onChange={e => setCustomSlug(e.currentTarget.value)}
        />
        <TextInput
          label="Заголовок (необязательно)"
          value={title} onChange={e => setTitle(e.currentTarget.value)}
        />
        <Group justify="flex-end" mt="md">
          <Button variant="default" onClick={onClose}>Отмена</Button>
          <Button onClick={handleSubmit} loading={loading} disabled={!longUrl.trim()}>
            Создать
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
  const [longUrl,  setLongUrl]  = useState(url.longUrl);
  const [title,    setTitle]    = useState(url.title ?? '');
  const [loading,  setLoading]  = useState(false);

  const handleSave = async () => {
    setLoading(true);
    try {
      await api.patch(`/api/shlink/short-urls/${url.shortCode}`, {
        longUrl: longUrl.trim(),
        title:   title.trim() || undefined,
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
          label="Целевой URL" required
          value={longUrl} onChange={e => setLongUrl(e.currentTarget.value)}
        />
        <TextInput
          label="Заголовок"
          value={title} onChange={e => setTitle(e.currentTarget.value)}
        />
        <Group justify="flex-end" mt="md">
          <Button variant="default" onClick={onClose}>Отмена</Button>
          <Button onClick={handleSave} loading={loading}>Сохранить</Button>
        </Group>
      </Stack>
    </Modal>
  );
}
