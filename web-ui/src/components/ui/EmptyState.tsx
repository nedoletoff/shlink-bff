import { Stack, Text, ThemeIcon, type ThemeIconProps } from '@mantine/core'
import type { ReactNode } from 'react'

interface Props {
  icon: ReactNode
  title: string
  description?: string
  action?: ReactNode
  color?: ThemeIconProps['color']
}

export function EmptyState({ icon, title, description, action, color = 'gray' }: Props) {
  return (
    <Stack align="center" gap="xs" py={48}>
      <ThemeIcon size={48} radius="xl" variant="light" color={color}>
        {icon}
      </ThemeIcon>
      <Text fw={500} size="lg">{title}</Text>
      {description && <Text size="sm" c="dimmed" ta="center" maw={360}>{description}</Text>}
      {action}
    </Stack>
  )
}
