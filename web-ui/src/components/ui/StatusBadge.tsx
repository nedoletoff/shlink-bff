import { Badge } from '@mantine/core'

const STATUS_MAP: Record<string, { color: string; label: string }> = {
  active:   { color: 'teal',   label: 'Активна' },
  inactive: { color: 'gray',   label: 'Неактивна' },
  pending:  { color: 'yellow', label: 'Ожидает' },
  disabled: { color: 'red',    label: 'Отключена' },
}

export function StatusBadge({ status }: { status: string }) {
  const { color, label } = STATUS_MAP[status.toLowerCase()] ?? { color: 'gray', label: status }
  return <Badge color={color} variant="light" size="sm">{label}</Badge>
}
