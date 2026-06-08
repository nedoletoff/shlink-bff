import { useState } from 'react'
import { AppShell, Burger, Group, Text, useMantineColorScheme } from '@mantine/core'
import { useDisclosure } from '@mantine/hooks'
import { Sidebar } from './Sidebar'
import { Header } from './Header'

export function AppShellWrapper({ children }: { children: React.ReactNode }) {
  const [opened, { toggle }] = useDisclosure()

  return (
    <AppShell
      header={{ height: 60 }}
      navbar={{
        width: 260,
        breakpoint: 'sm',
        collapsed: { mobile: !opened },
      }}
      padding="md"
    >
      <AppShell.Header>
        <Group h="100%" px="md" justify="space-between">
          <Group>
            <Burger opened={opened} onClick={toggle} hiddenFrom="sm" size="sm" />
            <Text fw={700} size="lg" c="blue">Shlink BFF</Text>
          </Group>
          <Header />
        </Group>
      </AppShell.Header>

      <AppShell.Navbar p="md">
        <Sidebar onNavigate={() => { if (opened) toggle() }} />
      </AppShell.Navbar>

      <AppShell.Main>
        {children}
      </AppShell.Main>
    </AppShell>
  )
}
