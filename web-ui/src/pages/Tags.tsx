import { useState } from 'react'
import {
  Title, Stack, Table, Group, Text, ActionIcon, Tooltip,
  TextInput, Skeleton, Modal, Button,
} from '@mantine/core'
import { IconSearch, IconPencil, IconTrash } from '@tabler/icons-react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { notifications } from '@mantine/notifications'
import { getTags, renameTag, deleteTag } from '@/api/endpoints/tags'
import { ConfirmDialog } from '@/components/ui/ConfirmDialog'
import { EmptyState } from '@/components/ui/EmptyState'
import { useAuth } from '@/contexts/AuthContext'
import type { TagEntry } from '@/types/api'

export function Tags() {
  const { can } = useAuth()
  const qc = useQueryClient()
  const [search, setSearch] = useState('')
  const [renameTarget, setRenameTarget] = useState<TagEntry | null>(null)
  const [newName, setNewName] = useState('')
  const [deleteTarget, setDeleteTarget] = useState<TagEntry | null>(null)

  const { data, isLoading } = useQuery({ queryKey: ['tags'], queryFn: getTags })

  const renameMutation = useMutation({
    mutationFn: ({ tagId, name }: { tagId: string; name: string }) => renameTag(tagId, name),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['tags'] })
      setRenameTarget(null)
      notifications.show({ color: 'teal', message: 'Тег переименован' })
    },
  })

  const deleteMutation = useMutation({
    mutationFn: (tagName: string) => deleteTag(tagName),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['tags'] })
      setDeleteTarget(null)
      notifications.show({ color: 'teal', message: 'Тег удалён' })
    },
  })

  const filtered = (data ?? []).filter((t) =>
    t.tag.toLowerCase().includes(search.toLowerCase())
  )

  return (
    <Stack gap="md">
      <Title order={2}>Теги</Title>

      <TextInput
        placeholder="Поиск по тегам..."
        leftSection={<IconSearch size={16} />}
        value={search}
        onChange={(e) => setSearch(e.currentTarget.value)}
        maw={400}
      />

      {isLoading ? (
        <Skeleton h={300} radius="md" />
      ) : filtered.length === 0 ? (
        <EmptyState title="Тегов не найдено" />
      ) : (
        <Table striped highlightOnHover withTableBorder>
          <Table.Thead>
            <Table.Tr>
              <Table.Th>Тег</Table.Th>
              <Table.Th>Ссылок</Table.Th>
              <Table.Th>Переходов</Table.Th>
              <Table.Th>Действия</Table.Th>
            </Table.Tr>
          </Table.Thead>
          <Table.Tbody>
            {filtered.map((tag) => (
              <Table.Tr key={tag.tag}>
                <Table.Td><Text size="sm" fw={500}>{tag.tag}</Text></Table.Td>
                <Table.Td>{tag.shortUrlsCount}</Table.Td>
                <Table.Td>{tag.visitsSummary.total}</Table.Td>
                <Table.Td>
                  <Group gap={4}>
                    {can('canManageAllTags') && (
                      <>
                        <Tooltip label="Переименовать">
                          <ActionIcon
                            variant="subtle" size="sm"
                            onClick={() => { setRenameTarget(tag); setNewName(tag.tag) }}
                          >
                            <IconPencil size={14} />
                          </ActionIcon>
                        </Tooltip>
                        <Tooltip label="Удалить">
                          <ActionIcon variant="subtle" size="sm" color="red" onClick={() => setDeleteTarget(tag)}>
                            <IconTrash size={14} />
                          </ActionIcon>
                        </Tooltip>
                      </>
                    )}
                  </Group>
                </Table.Td>
              </Table.Tr>
            ))}
          </Table.Tbody>
        </Table>
      )}

      <Modal
        opened={renameTarget !== null}
        onClose={() => setRenameTarget(null)}
        title="Переименовать тег"
      >
        <Stack>
          <Text size="sm" c="dimmed">Текущее имя: <b>{renameTarget?.tag}</b></Text>
          <TextInput
            label="Новое имя"
            value={newName}
            onChange={(e) => setNewName(e.currentTarget.value)}
          />
          <Group justify="flex-end">
            <Button variant="default" onClick={() => setRenameTarget(null)}>Отмена</Button>
            <Button
              loading={renameMutation.isPending}
              onClick={() => renameTarget && renameMutation.mutate({ tagId: renameTarget.tag, name: newName })}
            >
              Сохранить
            </Button>
          </Group>
        </Stack>
      </Modal>

      <ConfirmDialog
        opened={deleteTarget !== null}
        onClose={() => setDeleteTarget(null)}
        onConfirm={() => deleteTarget && deleteMutation.mutate(deleteTarget.tag)}
        title="Удалить тег?"
        message={`Тег «${deleteTarget?.tag}» будет удалён.`}
        confirmLabel="Удалить"
        loading={deleteMutation.isPending}
        danger
      />
    </Stack>
  )
}
