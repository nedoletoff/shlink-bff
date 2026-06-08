import { apiClient } from '../client'
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

export const getAuditLogs = (params: AuditParams) =>
  apiClient.get<AuditLogsResponse>('/api/admin/logs', { params }).then((r) => r.data)

export const deleteAuditLogs = (ids: number[]) =>
  apiClient.delete('/api/admin/logs', { data: { ids } }).then((r) => r.data)
