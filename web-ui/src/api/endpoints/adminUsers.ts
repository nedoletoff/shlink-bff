import api from '@/api/client'
import type { UserRecord } from '@/types/api'

export async function listUsers(): Promise<UserRecord[]> {
  const { data } = await api.get<UserRecord[]>('/api/admin/users')
  return data
}

export async function getUser(sub: string): Promise<UserRecord> {
  const { data } = await api.get<UserRecord>(`/api/admin/users/${sub}`)
  return data
}

export async function updateUser(sub: string, payload: Partial<Pick<UserRecord, 'role' | 'status' | 'slugPrefix'>>) {
  const { data } = await api.put<UserRecord>(`/api/admin/users/${sub}`, payload)
  return data
}

export async function updateApiKey(sub: string, apiKey: string) {
  await api.put(`/api/admin/users/${sub}/apikey`, { apiKey })
}
