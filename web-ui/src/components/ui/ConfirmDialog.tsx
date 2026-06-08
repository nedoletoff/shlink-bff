import { Modal, Button, Group, Text } from '@mantine/core'

interface Props {
  opened: boolean
  onClose: () => void
  onConfirm: () => void
  title?: string
  message: string
  confirmLabel?: string
  confirmColor?: string
  loading?: boolean
}

export function ConfirmDialog({
  opened, onClose, onConfirm,
  title = 'Подтвердите действие',
  message, confirmLabel = 'Подтвердить',
  confirmColor = 'red', loading = false,
}: Props) {
  return (
    <Modal opened={opened} onClose={onClose} title={title} size="sm">
      <Text size="sm" mb="lg">{message}</Text>
      <Group justify="flex-end">
        <Button variant="default" onClick={onClose} disabled={loading}>Отмена</Button>
        <Button color={confirmColor} onClick={onConfirm} loading={loading}>{confirmLabel}</Button>
      </Group>
    </Modal>
  )
}
