import api from '@/api/client'
import type { TagEntry } from '@/types/api'

export async function getTags(): Promise<TagEntry[]> {
  const { data } = await api.get<TagEntry[]>('/api/shlink/tags')
  return data
}

export async function renameTag(tagId: string, newName: string) {
  await api.put(`/api/shlink/tags/${tagId}`, { newName })
}

export async function deleteTag(tagName: string) {
  await api.delete(`/api/shlink/tags/${encodeURIComponent(tagName)}`)
}
