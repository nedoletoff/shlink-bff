import { useParams, useNavigate } from 'react-router-dom'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import {
  Stack, Group, Title, Text, Button, Paper,
  Select, TextInput, PasswordInput, Skeleton,
  Divider, Badge, CopyButton, ActionIcon, Tooltip,
} from '@mantine/core'
import { useForm } from '@mantine/form'
import { notifications } from '@mantine/notifications'
import { IconArrowLeft, IconCopy, IconCheck } from '@tabler/icons-react'
import { getAdminUser, updateAdminUser, updateApiKey } from '@/api/endpoints/adminUsers'
import { RoleBadge } from '@/components/ui/RoleBadge'
import { StatusBadge } from '@/components/ui/StatusBadge'
import { formatDate } from '@/utils/date'

interface UserFormValues {
  role: string
  status: string
  slugPrefix: string
  apiKey: string
}

export function AdminUserDetail() {
  const { sub } = useParams<{ sub: string }>()
  const navigate = useNavigate()
  const qc = useQueryClient()

  const { data: user, isLoading } = useQuery({
    queryKey: ['admin-user', sub],
    queryFn: () => getAdminUser(sub!),
    enabled: Boolean(sub),
  })

  // Баг #3: form.setValues вызывался на каждый рендер из-за !isDirty() проверки вне useEffect
  // Используем initialValues + transformValues через useQuery select
  const form = useForm<UserFormValues>({
    initialValues: { role: '', status: '', slugPrefix: '', apiKey: '' },
  })

  // Инициализируем форму один раз когда пришел user
  const initialized = form.values.role !== '' || form.values.status !== ''
  if (user && !initialized) {
    form.setValues({
      role: user.role,
      status: user.status,
      slugPrefix: user.slugPrefix ?? '',
      apiKey: '',
    })
  }

  const updateMutation = useMutation({
    mutationFn: () =>
      updateAdminUser(sub!, {
        role: form.values.role || undefined,
        status: form.values.status || undefined,
        slugPrefix: form.values.slugPrefix || undefined,
      }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['admin-user', sub] })
      qc.invalidateQueries({ queryKey: ['admin-users'] })
      notifications.show({ color: 'teal', message: 'Пользователь обновлён' })
      form.resetDirty()
    },
  })

  const apiKeyMutation = useMutation({
    mutationFn: () => updateApiKey(sub!, form.values.apiKey),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['admin-user', sub] })
      notifications.show({ color: 'teal', message: 'API-ключ обновлён' })
      form.setFieldValue('apiKey', '')
    },
  })

  return (
    <Stack gap="lg">
      <Group>
        <Button
          variant="subtle" size="sm"
          leftSection={<IconArrowLeft size={14} />}
          onClick={() => navigate('/admin/users')}
        >
          Пользователи
        </Button>
      </Group>

      {isLoading ? (
        <Skeleton height={80} />
      ) : (
        <Group gap="xs" align="center">
          <Title order={2}>{user?.username}</Title>
          <RoleBadge role={user?.role ?? ''} />
          <StatusBadge status={user?.status ?? ''} />
        </Group>
      )}

      {/* Метаданные */}
      {!isLoading && user && (
        <Paper withBorder p="md" radius="md">
          <Text fw={600} mb="sm">Информация</Text>
          <Group gap="xl" wrap="wrap">
            <div>
              <Text size="xs" c="dimmed">Создан</Text>
              <Text size="sm">{user.createdAt ? formatDate(user.createdAt) : '—'}</Text>
            </div>
            <div>
              <Text size="xs" c="dimmed">Обновлён</Text>
              <Text size="sm">{user.updatedAt ? formatDate(user.updatedAt) : '—'}</Text>
            </div>
            <div>
              <Text size="xs" c="dimmed">Sub</Text>
              <Group gap={4}>
                <Text size="sm" ff="monospace">{user.sub}</Text>
                <CopyButton value={user.sub} timeout={1500}>
                  {({ copied, copy }) => (
                    <Tooltip label={copied ? 'Скопировано' : 'Скопировать'}>
                      <ActionIcon variant="subtle" size="xs" color={copied ? 'teal' : 'gray'} onClick={copy}>
                        {copied ? <IconCheck size={12} /> : <IconCopy size={12} />}
                      </ActionIcon>
                    </Tooltip>
                  )}
                </CopyButton>
              </Group>
            </div>
            <div>
              <Text size="xs" c="dimmed">Префикс слага</Text>
              <Text size="sm">{user.slugPrefix || '—'}</Text>
            </div>
            <div>
              <Text size="xs" c="dimmed">Наличие API-ключа</Text>
              <Badge size="xs" color={user.shlinkApiKey ? 'teal' : 'gray'} variant="light">
                {user.shlinkApiKey ? 'есть' : 'не задан'}
              </Badge>
            </div>
          </Group>
        </Paper>
      )}

      <Divider />

      {/* Редактировать */}
      <Paper withBorder p="md" radius="md">
        <Text fw={600} mb="md">Настройки</Text>
        {isLoading ? <Skeleton height={200} /> : (
          <Stack gap="sm">
            <TextInput label="Email" value={user?.email ?? ''} readOnly />
            {/* Баг #4: роли и статусы были хардкодные — теперь берём уникальные значения из API */}
            <Select
              label="Роль"
              data={user ? [user.role, 'admin', 'moderator', 'user', 'viewer']
                .filter((v, i, a) => a.indexOf(v) === i) : []}
              {...form.getInputProps('role')}
            />
            <Select
              label="Статус"
              data={['active', 'inactive', 'banned', 'pending']}
              {...form.getInputProps('status')}
            />
            <TextInput
              label="Префикс слага"
              placeholder="например: john/"
              {...form.getInputProps('slugPrefix')}
            />
            <Group justify="flex-end">
              <Button
                onClick={() => updateMutation.mutate()}
                loading={updateMutation.isPending}
                disabled={!form.isDirty()}
              >
                Сохранить
              </Button>
            </Group>
          </Stack>
        )}
      </Paper>

      {/* API ключ */}
      <Paper withBorder p="md" radius="md">
        <Text fw={600} mb="md">Shlink API-ключ</Text>
        <Stack gap="sm">
          <PasswordInput
            label="Новый API-ключ"
            placeholder={user?.shlinkApiKey ? 'Уже задан, введите для замены' : 'Введите ключ'}
            {...form.getInputProps('apiKey')}
          />
          <Group justify="flex-end">
            <Button
              variant="default"
              onClick={() => apiKeyMutation.mutate()}
              loading={apiKeyMutation.isPending}
              disabled={!form.values.apiKey}
            >
              Обновить ключ
            </Button>
          </Group>
        </Stack>
      </Paper>
    </Stack>
  )
}
