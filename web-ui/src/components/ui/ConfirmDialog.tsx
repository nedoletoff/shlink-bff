import { Modal, Text, Group, Button, Stack, Alert } from '@mantine/core';
import { IconAlertTriangle } from '@tabler/icons-react';

interface Props {
  opened:        boolean;
  title:         string;
  message:       string;
  confirmLabel?: string;
  confirmColor?: string;
  onConfirm:     () => void | Promise<void>;
  onCancel:      () => void;
}

export function ConfirmDialog({
  opened, title, message,
  confirmLabel = 'Подтвердить',
  confirmColor = 'red',
  onConfirm, onCancel,
}: Props) {
  return (
    <Modal opened={opened} onClose={onCancel} title={title} size="sm" centered>
      <Stack gap="md">
        <Alert icon={<IconAlertTriangle size={18} />} color="red" variant="light">
          {message}
        </Alert>
        <Group justify="flex-end">
          <Button variant="default" onClick={onCancel}>Отмена</Button>
          <Button color={confirmColor} onClick={onConfirm}>{confirmLabel}</Button>
        </Group>
      </Stack>
    </Modal>
  );
}
