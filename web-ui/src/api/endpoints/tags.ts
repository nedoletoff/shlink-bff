import { apiClient } from '@/api/client'
import type { TagEntry } from '@/types/api'

export const getTags = () =>
  apiClient.get<TagEntry[]>('/tags').then((r) => r.data)

export const createTag = (tag: string) =>
  apiClient.post<TagEntry>('/tags', { tag }).then((r) => r.data)

export const deleteTag = (tag: string) =>
  apiClient.delete(`/tags/${encodeURIComponent(tag)}`).then((r) => r.data)

export const renameTag = (oldTag: string, newTag: string) =>
  apiClient.patch(`/tags/${encodeURIComponent(oldTag)}`, { tag: newTag }).then((r) => r.data)
