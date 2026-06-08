import type { ReactNode } from 'react'
import { useQuery } from '@tanstack/react-query'
import { Center, Loader } from '@mantine/core'
import { getMe } from '@/api/endpoints/auth'
import type { MeResponse } from '@/types/api'
import { AuthContext } from './AuthContextDef'

export function AuthProvider({ children }: { children: ReactNode }) {
  const { data: me, isLoading } = useQuery({
    queryKey: ['me'],
    queryFn: getMe,
    retry: false,
  })

  const can = (permission: keyof MeResponse['permissions']) =>
    me?.permissions?.[permission] ?? false

  // Баг #1: прежняя логика сравнивала role c adminRole которого нет в MeResponse -
  // используем canManageUsers как единственный надёжный источник
  const isAdmin = () => me?.permissions?.canManageUsers === true

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
