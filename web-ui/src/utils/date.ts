import dayjs from 'dayjs'
import 'dayjs/locale/ru'
import relativeTime from 'dayjs/plugin/relativeTime'

dayjs.extend(relativeTime)
dayjs.locale('ru')

export function formatDate(date: string | null | undefined): string {
  if (!date) return '—'
  return dayjs(date).format('DD.MM.YYYY')
}

export function formatDateTime(date: string | null | undefined): string {
  if (!date) return '—'
  return dayjs(date).format('DD.MM.YYYY HH:mm:ss')
}

export function formatRelative(date: string | null | undefined): string {
  if (!date) return '—'
  return dayjs(date).fromNow()
}

export function formatDateLong(date: string | null | undefined): string {
  if (!date) return '—'
  return dayjs(date).format('DD MMMM YYYY, HH:mm')
}
