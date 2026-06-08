import { apiClient } from '../client'
import type { UserRecord } from '@/types/api'

export const getAdminUsers = () =>
  apiClient.get<UserRecord[]>('/api/admin/users').then((r) => r.data)

export const getAdminUser = (sub: string) =>
  apiClient.get<UserRecord>(`/api/admin/users/${sub}`).then((r) => r.data)

export const updateAdminUser = (sub: string, payload: Partial<Pick<UserRecord, 'role' | 'status' | 'slugPrefix'>>) =>
  apiClient.put<UserRecord>(`/api/admin/users/${sub}`, payload).then((r) => r.data)

export const updateApiKey = (sub: string, apiKey: string) =>
  apiClient.put(`/api/admin/users/${sub}/apikey`, { apiKey }).then((r) => r.data)
