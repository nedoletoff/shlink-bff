import { useEffect, useState } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import {
  Stack, Title, Text, Card, Group, Badge,
  ActionIcon, Tooltip, Table, Skeleton, Center,
  Grid, Anchor,
} from '@mantine/core';
import { IconArrowLeft, IconUserCog } from '@tabler/icons-react';
import {
  LineChart, Line, XAxis, YAxis, CartesianGrid,
  Tooltip as RTooltip, ResponsiveContainer,
} from 'recharts';
import { api } from '../../api/client';
import { useAuth } from '../../contexts/AuthContext';
import { ErrorBoundary } from '../../components/ui/ErrorBoundary';
import { formatDate } from '../../utils/date';
import type { UserDetailResponse } from '../../types/api';

export function UserDetail() {
  const { id }       = useParams<{ id: string }>();
  const navigate     = useNavigate();
  const { user: me } = useAuth();

  const [data,    setData]    = useState<UserDetailResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [err,     setErr]     = useState<string | null>(null);

  useEffect(() => {
    if (!id) return;
    setLoading(true); setErr(null);
    api.get<UserDetailResponse>(`/api/admin/users/${encodeURIComponent(id)}`)
      .then(setData)
      .catch((e: unknown) => setErr(e instanceof Error ? e.message : 'Ошибка загрузки'))
      .finally(() => setLoading(false));
  }, [id]);

  if (loading) return (
    <Stack gap="lg">
      <Skeleton height={32} width={240} />
      <Skeleton height={100} radius="md" />
      <Skeleton height={240} radius="md" />
      <Skeleton height={300} radius="md" />
    </Stack>
  );

  if (err) return <Center h={300}><Text c="red">{err}</Text></Center>;
  if (!data) return null;

  const isSelf = data.sub === me?.sub;

  return (
    <Stack gap="lg">
      {/* Навигация */}
      <Group>
        <ActionIcon variant="subtle" onClick={() => navigate(-1)}>
          <IconArrowLeft size={18} />
        </ActionIcon>
        <Title order={2}>{data.username}</Title>
        <Badge
          ml={4}
          color={data.role === 'admin' ? 'red' : 'blue'}
          variant="light"
        >
          {data.role}
        </Badge>
      </Group>

      {/* Инфо-карточка */}
      <Card withBorder radius="md" p="lg">
        <Grid>
          <Grid.Col span={{ base: 12, sm: 4 }}>
            <Text size="xs" c="dimmed">Email</Text>
            <Text size="sm" fw={500}>{data.email}</Text>
          </Grid.Col>
          <Grid.Col span={{ base: 12, sm: 4 }}>
            <Text size="xs" c="dimmed">Ссылок</Text>
            <Text size="sm" fw={500}>{data.linksCount.toLocaleString('ru-RU')}</Text>
          </Grid.Col>
          <Grid.Col span={{ base: 12, sm: 4 }}>
            <Text size="xs" c="dimmed">Переходов всего</Text>
            <Text size="sm" fw={500}>{data.visitsTotal.toLocaleString('ru-RU')}</Text>
          </Grid.Col>
        </Grid>

        {/* Блок смены роли */}
        <Group mt="md">
          <Tooltip
            label={isSelf ? 'Нельзя изменить роль себе' : 'Изменить роль'}
            withArrow
          >
            <ActionIcon
              variant="light"
              color="blue"
              disabled={isSelf}
              onClick={() => navigate(`/admin/users?highlight=${data.sub}`)}
            >
              <IconUserCog size={16} />
            </ActionIcon>
          </Tooltip>
          <Text size="xs" c="dimmed">Изменить роль — перейти в управление пользователями</Text>
        </Group>
      </Card>

      {/* График активности */}
      <ErrorBoundary section="График">
        <Card withBorder radius="md" p="lg">
          <Text fw={600} mb="md">Активность по дням</Text>
          <ResponsiveContainer width="100%" height={220}>
            <LineChart data={data.activityPerDay}>
              <CartesianGrid strokeDasharray="3 3" stroke="#373A40" />
              <XAxis dataKey="date" tick={{ fontSize: 11 }} />
              <YAxis tick={{ fontSize: 11 }} />
              <RTooltip />
              <Line type="monotone" dataKey="clicks"
                stroke="#51cf66" strokeWidth={2} dot={false} />
            </LineChart>
          </ResponsiveContainer>
        </Card>
      </ErrorBoundary>

      {/* Таблица ссылок */}
      <ErrorBoundary section="Ссылки пользователя">
        <Card withBorder radius="md" p={0}>
          <Group px="lg" py="md">
            <Text fw={600}>Ссылки пользователя</Text>
          </Group>
          <Table striped highlightOnHover>
            <Table.Thead>
              <Table.Tr>
                <Table.Th>Название</Table.Th>
                <Table.Th>Короткая</Table.Th>
                <Table.Th>Переходов</Table.Th>
                <Table.Th>Создана</Table.Th>
              </Table.Tr>
            </Table.Thead>
            <Table.Tbody>
              {data.links.map(link => (
                <Table.Tr
                  key={link.shortCode}
                  style={{ cursor: 'pointer' }}
                  onClick={() => navigate(`/links/${link.shortCode}`)}
                >
                  <Table.Td>
                    <Text size="sm" fw={500} c={link.title ? undefined : 'dimmed'}>
                      {link.title || 'Без названия'}
                    </Text>
                  </Table.Td>
                  <Table.Td>
                    <Anchor
                      href={link.shortUrl} target="_blank"
                      size="sm" onClick={e => e.stopPropagation()}
                    >
                      {link.shortUrl}
                    </Anchor>
                  </Table.Td>
                  <Table.Td>{link.visitsSummary.total.toLocaleString('ru-RU')}</Table.Td>
                  <Table.Td><Text size="sm">{formatDate(link.dateCreated)}</Text></Table.Td>
                </Table.Tr>
              ))}
              {data.links.length === 0 && (
                <Table.Tr>
                  <Table.Td colSpan={4}>
                    <Center p="xl"><Text c="dimmed">Ссылок нет</Text></Center>
                  </Table.Td>
                </Table.Tr>
              )}
            </Table.Tbody>
          </Table>
        </Card>
      </ErrorBoundary>
    </Stack>
  );
}
