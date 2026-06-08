import axios, { type AxiosError } from 'axios'
import { notifications } from '@mantine/notifications'

export const apiClient = axios.create({ baseURL: '/' })

function humanizeError(err: AxiosError): string {
  const status = err.response?.status
  const serverMessage = (err.response?.data as Record<string, string> | undefined)?.error

  if (status === 401) return 'Сессия истекла — войдите снова'
  if (status === 403) return 'Недостаточно прав для этого действия'
  if (status === 404) return 'Ресурс не найден'
  if (status === 502) return 'Shlink недоступен, попробуйте позже'
  if (serverMessage) return serverMessage
  if (status && status >= 500) return 'Внутренняя ошибка сервера'
  if (!err.response) return 'Нет соединения с сервером'
  return 'Неизвестная ошибка'
}

apiClient.interceptors.response.use(
  (res) => res,
  (err: AxiosError) => {
    const message = humanizeError(err)
    notifications.show({ color: 'red', message, autoClose: 5000 })
    if (err.response?.status === 401) {
      window.location.href = '/auth/login'
    }
    return Promise.reject(err)
  },
)
