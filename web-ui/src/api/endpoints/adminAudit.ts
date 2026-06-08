import api from '@/api/client'
import type { AuditLogsResponse } from '@/types/api'

export interface AuditParams {
  page?: number
  perPage?: number
  username?: string
  action?: string
  result?: string
  dateFrom?: string
  dateTo?: string
}

export async function getAuditLogs(params: AuditParams = {}): Promise<AuditLogsResponse> {
  const { data } = await api.get<AuditLogsResponse>('/api/admin/logs', { params })
  return data
}
