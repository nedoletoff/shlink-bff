import dayjs from 'dayjs'
import relativeTime from 'dayjs/plugin/relativeTime'
import 'dayjs/locale/ru'

dayjs.extend(relativeTime)
dayjs.locale('ru')

export const formatDate = (iso: string) => dayjs(iso).format('DD.MM.YYYY')

export const formatDateTime = (iso: string) => dayjs(iso).format('DD.MM.YYYY HH:mm:ss')

export const formatRelative = (iso: string) => dayjs(iso).fromNow()
