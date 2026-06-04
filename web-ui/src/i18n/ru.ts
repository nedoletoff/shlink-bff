/**
 * Все пользовательские тексты интерфейса.
 * Изменяйте здесь — не разбрасывайте строки по файлам.
 */
export const RU = {
  // Общее
  save:           'Сохранить',
  cancel:         'Отмена',
  delete:         'Удалить',
  edit:           'Редактировать',
  create:         'Создать',
  loading:        'Загрузка...',
  noData:         'Ничего не найдено',
  error:          'Ошибка',
  confirm:        'Подтвердить',
  yes:            'Да',
  no:             'Нет',
  copy:           'Копировать',
  copied:         'Скопировано!',
  close:          'Закрыть',
  notFound:       'Не найдено',

  // Навигация
  nav: {
    dashboard:    'Дашборд',
    links:        'Мои ссылки',
    tags:         'Теги',
    users:        'Пользователи',
    roles:        'Роли',
    audit:        'Журнал аудита',
    settings:     'Настройки',
  },

  // Ссылки
  links: {
    title:          'Название',
    noTitle:        'Без названия',
    destination:    'Куда ведёт',
    shortUrl:       'Короткая ссылка',
    customSlug:     'Кастомная ссылка',
    customSlugHint: 'Придумайте короткое слово, например: konferencia',
    visits:         'Переходов',
    created:        'Создана',
    actions:        'Действия',
    createLink:     'Создать ссылку',
    longUrlLabel:   'Длинная ссылка',
    longUrlHint:    'Адрес страницы, на которую будет вести короткая ссылка',
    titleHint:      'Как вы будете её узнавать в списке',
    tags:           'Теги',
    deleteConfirm:  (code: string) =>
      `Ссылка «${code}» будет удалена безвозвратно. Все переходы перестанут работать.`,
  },

  // Роли
  roles: {
    title:          'Управление ролями',
    role:           'Роль',
    permissions:    'Разрешения',
    usersCount:     'Пользователей',
    addMapping:     'Добавить маппинг',
    kcGroup:        'Группа Keycloak',
    appRole:        'Роль приложения',
    changeRole:     'Изменить роль',
    selfGuard:      'Нельзя снять роль с самого себя',
    confirmDemotion: 'Вы снимаете админ-роль. После сохранения доступ в админ-панель будет закрыт. Продолжить?',
    noRoles:        'Роли не найдены',
  },

  // Настройки
  settings: {
    title:            'Настройки Shlink',
    generation:       'Генерация коротких ссылок',
    shortCodeLength:  'Длина кода',
    shortCodeHint:    'Число от 4 до 10',
    allowCustomSlugs: 'Разрешить кастомные ссылки',
    userSlugPrefix:   'Добавлять префикс пользователя',
    domain:           'Домен',
    domainHint:       'Значение DEFAULT_DOMAIN из конфига, только чтение',
    apiSection:       'API',
    shlinkVersion:    'Версия Shlink',
    connectionStatus: 'Статус подключения',
    connected:        'Подключено',
    disconnected:     'Нет связи',
    saved:            'Настройки сохранены',
  },

  // Пользователи
  users: {
    title:         'Управление пользователями',
    user:          'Пользователь',
    email:         'Email',
    role:          'Роль',
    prefix:        'Префикс',
    status:        'Статус',
    apiKey:        'API Key',
    notFound:      'Пользователи не найдены',
  },

  // Дашборд
  dashboard: {
    title:          'Дашборд',
    totalClicks:    'Всего кликов',
    activeLinks:    'Активных ссылок',
    topTag:         'Топ тег',
    clicksOverTime: 'Клики по времени',
    tagDistrib:     'Распределение по тегам',
    loadError:      'Ошибка загрузки дашборда',
    noData:         'Нет данных дашборда',
  },
} as const;
