import api from '@/api/client'
import type { ServerSettings, PatchSettingsPayload } from '@/types/api'

export async function getSettings(): Promise<ServerSettings> {
  const { data } = await api.get<ServerSettings>('/api/settings')
  return data
}

export async function patchSettings(payload: PatchSettingsPayload): Promise<ServerSettings> {
  const { data } = await api.patch<ServerSettings>('/api/settings', payload)
  return data
}
