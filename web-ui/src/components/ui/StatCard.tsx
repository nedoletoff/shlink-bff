import { Paper, Text, Group, type PaperProps } from '@mantine/core'
import type { ReactNode } from 'react'

interface Props extends PaperProps {
  label: string
  value: ReactNode
  icon?: ReactNode
  delta?: number
}

export function StatCard({ label, value, icon, delta, ...rest }: Props) {
  return (
    <Paper withBorder p="md" radius="md" {...rest}>
      <Group justify="space-between" mb={4}>
        <Text size="sm" c="dimmed" fw={500}>{label}</Text>
        {icon}
      </Group>
      <Text size="xl" fw={700} style={{ fontVariantNumeric: 'tabular-nums' }}>{value}</Text>
      {delta !== undefined && (
        <Text size="xs" c={delta >= 0 ? 'teal' : 'red'} mt={4}>
          {delta >= 0 ? '+' : ''}{delta}% за период
        </Text>
      )}
    </Paper>
  )
}
