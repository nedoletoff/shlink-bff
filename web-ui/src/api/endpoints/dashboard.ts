import { apiClient } from '../client'
import type { DashboardResponse } from '@/types/api'

export const getDashboard = (period: number) =>
  apiClient.get<DashboardResponse>('/api/dashboard', { params: { period } }).then((r) => r.data)
