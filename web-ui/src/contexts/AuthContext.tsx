import { createContext, useContext, type ReactNode } from 'react'
import { useQuery } from '@tanstack/react-query'
import { Center, Loader } from '@mantine/core'
import { getMe } from '@/api/endpoints/auth'
import type { MeResponse } from '@/types/api'

interface AuthContextValue {
  me: MeResponse | null
  isLoading: boolean
  can: (permission: keyof MeResponse['permissions']) => boolean
  isAdmin: () => boolean
}

const AuthContext = createContext<AuthContextValue | null>(null)

export function AuthProvider({ children }: { children: ReactNode }) {
  const { data: me, isLoading } = useQuery({
    queryKey: ['me'],
    queryFn: getMe,
    retry: false,
  })

  const can = (permission: keyof MeResponse['permissions']) =>
    me?.permissions?.[permission] ?? false

  const isAdmin = () =>
    me?.role === (me as unknown as Record<string, string>)?.adminRole ||
    me?.permissions?.canManageUsers === true

  if (isLoading) {
    return (
      <Center h="100vh">
        <Loader size="lg" />
      </Center>
    )
  }

  return (
    <AuthContext.Provider value={{ me: me ?? null, isLoading, can, isAdmin }}>
      {children}
    </AuthContext.Provider>
  )
}

export function useAuth() {
  const ctx = useContext(AuthContext)
  if (!ctx) throw new Error('useAuth must be used within AuthProvider')
  return ctx
}
