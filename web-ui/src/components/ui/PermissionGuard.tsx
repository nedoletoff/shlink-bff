import type { ReactNode } from 'react'
import { useAuth } from '@/contexts/AuthContext'
import type { MeResponse } from '@/types/api'

interface Props {
  permission: keyof MeResponse['permissions']
  fallback?: ReactNode
  children: ReactNode
}

export function PermissionGuard({ permission, fallback = null, children }: Props) {
  const { can } = useAuth()
  return can(permission) ? <>{children}</> : <>{fallback}</>
}
