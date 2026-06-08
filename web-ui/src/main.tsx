import React from 'react'
import ReactDOM from 'react-dom/client'
import { BrowserRouter } from 'react-router-dom'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MantineProvider, createTheme } from '@mantine/core'
import { Notifications } from '@mantine/notifications'
import { DatesProvider } from '@mantine/dates'
import { AuthProvider } from '@/contexts/AuthContext'
import App from './App'
import '@mantine/core/styles.css'
import '@mantine/notifications/styles.css'
import '@mantine/dates/styles.css'

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 30_000,
      retry: 1,
    },
  },
})

const theme = createTheme({
  primaryColor: 'teal',
  defaultRadius: 'md',
  fontFamily: 'Inter, system-ui, sans-serif',
  components: {
    Button: { defaultProps: { fw: 500 } },
    TextInput: { defaultProps: { size: 'sm' } },
    Select: { defaultProps: { size: 'sm' } },
    Table: { defaultProps: { striped: true, highlightOnHover: true } },
  },
})

ReactDOM.createRoot(document.getElementById('root')!).render(
  <React.StrictMode>
    <QueryClientProvider client={queryClient}>
      <MantineProvider theme={theme} defaultColorScheme="auto">
        <Notifications position="top-right" />
        <DatesProvider settings={{ locale: 'ru' }}>
          <BrowserRouter>
            <AuthProvider>
              <App />
            </AuthProvider>
          </BrowserRouter>
        </DatesProvider>
      </MantineProvider>
    </QueryClientProvider>
  </React.StrictMode>,
)
