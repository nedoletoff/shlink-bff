import { ActionIcon, Tooltip } from '@mantine/core'
import { useClipboard } from '@mantine/hooks'
import { IconCopy, IconCheck } from '@tabler/icons-react'

export function CopyButton({ value, size = 16 }: { value: string; size?: number }) {
  const clipboard = useClipboard({ timeout: 2000 })

  return (
    <Tooltip label={clipboard.copied ? 'Скопировано!' : 'Скопировать'} withArrow>
      <ActionIcon
        variant="subtle"
        size="sm"
        onClick={() => clipboard.copy(value)}
        aria-label="Скопировать"
      >
        {clipboard.copied ? <IconCheck size={size} color="teal" /> : <IconCopy size={size} />}
      </ActionIcon>
    </Tooltip>
  )
}
