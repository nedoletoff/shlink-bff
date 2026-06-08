import { Badge } from '@mantine/core'

interface Props {
  active: boolean | undefined
  activeLabel?: string
  inactiveLabel?: string
}

export function StatusBadge({ active, activeLabel = 'Активна', inactiveLabel = 'Неактивна' }: Props) {
  return (
    <Badge color={active ? 'green' : 'red'} variant="light" size="sm">
      {active ? activeLabel : inactiveLabel}
    </Badge>
  )
}
