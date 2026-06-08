import { useEffect } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import {
  Stack, Title, Paper, SimpleGrid, TextInput,
  NumberInput, Switch, Button, Group, Skeleton, Text,
  Divider, Badge,
} from '@mantine/core'
import { useForm } from '@mantine/form'
import { notifications } from '@mantine/notifications'
import { getAdminSettings, patchAdminSettings } from '@/api/endpoints/adminSettings'
import type { PatchSettingsPayload } from '@/types/api'

export function AdminSettings() {
  const qc = useQueryClient()

  const { data, isLoading } = useQuery({
    queryKey: ['admin-settings'],
    queryFn: getAdminSettings,
  })

  const form = useForm<PatchSettingsPayload>({
    initialValues: {
      shortCodeLength: 6,
      allowCustomSlugs: true,
      userSlugPrefix: false,
      domain: '',
      maxVisitsDefault: 0,
      linkTtlDefaultDays: 0,
      adminRole: '',
      shlinkRunnerMode: '',
      shlinkContainerName: '',
    },
  })

  useEffect(() => {
    if (data) {
      form.setValues({
        shortCodeLength: data.shortCodeLength,
        allowCustomSlugs: data.allowCustomSlugs,
        userSlugPrefix: data.userSlugPrefix,
        domain: data.domain,
        maxVisitsDefault: data.maxVisitsDefault,
        linkTtlDefaultDays: data.linkTtlDefaultDays,
        adminRole: data.adminRole,
        shlinkRunnerMode: data.shlinkRunnerMode,
        shlinkContainerName: data.shlinkContainerName,
      })
    }
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [data])

  const mutation = useMutation({
    mutationFn: (values: PatchSettingsPayload) => patchAdminSettings(values),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['admin-settings'] })
      notifications.show({ color: 'teal', message: 'Настройки сохранены' })
    },
  })

  return (
    <Stack gap="lg">
      <Group justify="space-between">
        <Title order={2}>Настройки сервера</Title>
        {data && (
          <Badge color={data.connected ? 'teal' : 'red'} variant="light">
            Shlink {data.connected ? 'подключён' : 'недоступен'}
          </Badge>
        )}
      </Group>

      {isLoading ? (
        <Skeleton height={500} />
      ) : (
        <form onSubmit={form.onSubmit((v: PatchSettingsPayload) => mutation.mutate(v))}>
          <Stack gap="md">
            <Paper withBorder p="md" radius="md">
              <Text fw={600} mb="md">Shlink</Text>
              <SimpleGrid cols={{ base: 1, sm: 2 }} spacing="sm">
                <TextInput label="Домен" {...form.getInputProps('domain')} />
                <TextInput label="Версия Shlink" value={data?.shlinkVersion ?? ''} readOnly />
                <NumberInput label="Длина кода по умолчанию" min={4} max={12} {...form.getInputProps('shortCodeLength')} />
                <NumberInput label="Макс. переходов по умолчанию" {...form.getInputProps('maxVisitsDefault')} />
                <NumberInput label="TTL ссылок (дней)" {...form.getInputProps('linkTtlDefaultDays')} />
              </SimpleGrid>
              <Group mt="sm" gap="lg">
                <Switch label="Кастомные слаги" {...form.getInputProps('allowCustomSlugs', { type: 'checkbox' })} />
                <Switch label="Префикс пользователя" {...form.getInputProps('userSlugPrefix', { type: 'checkbox' })} />
              </Group>
            </Paper>

            <Divider />

            <Paper withBorder p="md" radius="md">
              <Text fw={600} mb="md">Инфраструктура</Text>
              <SimpleGrid cols={{ base: 1, sm: 2 }} spacing="sm">
                <TextInput label="Роль админа" {...form.getInputProps('adminRole')} />
                <TextInput label="Режим запуска Shlink" {...form.getInputProps('shlinkRunnerMode')} />
                <TextInput label="Имя контейнера" {...form.getInputProps('shlinkContainerName')} />
                <TextInput label="Источник ролей" value={data?.roleSource ?? ''} readOnly />
                <TextInput label="CORS разрешённые оригины" value={data?.corsAllowedOrigins ?? ''} readOnly />
              </SimpleGrid>
            </Paper>

            <Group justify="flex-end">
              <Button type="submit" loading={mutation.isPending}>Сохранить</Button>
            </Group>
          </Stack>
        </form>
      )}
    </Stack>
  )
}
