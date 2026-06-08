import { apiClient } from '../client'
import type { URLDetailResponse } from '@/types/api'

export const getLinkDetail = (shortCode: string, period: number) =>
  apiClient.get<URLDetailResponse>(`/api/urls/${shortCode}/detail`, { params: { period } }).then((r) => r.data)
