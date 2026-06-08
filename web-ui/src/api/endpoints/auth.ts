import { apiClient } from '../client'
import type { MeResponse } from '@/types/api'

export const getMe = () =>
  apiClient.get<MeResponse>('/api/me').then((r) => r.data)
