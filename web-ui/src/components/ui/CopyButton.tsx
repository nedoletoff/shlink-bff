import { ActionIcon, Tooltip } from '@mantine/core'
import { useClipboard } from '@mantine/hooks'
import { notifications } from '@mantine/notifications'
import { IconCopy, IconCheck } from '@tabler/icons-react'

interface Props {
  value: string
}

export function CopyButton({ value }: Props) {
  const clipboard = useClipboard({ timeout: 2000 })

  const handleCopy = () => {
    clipboard.copy(value)
    notifications.show({ message: 'Скопировано', color: 'teal', autoClose: 2000 })
  }

  return (
    <Tooltip label={clipboard.copied ? 'Скопировано' : 'Копировать'}>
      <ActionIcon variant="subtle" size="sm" onClick={handleCopy}>
        {clipboard.copied ? <IconCheck size={14} color="teal" /> : <IconCopy size={14} />}
      </ActionIcon>
    </Tooltip>
  )
}
