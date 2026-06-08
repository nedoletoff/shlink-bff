import { AppShell, Box } from '@mantine/core'
import { useDisclosure } from '@mantine/hooks'
import { type ReactNode } from 'react'
import { Sidebar } from './Sidebar'
import { Header } from './Header'

export function AppShellWrapper({ children }: { children: ReactNode }) {
  const [opened, { toggle }] = useDisclosure()

  return (
    <AppShell
      header={{ height: 56 }}
      navbar={{ width: 220, breakpoint: 'sm', collapsed: { mobile: !opened } }}
      padding="md"
    >
      <AppShell.Header>
        <Header opened={opened} onToggle={toggle} />
      </AppShell.Header>
      <AppShell.Navbar>
        <Sidebar onClose={toggle} />
      </AppShell.Navbar>
      <AppShell.Main>
        <Box maw={1400} mx="auto">
          {children}
        </Box>
      </AppShell.Main>
    </AppShell>
  )
}
