import { apiClient } from '../client'
import type { ServerSettings, PatchSettingsPayload } from '@/types/api'

export const getAdminSettings = () =>
  apiClient.get<ServerSettings>('/api/admin/settings').then((r) => r.data)

export const patchAdminSettings = (payload: PatchSettingsPayload) =>
  apiClient.patch<ServerSettings>('/api/admin/settings', payload).then((r) => r.data)
