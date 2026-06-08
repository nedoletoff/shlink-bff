import { Badge } from '@mantine/core'

const COLOR_MAP: Record<string, string> = {
  admin: 'red',
  user: 'blue',
}

export function RoleBadge({ role }: { role: string }) {
  const color = COLOR_MAP[role] ?? 'gray'
  return (
    <Badge color={color} variant="light" size="sm">
      {role}
    </Badge>
  )
}
