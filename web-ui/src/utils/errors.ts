import type { AxiosError } from 'axios'

export function humanizeError(err: unknown): string {
  const axiosErr = err as AxiosError<{ error?: string }>
  const status = axiosErr.response?.status
  const serverMessage = axiosErr.response?.data?.error

  if (status === 401) return 'Сессия истекла — войдите снова'
  if (status === 403) return 'Недостаточно прав для этого действия'
  if (status === 404) return 'Ресурс не найден'
  if (status === 502) return 'Shlink недоступен, попробуйте позже'
  if (serverMessage) return serverMessage
  if (status && status >= 500) return 'Внутренняя ошибка сервера'
  if (!axiosErr.response) return 'Нет соединения с сервером'
  return 'Неизвестная ошибка'
}
