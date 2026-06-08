import api from '@/api/client'
import type { DashboardResponse } from '@/types/api'

export async function getDashboard(period = 30): Promise<DashboardResponse> {
  const { data } = await api.get<DashboardResponse>('/api/dashboard', { params: { period } })
  return data
}
