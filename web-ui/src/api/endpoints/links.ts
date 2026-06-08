import { apiClient } from '../client'
import type { ShortURL, ShortURLsResponse, CreateShortURLPayload, EditShortURLPayload } from '@/types/api'

export interface ListLinksParams {
  page?: number
  itemsPerPage?: number
  searchTerm?: string
  status?: 'active' | 'inactive' | 'all'
}

export const getLinks = (params: ListLinksParams) =>
  apiClient.get<ShortURLsResponse>('/api/shlink/short-urls', { params }).then((r) => r.data)

export const createLink = (payload: CreateShortURLPayload) =>
  apiClient.post<ShortURL>('/api/shlink/short-urls', payload).then((r) => r.data)

export const editLink = (shortCode: string, domain: string, payload: EditShortURLPayload) =>
  apiClient.patch<ShortURL>(`/api/shlink/short-urls/${shortCode}`, payload, { params: { domain } }).then((r) => r.data)

export const deactivateLink = (shortCode: string, domain: string) =>
  apiClient.post(`/api/shlink/short-urls/${shortCode}/deactivate`, null, { params: { domain } }).then((r) => r.data)

export const activateLink = (shortCode: string, domain: string) =>
  apiClient.post(`/api/shlink/short-urls/${shortCode}/activate`, null, { params: { domain } }).then((r) => r.data)

export const deleteLink = (shortCode: string, domain: string) =>
  apiClient.delete(`/api/shlink/short-urls/${shortCode}`, { params: { domain } })
