import api from '@/api/client'
import type { ServerSettings, PatchSettingsPayload } from '@/types/api'

export async function getAdminSettings(): Promise<ServerSettings> {
  const { data } = await api.get<ServerSettings>('/api/admin/settings')
  return data
}

export async function patchAdminSettings(payload: PatchSettingsPayload): Promise<ServerSettings> {
  const { data } = await api.patch<ServerSettings>('/api/admin/settings', payload)
  return data
}
