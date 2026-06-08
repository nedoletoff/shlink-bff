import { apiClient } from '../client'
import type { RolesResponse, RolePermissions } from '@/types/api'

export const getAdminRoles = () =>
  apiClient.get<RolesResponse>('/api/admin/roles').then((r) => r.data)

export const updateRolePermissions = (role: string, permissions: Partial<Omit<RolePermissions, 'role' | 'updatedAt'>>) =>
  apiClient.put(`/api/admin/roles/${role}/permissions`, permissions)
