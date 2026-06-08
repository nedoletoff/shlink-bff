import api from '@/api/client'
import type { ShortURLsResponse, CreateShortURLPayload, UpdateShortURLPayload } from '@/types/api'

export interface ListLinksParams {
  page?: number
  itemsPerPage?: number
  searchTerm?: string
  status?: string
}

export async function listLinks(params: ListLinksParams = {}): Promise<ShortURLsResponse> {
  const { data } = await api.get<ShortURLsResponse>('/api/shlink/short-urls', { params })
  return data
}

export async function createLink(payload: CreateShortURLPayload) {
  const { data } = await api.post('/api/shlink/short-urls', payload)
  return data
}

export async function updateLink(shortCode: string, payload: UpdateShortURLPayload) {
  const { data } = await api.patch(`/api/shlink/short-urls/${shortCode}`, payload)
  return data
}

export async function deleteLink(shortCode: string) {
  await api.delete(`/api/shlink/short-urls/${shortCode}`)
}

export async function deactivateLink(shortCode: string) {
  await api.post(`/api/shlink/short-urls/${shortCode}/deactivate`)
}

export async function activateLink(shortCode: string) {
  await api.post(`/api/shlink/short-urls/${shortCode}/activate`)
}
