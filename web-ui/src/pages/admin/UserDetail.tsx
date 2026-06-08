import { useParams, useNavigate } from 'react-router-dom'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import {
  Stack, Group, Title, Text, Button, Paper,
  Select, TextInput, PasswordInput, Skeleton,
  Divider,
} from '@mantine/core'
import { useForm } from '@mantine/form'
import { notifications } from '@mantine/notifications'
import { IconArrowLeft } from '@tabler/icons-react'
import { getAdminUser, updateAdminUser, updateApiKey } from '@/api/endpoints/adminUsers'
import { RoleBadge } from '@/components/ui/RoleBadge'
import { StatusBadge } from '@/components/ui/StatusBadge'

export function AdminUserDetail() {
  const { sub } = useParams<{ sub: string }>()
  const navigate = useNavigate()
  const qc = useQueryClient()

  const { data: user, isLoading } = useQuery({
    queryKey: ['admin-user', sub],
    queryFn: () => getAdminUser(sub!),
    enabled: Boolean(sub),
  })

  const form = useForm({
    initialValues: { role: '', status: '', slugPrefix: '', apiKey: '' },
  })

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

  if (!isLoading && user && !form.isDirty()) {
    form.setValues({
      role: user.role,
      status: user.status,
      slugPrefix: user.slugPrefix ?? '',
      apiKey: '',
    })
  }

  return (
    <Stack gap="lg">
      <Group>
        <Button variant="subtle" size="sm" leftSection={<IconArrowLeft size={14} />} onClick={() => navigate('/admin/users')}>
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

      <Paper withBorder p="md" radius="md">
        <Text fw={600} mb="md">Основное</Text>
        <Stack gap="sm">
          <TextInput label="Email" value={user?.email ?? ''} readOnly />
          <TextInput label="Sub" value={user?.sub ?? ''} readOnly />
          <Select
            label="Роль"
            data={['admin', 'moderator', 'user', 'viewer']}
            {...form.getInputProps('role')}
          />
          <Select
            label="Статус"
            data={['active', 'inactive', 'pending', 'disabled']}
            {...form.getInputProps('status')}
          />
          <TextInput label="Префикс слага" {...form.getInputProps('slugPrefix')} />
          <Group justify="flex-end">
            <Button onClick={() => updateMutation.mutate()} loading={updateMutation.isPending}>
              Сохранить
            </Button>
          </Group>
        </Stack>
      </Paper>

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
