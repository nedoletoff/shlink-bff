import { Stack, Text, ThemeIcon } from '@mantine/core'
import { IconLink } from '@tabler/icons-react'
import type { ReactNode } from 'react'

interface Props {
  icon?: ReactNode
  title?: string
  description?: string
  action?: ReactNode
}

export function EmptyState({
  icon = <IconLink size={32} />,
  title = 'Ничего не найдено',
  description,
  action,
}: Props) {
  return (
    <Stack align="center" py="xl" gap="sm">
      <ThemeIcon size="xl" variant="light" color="gray">
        {icon}
      </ThemeIcon>
      <Text fw={500} c="dimmed">{title}</Text>
      {description && <Text size="sm" c="dimmed">{description}</Text>}
      {action}
    </Stack>
  )
}
