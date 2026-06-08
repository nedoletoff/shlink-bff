import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import {
  Stack, Group, Title, Table, Text, Badge,
  ActionIcon, Tooltip, TextInput, Modal, Button, Skeleton,
} from '@mantine/core'
import { useForm } from '@mantine/form'
import { notifications } from '@mantine/notifications'
import { IconPlus, IconPencil, IconTrash, IconTags } from '@tabler/icons-react'
import { getTags, createTag, renameTag, deleteTag } from '@/api/endpoints/tags'
import { ConfirmDialog } from '@/components/ui/ConfirmDialog'
import { EmptyState } from '@/components/ui/EmptyState'

const TAG_COLORS = ['teal', 'blue', 'grape', 'orange', 'cyan', 'pink', 'lime', 'indigo']
function tagColor(tag: string) {
  let h = 0
  for (let i = 0; i < tag.length; i++) h = (h * 31 + tag.charCodeAt(i)) & 0xffffffff
  return TAG_COLORS[Math.abs(h) % TAG_COLORS.length]
}

export function Tags() {
  const qc = useQueryClient()
  const [renaming, setRenaming] = useState<string | null>(null)
  const [removing, setRemoving] = useState<string | null>(null)
  const [createOpened, setCreateOpened] = useState(false)

  const { data, isLoading } = useQuery({
    queryKey: ['tags'],
    queryFn: getTags,
  })

  const tags = data?.tags.data ?? []

  const createMutation = useMutation({
    mutationFn: (name: string) => createTag(name),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['tags'] })
      notifications.show({ color: 'teal', message: 'Тег создан' })
      setCreateOpened(false)
    },
  })

  const renameMutation = useMutation({
    mutationFn: ({ old, next }: { old: string; next: string }) => renameTag(old, next),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['tags'] })
      notifications.show({ color: 'teal', message: 'Тег переименован' })
      setRenaming(null)
    },
  })

  const deleteMutation = useMutation({
    mutationFn: () => deleteTag(removing!),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['tags'] })
      notifications.show({ color: 'teal', message: 'Тег удалён' })
      setRemoving(null)
    },
  })

  const createForm = useForm<{ name: string }>({ initialValues: { name: '' } })
  const renameForm = useForm<{ name: string }>({ initialValues: { name: renaming ?? '' } })

  const openRename = (tag: string) => {
    setRenaming(tag)
    renameForm.setValues({ name: tag })
  }

  return (
    <Stack gap="md">
      <Group justify="space-between">
        <Title order={2}>Теги</Title>
        <Button leftSection={<IconPlus size={16} />} onClick={() => { createForm.reset(); setCreateOpened(true) }}>
          Создать тег
        </Button>
      </Group>

      {isLoading ? (
        <Stack gap="xs">
          {Array.from({ length: 6 }).map((_, i) => <Skeleton key={i} height={44} radius="sm" />)}
        </Stack>
      ) : tags.length === 0 ? (
        <EmptyState
          icon={<IconTags size={24} />}
          title="Тегов нет"
          description="Создайте первый тег или добавьте теги к ссылкам"
          action={<Button onClick={() => setCreateOpened(true)}>Создать тег</Button>}
        />
      ) : (
        <Table.ScrollContainer minWidth={400}>
          <Table striped highlightOnHover>
            <Table.Thead>
              <Table.Tr>
                <Table.Th>Тег</Table.Th>
                <Table.Th>Ссылок</Table.Th>
                <Table.Th>Переходов</Table.Th>
                <Table.Th />
              </Table.Tr>
            </Table.Thead>
            <Table.Tbody>
              {tags.map((t) => (
                <Table.Tr key={t.tag}>
                  <Table.Td>
                    <Badge variant="light" color={tagColor(t.tag)}>{t.tag}</Badge>
                  </Table.Td>
                  <Table.Td>
                    <Text size="sm" style={{ fontVariantNumeric: 'tabular-nums' }}>{t.shortUrlsCount}</Text>
                  </Table.Td>
                  <Table.Td>
                    <Text size="sm" style={{ fontVariantNumeric: 'tabular-nums' }}>
                      {t.visitsSummary.total.toLocaleString('ru')}
                    </Text>
                  </Table.Td>
                  <Table.Td>
                    <Group gap={4} justify="flex-end">
                      <Tooltip label="Переименовать">
                        <ActionIcon variant="subtle" size="sm" onClick={() => openRename(t.tag)} aria-label="Переименовать">
                          <IconPencil size={14} />
                        </ActionIcon>
                      </Tooltip>
                      <Tooltip label="Удалить">
                        <ActionIcon variant="subtle" size="sm" color="red" onClick={() => setRemoving(t.tag)} aria-label="Удалить">
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

      {/* Create modal */}
      <Modal opened={createOpened} onClose={() => setCreateOpened(false)} title="Создать тег" size="sm">
        <form onSubmit={createForm.onSubmit((v) => createMutation.mutate(v.name))}>
          <Stack gap="sm">
            <TextInput label="Название" placeholder="например: marketing" required {...createForm.getInputProps('name')} />
            <Group justify="flex-end">
              <Button variant="default" onClick={() => setCreateOpened(false)}>Отмена</Button>
              <Button type="submit" loading={createMutation.isPending}>Создать</Button>
            </Group>
          </Stack>
        </form>
      </Modal>

      {/* Rename modal */}
      <Modal opened={Boolean(renaming)} onClose={() => setRenaming(null)} title="Переименовать тег" size="sm">
        <form onSubmit={renameForm.onSubmit((v) => renameMutation.mutate({ old: renaming!, next: v.name }))}>
          <Stack gap="sm">
            <TextInput label="Новое название" required {...renameForm.getInputProps('name')} />
            <Group justify="flex-end">
              <Button variant="default" onClick={() => setRenaming(null)}>Отмена</Button>
              <Button type="submit" loading={renameMutation.isPending}>Сохранить</Button>
            </Group>
          </Stack>
        </form>
      </Modal>

      <ConfirmDialog
        opened={Boolean(removing)}
        onClose={() => setRemoving(null)}
        onConfirm={() => deleteMutation.mutate()}
        message={`Удалить тег «${removing}»? Ссылки сохранятся.`}
        loading={deleteMutation.isPending}
      />
    </Stack>
  )
}
