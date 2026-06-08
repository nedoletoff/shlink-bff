import axios from 'axios'
import { notifications } from '@mantine/notifications'
import { humanizeError } from '@/utils/errors'

const api = axios.create({
  baseURL: '/',
  headers: { 'Content-Type': 'application/json' },
})

api.interceptors.response.use(
  (res) => res,
  (err) => {
    const status = err?.response?.status
    const message = humanizeError(err)

    if (status === 401) {
      window.location.href = '/auth/login'
    }

    notifications.show({
      color: 'red',
      message,
      autoClose: 5000,
    })

    return Promise.reject(err)
  },
)

export default api
