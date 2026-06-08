import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import {
  Stack, Title, Tabs, Table, Text, Switch,
  Skeleton, Group, Badge, Paper,
} from '@mantine/core'
import { notifications } from '@mantine/notifications'
import { getAdminRoles, updateRolePermissions } from '@/api/endpoints/adminRoles'
import type { RolePermissions } from '@/types/api'

const PERM_LABELS: Record<keyof Omit<RolePermissions, 'role' | 'updatedAt'>, string> = {
  canViewOwnLinks:           'Смотреть свои ссылки',
  canViewAllLinks:           'Смотреть все ссылки',
  canCreateLinks:            'Создавать ссылки',
  canCreateWithCustomSlug:   'Создавать с кастом. слагом',
  canCreateWithoutSlug:      'Создавать без слага',
  canEditOwnLinks:           'Редакт. свои ссылки',
  canEditAllLinks:           'Редакт. все ссылки',
  canDeleteOwnLinks:         'Удалять свои',
  canDeleteAllLinks:         'Удалять все',
  canDeactivateOwnLinks:     'Деакт. свои',
  canDeactivateAllLinks:     'Деакт. все',
  canReactivateOwnLinks:     'Реакт. свои',
  canReactivateAllLinks:     'Реакт. все',
  canDeleteOwnLinksPermanently:  'Удалить свои навсегда',
  canDeleteAllLinksPermanently:  'Удалить все навсегда',
  canManageOwnTags:          'Упр. своими тегами',
  canManageAllTags:          'Упр. всеми тегами',
  canViewOwnStats:           'Стат. своих ссылок',
  canViewAllStats:           'Стат. всех ссылок',
  canViewAuditLogs:          'Аудит-логи',
  canManageUsers:            'Управлять пользователями',
  canManageRoles:            'Управлять ролями',
}

export function AdminRoles() {
  const qc = useQueryClient()
  const { data, isLoading } = useQuery({
    queryKey: ['admin-roles'],
    queryFn: getAdminRoles,
  })

  const mutation = useMutation({
    mutationFn: ({ role, key, value }: { role: string; key: keyof Omit<RolePermissions, 'role' | 'updatedAt'>; value: boolean }) =>
      updateRolePermissions(role, { [key]: value }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['admin-roles'] })
      notifications.show({ color: 'teal', message: 'Разрешение обновлено' })
    },
  })

  const roles = data?.roles ?? []

  return (
    <Stack gap="md">
      <Title order={2}>Роли и разрешения</Title>

      {isLoading ? (
        <Skeleton height={400} />
      ) : (
        <Tabs defaultValue={roles[0]?.role}>
          <Tabs.List>
            {roles.map((r) => (
              <Tabs.Tab key={r.role} value={r.role}>
                <Group gap="xs">
                  <Text size="sm">{r.role}</Text>
                  <Badge size="xs" variant="light">{r.permissions.length}</Badge>
                </Group>
              </Tabs.Tab>
            ))}
          </Tabs.List>

          {roles.map((roleEntry) => {
            const perms = roleEntry.permissions
            return (
              <Tabs.Panel key={roleEntry.role} value={roleEntry.role} pt="md">
                <Paper withBorder radius="md">
                  <Table striped highlightOnHover>
                    <Table.Thead>
                      <Table.Tr>
                        <Table.Th>Разрешение</Table.Th>
                        <Table.Th w={80}>Вкл</Table.Th>
                      </Table.Tr>
                    </Table.Thead>
                    <Table.Tbody>
                      {(Object.keys(PERM_LABELS) as Array<keyof typeof PERM_LABELS>).map((key) => (
                        <Table.Tr key={key}>
                          <Table.Td><Text size="sm">{PERM_LABELS[key]}</Text></Table.Td>
                          <Table.Td>
                            <Switch
                              checked={perms.includes(key)}
                              onChange={(e) =>
                                mutation.mutate({ role: roleEntry.role, key, value: e.currentTarget.checked })
                              }
                              disabled={mutation.isPending}
                            />
                          </Table.Td>
                        </Table.Tr>
                      ))}
                    </Table.Tbody>
                  </Table>
                </Paper>
              </Tabs.Panel>
            )
          })}
        </Tabs>
      )}
    </Stack>
  )
}
