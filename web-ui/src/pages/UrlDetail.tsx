import { useEffect, useState } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import {
  Stack, Title, Text, Card, Group, Anchor,
  ActionIcon, Tooltip, SegmentedControl, Table,
  Skeleton, Center, Grid, CopyButton, Pagination,
  Badge, Alert, Button,
} from '@mantine/core';
import {
  IconArrowLeft, IconCopy, IconCheck,
  IconEdit, IconTrash, IconBan, IconPlayerPlay,
  IconAlertTriangle,
} from '@tabler/icons-react';
import {
  LineChart, Line, XAxis, YAxis, CartesianGrid,
  Tooltip as RTooltip, ResponsiveContainer,
  PieChart, Pie, Cell, Legend,
} from 'recharts';
import { api } from '../api/client';
import { useAuth } from '../contexts/AuthContext';
import { ErrorBoundary } from '../components/ui/ErrorBoundary';
import { formatDate, formatDateTime } from '../utils/date';
import type { UrlDetailResponse } from '../types/api';

const COLORS = ['#4dabf7', '#51cf66', '#ff6b6b', '#ffd43b', '#cc5de8'];

const PERIOD_OPTIONS = [
  { label: '7 д',  value: '7'  },
  { label: '30 д', value: '30' },
  { label: '90 д', value: '90' },
];

const PAGE_SIZE = 20;

