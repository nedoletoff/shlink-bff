import type { ReactNode } from 'react'
import { useAuth } from '@/contexts/AuthContext'

interface Props {
  permission: string
  fallback?: ReactNode
  children: ReactNode
}

export function PermissionGuard({ permission, fallback = null, children }: Props) {
  const { can } = useAuth()
  return can(permission) ? <>{children}</> : <>{fallback}</>
}
