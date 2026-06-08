import { createContext, useContext, type ReactNode } from 'react'
import { useQuery } from '@tanstack/react-query'
import { getMe } from '@/api/endpoints/auth'
import type { MeResponse } from '@/types/api'

interface AuthContextValue {
  me: MeResponse | null
  isLoading: boolean
  can: (permission: string) => boolean
  isAdmin: () => boolean
}

const AuthContext = createContext<AuthContextValue | null>(null)

export function AuthProvider({ children }: { children: ReactNode }) {
  const { data: me, isLoading } = useQuery({
    queryKey: ['me'],
    queryFn: getMe,
    staleTime: 60_000,
    retry: false,
  })

  const can = (permission: string): boolean => {
    if (!me) return false
    return me.permissions[permission] === true
  }

  const isAdmin = (): boolean => {
    if (!me) return false
    return me.role === 'admin'
  }

  return (
    <AuthContext.Provider value={{ me: me ?? null, isLoading, can, isAdmin }}>
      {children}
    </AuthContext.Provider>
  )
}

export function useAuth(): AuthContextValue {
  const ctx = useContext(AuthContext)
  if (!ctx) throw new Error('useAuth must be used inside AuthProvider')
  return ctx
}

export function usePermission(permission: string): boolean {
  const { can } = useAuth()
  return can(permission)
}
