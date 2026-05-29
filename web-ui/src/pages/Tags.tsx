import { useEffect, useState } from 'react';
import {
  Stack, Title, Table, Text, Badge, Group,
  Button, Loader, Center, Modal, TextInput,
} from '@mantine/core';
import { useDisclosure } from '@mantine/hooks';
import { notifications } from '@mantine/notifications';
import { IconTrash, IconEdit } from '@tabler/icons-react';
import { api } from '../api/client';
import { useAuth } from '../contexts/AuthContext';
import { ConfirmDialog } from '../components/ui/ConfirmDialog';
import type { TagStats, TagsResponse } from '../types/api';

export function Tags() {
  const { user }  = useAuth();
  const [tags,         setTags]         = useState<TagStats[]>([]);
  const [loading,      setLoading]      = useState(true);
  const [deleteTarget, setDeleteTarget] = useState<string | null>(null);
  const [editTarget,   setEditTarget]   = useState<string | null>(null);
  const [renameOpen, { open: openRename, close: closeRename }] = useDisclosure(false);

  const fetchTags = () => {
    setLoading(true);
    api.get<TagsResponse>('/api/shlink/tags')
      .then(r => setTags(r.tags.data))
      .catch(() => setTags([]))
      .finally(() => setLoading(false));
  };

  useEffect(() => { fetchTags(); }, []);

  const handleDelete = async () => {
    if (!deleteTarget) return;
    try {
      await api.delete(`/api/shlink/tags/${encodeURIComponent(deleteTarget)}`);
      notifications.show({ message: `Тег "${deleteTarget}" удалён`, color: 'green' });
      setDeleteTarget(null);
      fetchTags();
    } catch {
      // handled
    }
  };

  const handleEditOpen = (tag: string) => {
    setEditTarget(tag);
    openRename();
  };

  return (
    <Stack gap="lg">
      <Title order={2}>Теги</Title>

      {loading ? (
        <Center h={200}><Loader /></Center>
      ) : (
        <Table striped highlightOnHover withTableBorder>
          <Table.Thead>
            <Table.Tr>
              <Table.Th>Тег</Table.Th>
              <Table.Th>Ссылок</Table.Th>
              <Table.Th>Переходов</Table.Th>
              <Table.Th />
            </Table.Tr>
          </Table.Thead>
          <Table.Tbody>
            {tags.map(t => (
              <Table.Tr key={t.tag}>
                <Table.Td>
                  <Badge variant="light" size="md">{t.tag}</Badge>
                </Table.Td>
                <Table.Td>{t.shortUrlsCount}</Table.Td>
                              <Table.Td>{t.visitsSummary.total.toLocaleString('ru')}</Table.Td>
                <Table.Td>
                  <Group gap={4}>
                    {user?.permissions.canManageOwnTags && (
                      <>
                        <Button
                          size="xs" variant="subtle"
                          leftSection={<IconEdit size={14} />}
                          onClick={() => handleEditOpen(t.tag)}
                        >
                          Переименовать
                        </Button>
                        <Button
                          size="xs" variant="subtle" color="red"
                          leftSection={<IconTrash size={14} />}
                          onClick={() => setDeleteTarget(t.tag)}
                        >
                          Удалить
                        </Button>
                      </>
                    )}
                  </Group>
                </Table.Td>
              </Table.Tr>
            ))}
            {tags.length === 0 && (
              <Table.Tr>
                <Table.Td colSpan={4}>
                  <Center p="xl"><Text c="dimmed">Теги не найдены</Text></Center>
                </Table.Td>
              </Table.Tr>
            )}
          </Table.Tbody>
        </Table>
      )}

      <ConfirmDialog
        opened={!!deleteTarget}
        title="Удалить тег?"
        message={`Тег "${deleteTarget}" будет удалён. Ссылки с этим тегом останутся, но тег будет снят.`}
        confirmLabel="Удалить"
        onConfirm={handleDelete}
        onCancel={() => setDeleteTarget(null)}
      />

      <RenameTagModal
        opened={renameOpen}
        oldTag={editTarget ?? ''}
        onClose={() => { closeRename(); setEditTarget(null); }}
        onRenamed={() => { closeRename(); setEditTarget(null); fetchTags(); }}
      />
    </Stack>
  );
}

function RenameTagModal({
  opened, oldTag, onClose, onRenamed,
}: {
  opened: boolean; oldTag: string; onClose: () => void; onRenamed: () => void;
}) {
  const [newTag,  setNewTag]  = useState('');
  const [loading, setLoading] = useState(false);

  const handleRename = async () => {
    if (!newTag.trim()) return;
    setLoading(true);
    try {
      await api.put(`/api/shlink/tags/${encodeURIComponent(oldTag)}`, {
        oldName: oldTag,
        newName: newTag.trim(),
      });
      notifications.show({ message: `Тег переименован в "${newTag}"`, color: 'green' });
      onRenamed();
    } catch {
      // handled
    } finally {
      setLoading(false);
    }
  };

  return (
    <Modal opened={opened} onClose={onClose} title={`Переименовать тег: ${oldTag}`} size="sm">
      <Stack gap="sm">
        <TextInput
          label="Новое имя"
          placeholder="new-tag-name"
          value={newTag}
          onChange={e => setNewTag(e.currentTarget.value)}
          onKeyDown={e => e.key === 'Enter' && handleRename()}
        />
        <Group justify="flex-end" mt="md">
          <Button variant="default" onClick={onClose}>Отмена</Button>
          <Button onClick={handleRename} loading={loading} disabled={!newTag.trim()}>
            Переименовать
          </Button>
        </Group>
      </Stack>
    </Modal>
  );
}
