import { Badge } from '@mantine/core'

const ROLE_MAP: Record<string, string> = {
  admin:     'red',
  moderator: 'orange',
  user:      'blue',
  viewer:    'gray',
}

export function RoleBadge({ role }: { role: string }) {
  const color = ROLE_MAP[role.toLowerCase()] ?? 'gray'
  return <Badge color={color} variant="outline" size="sm">{role}</Badge>
}
