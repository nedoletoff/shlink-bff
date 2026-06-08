import { useEffect } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import {
  Stack, Title, Paper, SimpleGrid, TextInput,
  NumberInput, Switch, Button, Group, Skeleton, Text,
  Divider, Badge, Tooltip, Alert,
} from '@mantine/core'
import { useForm } from '@mantine/form'
import { notifications } from '@mantine/notifications'
import { IconInfoCircle, IconAlertTriangle } from '@tabler/icons-react'
import { getAdminSettings, patchAdminSettings } from '@/api/endpoints/adminSettings'
import type { PatchSettingsPayload } from '@/types/api'

function FieldLabel({ label, hint }: { label: string; hint: string }) {
  return (
    <Group gap={4} wrap="nowrap">
      <Text size="sm" fw={500}>{label}</Text>
      <Tooltip label={hint} multiline w={220} withArrow position="top-start">
        <IconInfoCircle size={14} style={{ color: 'var(--mantine-color-dimmed)', cursor: 'help' }} />
      </Tooltip>
    </Group>
  )
}

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
      form.resetDirty()
    }
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [data])

  const mutation = useMutation({
    mutationFn: (values: PatchSettingsPayload) => patchAdminSettings(values),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['admin-settings'] })
      notifications.show({ color: 'teal', message: 'Настройки сохранены' })
      form.resetDirty()
    },
  })

  return (
    <Stack gap="lg">
      <Group justify="space-between">
        <Title order={2}>Настройки сервера</Title>
        {data && (
          <Badge color={data.connected ? 'teal' : 'red'} variant="light" size="lg">
            Shlink {data.connected ? 'подключён' : 'недоступен'}
          </Badge>
        )}
      </Group>

      {!isLoading && data && !data.connected && (
        <Alert icon={<IconAlertTriangle size={16} />} color="orange" title="Shlink недоступен">
          Проверьте конфигурацию инфраструктуры и перезапустите сервис.
        </Alert>
      )}

      {form.isDirty() && (
        <Alert icon={<IconInfoCircle size={16} />} color="blue">
          Есть несохранённые изменения. Не забудьте нажать «Сохранить».
        </Alert>
      )}

      {isLoading ? (
        <SimpleGrid cols={{ base: 1, sm: 2 }} spacing="md">
          {Array.from({ length: 6 }).map((_, i) => <Skeleton key={i} height={70} radius="sm" />)}
        </SimpleGrid>
      ) : (
        <form onSubmit={form.onSubmit((v: PatchSettingsPayload) => mutation.mutate(v))}>
          <Stack gap="md">

            {/* Shlink */}
            <Paper withBorder p="md" radius="md">
              <Group gap="xs" mb="md">
                <Text fw={600}>Shlink</Text>
                {data?.shlinkVersion && (
                  <Badge size="xs" variant="outline" color="gray">v{data.shlinkVersion}</Badge>
                )}
              </Group>
              <SimpleGrid cols={{ base: 1, sm: 2 }} spacing="sm">
                <TextInput
                  label={<FieldLabel label="Домен" hint="Базовый домен для создаваемых коротких ссылок" />}
                  placeholder="https://example.com"
                  {...form.getInputProps('domain')}
                />
                <TextInput
                  label={<FieldLabel label="Версия Shlink" hint="Автоматически определяется из API" />}
                  value={data?.shlinkVersion ?? '—'}
                  readOnly
                />
                <NumberInput
                  label={<FieldLabel label="Длина кода" hint="Количество символов в автогенерируемом short code" />}
                  min={4} max={12}
                  {...form.getInputProps('shortCodeLength')}
                />
                <NumberInput
                  label={<FieldLabel label="Макс. переходов" hint="0 — без ограничений" />}
                  min={0}
                  {...form.getInputProps('maxVisitsDefault')}
                />
                <NumberInput
                  label={<FieldLabel label="TTL ссылок (дней)" hint="0 — ссылки хранятся бессрочно" />}
                  min={0}
                  {...form.getInputProps('linkTtlDefaultDays')}
                />
              </SimpleGrid>
              <Group mt="md" gap="xl">
                <Switch
                  label={<FieldLabel label="Кастомные слаги" hint="Разрешить пользователям задавать свой short code" />}
                  {...form.getInputProps('allowCustomSlugs', { type: 'checkbox' })}
                />
                <Switch
                  label={<FieldLabel label="Префикс пользователя" hint="Добавляет username/ перед каждым слагом" />}
                  {...form.getInputProps('userSlugPrefix', { type: 'checkbox' })}
                />
              </Group>
            </Paper>

            <Divider />

            {/* Keycloak (read-only) */}
            <Paper withBorder p="md" radius="md">
              <Text fw={600} mb="md">Keycloak</Text>
              <SimpleGrid cols={{ base: 1, sm: 2 }} spacing="sm">
                <TextInput
                  label={<FieldLabel label="Источник ролей" hint="roleSource: JWT | Keycloak API" />}
                  value={data?.roleSource ?? '—'}
                  readOnly
                />
                <TextInput
                  label={<FieldLabel label="Роль админа" hint="Название роли в Keycloak, отвечающей за доступ админа" />}
                  {...form.getInputProps('adminRole')}
                />
                <TextInput
                  label={<FieldLabel label="CORS оригины" hint="Разрешённые оригины для CORS-запросов" />}
                  value={data?.corsAllowedOrigins ?? '—'}
                  readOnly
                />
              </SimpleGrid>
            </Paper>

            <Divider />

            {/* Infrastructure */}
            <Paper withBorder p="md" radius="md">
              <Text fw={600} mb="md">Инфраструктура</Text>
              <SimpleGrid cols={{ base: 1, sm: 2 }} spacing="sm">
                <TextInput
                  label={<FieldLabel label="Режим запуска Shlink" hint="docker | process" />}
                  {...form.getInputProps('shlinkRunnerMode')}
                />
                <TextInput
                  label={<FieldLabel label="Имя контейнера" hint="Используется если shlinkRunnerMode = docker" />}
                  {...form.getInputProps('shlinkContainerName')}
                />
              </SimpleGrid>
            </Paper>

            <Group justify="flex-end" gap="sm">
              <Button
                variant="default"
                onClick={() => { form.reset(); if (data) form.setValues(data as PatchSettingsPayload) }}
                disabled={!form.isDirty()}
              >
                Сбросить
              </Button>
              <Button type="submit" loading={mutation.isPending} disabled={!form.isDirty()}>
                Сохранить
              </Button>
            </Group>
          </Stack>
        </form>
      )}
    </Stack>
  )
}
