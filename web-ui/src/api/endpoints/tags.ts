import { apiClient } from '../client'
import type { TagEntry } from '@/types/api'

export const getTags = () =>
  apiClient.get<{ tags: { data: TagEntry[] } }>('/api/shlink/tags').then((r) => r.data)

export const renameTag = (tagId: string, newName: string) =>
  apiClient.put(`/api/shlink/tags/${tagId}`, { name: newName })

export const deleteTag = (tagName: string) =>
  apiClient.delete(`/api/shlink/tags/${encodeURIComponent(tagName)}`)