export function UrlDetail() {
  const { shortCode } = useParams<{ shortCode: string }>();
  const navigate = useNavigate();
  const { user } = useAuth();

  const [data,    setData]    = useState<UrlDetailResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [err,     setErr]     = useState<string | null>(null);
  const [period,  setPeriod]  = useState('30');
  const [page,    setPage]    = useState(1);

  const perms = user?.permissions;
  const canDeactivate = perms?.canDeactivateOwnLinks || perms?.canDeactivateAllLinks;
  const canReactivate = perms?.canReactivateOwnLinks || perms?.canReactivateAllLinks;
  const canPermDelete = perms?.canDeleteOwnLinksPermanently || perms?.canDeleteAllLinksPermanently;

  const loadData = () => {
    if (!shortCode) return;
    setLoading(true);
    setErr(null);
    api.get<UrlDetailResponse>(`/api/urls/${shortCode}/detail`, { params: { period } })
      .then(setData)
      .catch((e: Error) => setErr(e.message ?? 'Ошибка загрузки'))
      .finally(() => setLoading(false));
  };

  useEffect(() => { loadData(); }, [shortCode, period]);

  const handleDeactivate = async () => {
    if (!shortCode) return;
    if (!confirm('Деактивировать ссылку? Она перестанет работать.')) return;
    try {
      await api.post(`/api/shlink/short-urls/${shortCode}/deactivate`, {});
      loadData();
    } catch { /* shown */ }
  };

  const handleActivate = async () => {
    if (!shortCode) return;
    if (!confirm('Активировать ссылку? Она снова начнёт работать.')) return;
    try {
      await api.post(`/api/shlink/short-urls/${shortCode}/activate`, {});
      loadData();
    } catch { /* shown */ }
  };

  const handlePermanentDelete = async () => {
    if (!shortCode) return;
    if (!confirm('Удалить ссылку БЕЗВОЗВРАТНО? Все переходы будут уничтожены.')) return;
    try {
      await api.delete(`/api/shlink/short-urls/${shortCode}/permanent`);
      navigate(-1);
    } catch { /* shown */ }
  };

  if (loading) return (
    <Stack gap="lg">
      <Skeleton height={32} width={300} />
      <Skeleton height={240} radius="md" />
      <Grid>
        {[1, 2, 3].map(i => (
          <Grid.Col key={i} span={{ base: 12, sm: 4 }}>
            <Skeleton height={200} radius="md" />
          </Grid.Col>
        ))}
      </Grid>
    </Stack>
  );

  if (err) return (
    <Center py="xl"><Text c="red">{err}</Text></Center>
  );

  if (!data) return null;

  const isInactive = data.isActive === false;

  const totalPages = Math.ceil((data.visits?.length ?? 0) / PAGE_SIZE);
  const visitsPage = (data.visits ?? []).slice((page - 1) * PAGE_SIZE, page * PAGE_SIZE);

  const devicesArr = data.devices
    ? [
        { name: 'Desktop', value: data.devices.desktop },
        { name: 'Mobile',  value: data.devices.mobile  },
        { name: 'Tablet',  value: data.devices.tablet  },
      ]
    : [];

  const toValueArr = (arr: { name: string; count: number }[]) =>
    arr.map(x => ({ name: x.name, value: x.count }));

  const renderDonut = (items: { name: string; value: number }[], title: string) => (
    <Card withBorder radius="md" p="md">
      <Title order={5} mb="sm">{title}</Title>
      <ResponsiveContainer width="100%" height={180}>
        <PieChart>
          <Pie data={items} dataKey="value" nameKey="name" cx="50%" cy="50%" innerRadius={40} outerRadius={70} paddingAngle={2}>
            {items.map((_, i) => <Cell key={i} fill={COLORS[i % COLORS.length]} />)}
          </Pie>
          <RTooltip formatter={(v: number) => v.toLocaleString('ru-RU')} />
          <Legend />
        </PieChart>
      </ResponsiveContainer>
    </Card>
  );

  return (
    <ErrorBoundary>
      <Stack gap="lg">
        {/* Header */}
        <Group gap="sm">
          <ActionIcon variant="subtle" onClick={() => navigate(-1)}>
            <IconArrowLeft size={18} />
          </ActionIcon>
          <Stack gap={2} style={{ flex: 1 }}>
            <Group gap="sm">
              <Title order={3}>{data.title || 'Без названия'}</Title>
              {isInactive && (
                <Badge color="orange" variant="light" leftSection={<IconBan size={12} />}>
                  Деактивирована
                </Badge>
              )}
            </Group>
            <Group gap="xs">
              <Anchor href={data.shortUrl} target="_blank" fz="sm" fw={500}>
                {data.shortUrl}
              </Anchor>
              <CopyButton value={data.shortUrl}>
                {({ copied, copy }) => (
                  <Tooltip label={copied ? 'Скопировано' : 'Копировать'} withArrow>
                    <ActionIcon size="xs" variant="subtle" onClick={copy} color={copied ? 'teal' : 'gray'}>
                      {copied ? <IconCheck size={12} /> : <IconCopy size={12} />}
                    </ActionIcon>
                  </Tooltip>
                )}
              </CopyButton>
            </Group>
          </Stack>
        </Group>

        {/* Deactivated banner */}
        {isInactive && (
          <Alert
            icon={<IconAlertTriangle size={16} />}
            color="orange"
            title="Ссылка деактивирована"
          >
            {data.deactivatedAt && (
              <Text size="sm">
                Деактивирована {formatDateTime(data.deactivatedAt)}
                {data.deactivatedBy ? ` пользователем ${data.deactivatedBy}` : ''}.
              </Text>
            )}
            <Text size="sm">Переходы по ней не работают.</Text>
          </Alert>
        )}

        {/* Info card */}
        <Card withBorder radius="md" p="md">
          <Grid>
            <Grid.Col span={{ base: 12, sm: 6 }}>
              <Text fz="sm" c="dimmed">Куда ведёт</Text>
              <Anchor href={data.longUrl} target="_blank" fz="sm" lineClamp={2}>
                {data.longUrl}
              </Anchor>
            </Grid.Col>
            <Grid.Col span={{ base: 6, sm: 3 }}>
              <Text fz="sm" c="dimmed">Создана</Text>
              <Text fz="sm">{formatDate(data.dateCreated)}</Text>
            </Grid.Col>
            <Grid.Col span={{ base: 6, sm: 3 }}>
              <Text fz="sm" c="dimmed">Переходов всего</Text>
              <Text fz="sm" fw={600}>{(data.visitsTotal ?? 0).toLocaleString('ru-RU')}</Text>
            </Grid.Col>
          </Grid>
        </Card>

        {/* Analytics toolbar */}
        <Group justify="space-between">
          <Title order={5}>Аналитика</Title>
          <Group gap="sm">
            <SegmentedControl
              value={period}
              onChange={v => { setPeriod(v); setPage(1); }}
              data={PERIOD_OPTIONS}
              size="xs"
            />
            {!isInactive && user?.permissions.canEditOwnLinks && (
              <ActionIcon
                variant="light" color="blue"
                onClick={() => navigate(`/admin/urls/${shortCode}/edit`)}
              >
                <IconEdit size={16} />
              </ActionIcon>
            )}
            {canDeactivate && !isInactive && (
              <Tooltip label="Деактивировать ссылку">
                <ActionIcon variant="light" color="orange" onClick={handleDeactivate}>
                  <IconBan size={16} />
                </ActionIcon>
              </Tooltip>
            )}
            {canReactivate && isInactive && (
              <Tooltip label="Активировать ссылку">
                <ActionIcon variant="light" color="green" onClick={handleActivate}>
                  <IconPlayerPlay size={16} />
                </ActionIcon>
              </Tooltip>
            )}
            {canPermDelete && (
              <Tooltip label="Удалить безвозвратно">
                <ActionIcon variant="light" color="red" onClick={handlePermanentDelete}>
                  <IconTrash size={16} />
                </ActionIcon>
              </Tooltip>
            )}
          </Group>
        </Group>

        {/* Line chart */}
        <Card withBorder radius="md" p="md">
          <Title order={5} mb="sm">Переходы по дням</Title>
          <ResponsiveContainer width="100%" height={220}>
            <LineChart data={data.clicksPerDay ?? []}>
              <CartesianGrid strokeDasharray="3 3" />
              <XAxis dataKey="date" tickFormatter={formatDate} tick={{ fontSize: 11 }} />
              <YAxis tick={{ fontSize: 11 }} />
              <RTooltip labelFormatter={v => formatDate(String(v))} />
              <Line type="monotone" dataKey="clicks" stroke="#4dabf7" dot={false} strokeWidth={2} />
            </LineChart>
          </ResponsiveContainer>
        </Card>

        {/* Donuts */}
        <Grid>
          <Grid.Col span={{ base: 12, sm: 4 }}>{renderDonut(devicesArr, 'Устройства')}</Grid.Col>
          <Grid.Col span={{ base: 12, sm: 4 }}>{renderDonut(toValueArr(data.browsers ?? []), 'Браузеры')}</Grid.Col>
          <Grid.Col span={{ base: 12, sm: 4 }}>{renderDonut(toValueArr(data.os ?? []), 'ОС')}</Grid.Col>
        </Grid>

        {/* Visits table */}
        <Card withBorder radius="md" p="md">
          <Title order={5} mb="sm">Журнал переходов</Title>
          <Table fz="sm" verticalSpacing="xs" style={{ tableLayout: 'fixed' }}>
            <Table.Thead>
              <Table.Tr>
                <Table.Th>Дата / время</Table.Th>
                <Table.Th>Устройство</Table.Th>
                <Table.Th>ОС</Table.Th>
                <Table.Th>Реферер</Table.Th>
              </Table.Tr>
            </Table.Thead>
            <Table.Tbody>
              {visitsPage.map((v, i) => (
                <Table.Tr key={i}>
                  <Table.Td>{formatDateTime(v.date)}</Table.Td>
                  <Table.Td>{v.device ?? '—'}</Table.Td>
                  <Table.Td>{v.os ?? '—'}</Table.Td>
                  <Table.Td style={{ maxWidth: 200, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                    {v.referer ?? '—'}
                  </Table.Td>
                </Table.Tr>
              ))}
            </Table.Tbody>
          </Table>
          {totalPages > 1 && (
            <Group justify="center" mt="sm">
              <Pagination value={page} onChange={setPage} total={totalPages} size="sm" />
            </Group>
          )}
        </Card>
      </Stack>
    </ErrorBoundary>
  );
}
