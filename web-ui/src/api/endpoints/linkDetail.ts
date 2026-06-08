import api from '@/api/client'
import type { URLDetailResponse } from '@/types/api'

export async function getLinkDetail(shortCode: string, period = 30): Promise<URLDetailResponse> {
  const { data } = await api.get<URLDetailResponse>(`/api/urls/${shortCode}/detail`, {
    params: { period },
  })
  return data
}
