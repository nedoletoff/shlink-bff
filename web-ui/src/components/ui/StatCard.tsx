import { Card, Text, Group, ThemeIcon } from '@mantine/core'
import type { ReactNode } from 'react'

interface Props {
  title: string
  value: string | number
  icon: ReactNode
  color?: string
}

export function StatCard({ title, value, icon, color = 'blue' }: Props) {
  return (
    <Card withBorder radius="md" p="md">
      <Group justify="space-between" align="flex-start">
        <div>
          <Text size="xs" c="dimmed" tt="uppercase" fw={700}>{title}</Text>
          <Text size="xl" fw={700} mt={4}>{value}</Text>
        </div>
        <ThemeIcon size="lg" variant="light" color={color} radius="md">
          {icon}
        </ThemeIcon>
      </Group>
    </Card>
  )
}
