import { Modal, Text, Group, Button, Stack } from '@mantine/core'
import { IconAlertTriangle } from '@tabler/icons-react'

interface Props {
  opened: boolean
  onClose: () => void
  onConfirm: () => void
  title: string
  message: string
  confirmLabel?: string
  loading?: boolean
  danger?: boolean
}

export function ConfirmDialog({
  opened, onClose, onConfirm, title, message,
  confirmLabel = 'Подтвердить',
  loading = false,
  danger = false,
}: Props) {
  return (
    <Modal
      opened={opened}
      onClose={onClose}
      title={title}
      styles={danger ? { header: { borderBottom: '2px solid var(--mantine-color-red-6)' } } : {}}
    >
      <Stack>
        {danger && (
          <Group gap="xs" c="red">
            <IconAlertTriangle size={16} />
            <Text size="sm">**Это действие нельзя отменить**</Text>
          </Group>
        )}
        <Text size="sm">{message}</Text>
        <Group justify="flex-end">
          <Button variant="default" onClick={onClose}>Отмена</Button>
          <Button
            color={danger ? 'red' : 'blue'}
            onClick={onConfirm}
            loading={loading}
          >
            {confirmLabel}
          </Button>
        </Group>
      </Stack>
    </Modal>
  )
}
