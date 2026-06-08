import api from '@/api/client'
import type { RolesResponse, RolePermissions } from '@/types/api'

export async function getRoles(): Promise<RolesResponse> {
  const { data } = await api.get<RolesResponse>('/api/admin/roles')
  return data
}

export async function updateRolePermissions(role: string, permissions: Partial<RolePermissions>) {
  const { data } = await api.put(`/api/admin/roles/${role}/permissions`, permissions)
  return data
}
