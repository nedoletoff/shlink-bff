import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import {
  Stack, Group, Title, Table, Text,
  ActionIcon, Tooltip, TextInput, Modal, Button, Skeleton,
} from '@mantine/core'
import { useForm } from '@mantine/form'
import { notifications } from '@mantine/notifications'
import { IconPencil, IconTrash, IconTags } from '@tabler/icons-react'
import { getTags, renameTag, deleteTag } from '@/api/endpoints/tags'
import { ConfirmDialog } from '@/components/ui/ConfirmDialog'
import { EmptyState } from '@/components/ui/EmptyState'

export function Tags() {
  const qc = useQueryClient()
  const [renaming, setRenaming] = useState<string | null>(null)
  const [removing, setRemoving] = useState<string | null>(null)

  const { data, isLoading } = useQuery({
    queryKey: ['tags'],
    queryFn: getTags,
  })

  const tags = data?.tags.data ?? []

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
      setRenaming(null)
    },
  })

  const form = useForm({ initialValues: { name: renaming ?? '' } })

  const openRename = (tag: string) => {
    setRenaming(tag)
    form.setValues({ name: tag })
  }

  return (
    <Stack gap="md">
      <Title order={2}>Теги</Title>

      {isLoading ? (
        <Skeleton height={300} />
      ) : tags.length === 0 ? (
        <EmptyState icon={<IconTags size={24} />} title="Тегов нет" description="Добавьте теги к ссылкам" />
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
                  <Table.Td><Text size="sm" fw={500}>{t.tag}</Text></Table.Td>
                  <Table.Td><Text size="sm">{t.shortUrlsCount}</Text></Table.Td>
                  <Table.Td><Text size="sm">{t.visitsSummary.total.toLocaleString('ru')}</Text></Table.Td>
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

      <Modal opened={Boolean(renaming)} onClose={() => setRenaming(null)} title="Переименовать тег" size="sm">
        <form onSubmit={form.onSubmit((v) => renameMutation.mutate({ old: renaming!, next: v.name }))}>
          <Stack gap="sm">
            <TextInput label="Новое название" required {...form.getInputProps('name')} />
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
        message={`Удалить тег «${removing}»? Это не удалит ссылки.`}
        loading={deleteMutation.isPending}
      />
    </Stack>
  )
}
