/**
 * Утилиты для форматирования дат.
 * Все компоненты должны использовать эти функции вместо инлайнового new Date().
 */

/** ДД.ММ.ГГГГ ЧЧ:ММ — для таблиц с timestamp */
export function formatDateTime(value: string | null | undefined): string {
  if (value == null || value === '') return '—';
  const d = new Date(value);
  if (isNaN(d.getTime())) return String(value); // невалидная строка — показываем как есть
  return d.toLocaleString('ru-RU', {
    day:    '2-digit',
    month:  '2-digit',
    year:   'numeric',
    hour:   '2-digit',
    minute: '2-digit',
  });
}

/** ДД.ММ.ГГГГ — для колонок «Создана» */
export function formatDate(value: string | null | undefined): string {
  if (value == null || value === '') return '—';
  const d = new Date(value);
  if (isNaN(d.getTime())) return String(value);
  return d.toLocaleDateString('ru-RU', {
    day:   '2-digit',
    month: '2-digit',
    year:  'numeric',
  });
}
