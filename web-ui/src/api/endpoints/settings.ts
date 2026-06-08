import { apiClient } from '../client'
import type { ServerSettings, PatchSettingsPayload } from '@/types/api'

export const getSettings = () =>
  apiClient.get<ServerSettings>('/api/settings').then((r) => r.data)

export const patchSettings = (payload: PatchSettingsPayload) =>
  apiClient.patch<ServerSettings>('/api/settings', payload).then((r) => r.data)
