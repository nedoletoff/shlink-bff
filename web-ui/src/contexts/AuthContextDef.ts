import { createContext } from 'react'
import type { MeResponse } from '@/types/api'

export interface AuthContextValue {
  me: MeResponse | null
  isLoading: boolean
  can: (permission: keyof MeResponse['permissions']) => boolean
  isAdmin: () => boolean
}

export const AuthContext = createContext<AuthContextValue | null>(null)
