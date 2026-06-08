import { Group, Text, Button, Anchor } from '@mantine/core'
import { IconLogout } from '@tabler/icons-react'
import { ThemeToggle } from './ThemeToggle'
import { RoleBadge } from '@/components/ui/RoleBadge'
import { useAuth } from '@/contexts/AuthContext'

export function Header() {
  const { me } = useAuth()

  return (
    <Group gap="sm">
      <ThemeToggle />
      {me && (
        <Group gap="xs">
          <Text size="sm" fw={500}>{me.username}</Text>
          <RoleBadge role={me.role} />
        </Group>
      )}
      <Anchor href="/auth/logout" underline="never">
        <Button variant="subtle" size="xs" leftSection={<IconLogout size={14} />}>
          Выйти
        </Button>
      </Anchor>
    </Group>
  )
}
