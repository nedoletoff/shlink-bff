import api from '@/api/client'
import type { MeResponse } from '@/types/api'

export async function getMe(): Promise<MeResponse> {
  const { data } = await api.get<MeResponse>('/api/me')
  return data
}
