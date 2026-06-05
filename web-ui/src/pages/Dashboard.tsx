import { useEffect, useState, useMemo } from 'react';
import {
  Grid, Card, Text, Title, Stack, Skeleton,
  Tabs, Group, SegmentedControl, Table, Badge,
  Center, Box, Tooltip as MTooltip,
} from '@mantine/core';
import {
  IconClick, IconLink, IconUsers, IconChartBar,
  IconDevices, IconAlertTriangle,
} from '@tabler/icons-react';
import {
  BarChart, Bar, XAxis, YAxis, CartesianGrid,
  Tooltip, ResponsiveContainer, PieChart, Pie, Cell, Legend,
} from 'recharts';
import { useNavigate } from 'react-router-dom';
import { api } from '../api/client';
import { useIsAdmin } from '../contexts/AuthContext';
import { ErrorBoundary } from '../components/ui/ErrorBoundary';
import { formatDate } from '../utils/date';
import type {
  OverviewResponse, UserActivityResponse,
  UrlStatsResponse, DevicesResponse, HeatmapCell,
} from '../types/api';

const COLORS = ['#4dabf7', '#51cf66', '#ff6b6b', '#ffd43b', '#cc5de8', '#74c0fc'];

const PERIOD_OPTIONS = [
  { label: '7 д', value: '7'  },
  { label: '30 д', value: '30' },
  { label: '90 д', value: '90' },
];

const WEEKDAYS = ['Пн', 'Вт', 'Ср', 'Чт', 'Пт', 'Сб', 'Вс'];

// ─── Activity Heatmap ───────────────────────────────────────────────────────

function ActivityHeatmap({ cells }: { cells: HeatmapCell[] }) {
  const maxVal = Math.max(...cells.map(c => c.value), 1);

  const grid: Record<string, number> = {};
  for (const c of cells) {
    grid[`${c.weekday}-${c.hour}`] = c.value;
  }

  return (
    <Box>
      <Title order={5} mb="sm">Активность по часам недели</Title>
      <Box style={{ overflowX: 'auto' }}>
        <Box style={{ display: 'grid', gridTemplateColumns: '32px repeat(24, 24px)', gap: 2, minWidth: 640 }}>
          <div />
          {Array.from({ length: 24 }, (_, h) => (
            <Text key={h} fz={10} ta="center" c="dimmed">{h}</Text>
          ))}
          {WEEKDAYS.map((day, wd) => (
            <>
              <Text key={`label-${wd}`} fz={11} style={{ lineHeight: '24px' }} c="dimmed">{day}</Text>
              {Array.from({ length: 24 }, (_, h) => {
                const val = grid[`${wd}-${h}`] ?? 0;
                const alpha = val === 0 ? 0.05 : 0.15 + (val / maxVal) * 0.8;
                return (
                  <MTooltip
                    key={`${wd}-${h}`}
                    label={`${day} ${String(h).padStart(2, '0')}:00 — ${val.toLocaleString('ru-RU')} переходов`}
                    withArrow
                  >
                    <Box
                      style={{
                        width: 24,
                        height: 24,
                        borderRadius: 3,
                        background: `oklch(0.55 0.15 192 / ${alpha})`,
                        cursor: 'default',
                      }}
                    />
                  </MTooltip>
                );
              })}
            </>
          ))}
        </Box>
      </Box>
    </Box>
  );
}

// ─── Overview Tab ────────────────────────────────────────────────────────────

function OverviewTab({ period }: { period: string }) {
  const [data, setData] = useState<OverviewResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [err, setErr] = useState<string | null>(null);

  useEffect(() => {
    setLoading(true);
    setErr(null);
    api.get<OverviewResponse>('/api/dashboard/overview', { params: { period } })
      .then(setData)
      .catch((e: Error) => setErr(e.message ?? 'Ошибка загрузки'))
      .finally(() => setLoading(false));
  }, [period]);

  if (loading) return (
    <Stack gap="lg">
      <Skeleton height={32} width={200} />
      <Grid>
        {[1, 2, 3, 4].map(i => (
          <Grid.Col key={i} span={{ base: 12, sm: 6, md: 3 }}>
            <Skeleton height={100} radius="md" />
          </Grid.Col>
        ))}
      </Grid>
      <Skeleton height={300} radius="md" />
    </Stack>
  );

  if (err) return (
    <Center py="xl">
      <Stack align="center" gap="xs">
        <IconAlertTriangle size={32} color="var(--mantine-color-red-6)" />
        <Text c="red">{err}</Text>
      </Stack>
    </Center>
  );

  if (!data) return null;

  const kpis = [
    { label: 'Переходов за период',   value: data.totalClicks,    icon: <IconClick size={24} /> },
    { label: 'Активных ссылок',        value: data.activeLinks,    icon: <IconLink size={24} /> },
    { label: 'Ссылок создано',         value: data.createdPeriod,  icon: <IconChartBar size={24} /> },
    { label: 'Уникальных посетителей', value: data.uniqueVisitors ?? '—', icon: <IconUsers size={24} /> },
  ];

  return (
    <Stack gap="lg">
      <Grid>
        {kpis.map(k => (
          <Grid.Col key={k.label} span={{ base: 12, sm: 6, md: 3 }}>
            <Card withBorder radius="md" p="md">
              <Group gap="sm" mb={4}>
                <Box c="blue">{k.icon}</Box>
                <Text fz="sm" c="dimmed">{k.label}</Text>
              </Group>
              <Text fz="xl" fw={700}>
                {typeof k.value === 'number' ? k.value.toLocaleString('ru-RU') : k.value}
              </Text>
            </Card>
          </Grid.Col>
        ))}
      </Grid>

      <Card withBorder radius="md" p="md">
        <Title order={5} mb="sm">Активность по дням</Title>
        <ResponsiveContainer width="100%" height={240}>
          <BarChart data={data.clicksPerDay ?? []}>
            <CartesianGrid strokeDasharray="3 3" />
            <XAxis dataKey="date" tickFormatter={formatDate} tick={{ fontSize: 11 }} />
            <YAxis tick={{ fontSize: 11 }} />
            <Tooltip labelFormatter={v => formatDate(String(v))} />
            <Bar dataKey="clicks" fill="#4dabf7" radius={[3, 3, 0, 0]} />
          </BarChart>
        </ResponsiveContainer>
      </Card>

      <Card withBorder radius="md" p="md">
        <Title order={5} mb="sm">Топ-5 ссылок по переходам</Title>
        <ResponsiveContainer width="100%" height={180}>
          <BarChart layout="vertical" data={(data.topLinks ?? []).slice(0, 5)}>
            <CartesianGrid strokeDasharray="3 3" />
            <XAxis type="number" tick={{ fontSize: 11 }} />
            <YAxis type="category" dataKey="title" width={160} tick={{ fontSize: 11 }} />
            <Tooltip />
            <Bar dataKey="visits" fill="#51cf66" radius={[0, 3, 3, 0]} />
          </BarChart>
        </ResponsiveContainer>
      </Card>
    </Stack>
  );
}

// ─── Users Activity Tab ───────────────────────────────────────────────────────

function UsersTab({ period }: { period: string }) {
  const [data, setData] = useState<UserActivityResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [err, setErr] = useState<string | null>(null);
  const navigate = useNavigate();

  useEffect(() => {
    setLoading(true);
    setErr(null);
    api.get<UserActivityResponse>('/api/dashboard/users', { params: { period } })
      .then(setData)
      .catch((e: Error) => setErr(e.message ?? 'Ошибка загрузки'))
      .finally(() => setLoading(false));
  }, [period]);

  if (loading) return <Skeleton height={300} radius="md" />;
  if (err) return <Text c="red">{err}</Text>;
  if (!data) return null;

  const rows = (data.users ?? []).map(u => (
    <Table.Tr key={u.sub} style={{ cursor: 'pointer' }} onClick={() => navigate(`/admin/users/${u.sub}`)}>
      <Table.Td>{u.username}</Table.Td>
      <Table.Td>{u.linksCount.toLocaleString('ru-RU')}</Table.Td>
      <Table.Td>{u.visitsCount.toLocaleString('ru-RU')}</Table.Td>
      <Table.Td>{u.lastActivityAt ? formatDate(u.lastActivityAt) : '—'}</Table.Td>
    </Table.Tr>
  ));

  return (
    <Stack gap="lg">
      <Card withBorder radius="md" p="md">
        <Title order={5} mb="sm">Активность пользователей</Title>
        <Table fz="sm" verticalSpacing="xs">
          <Table.Thead>
            <Table.Tr>
              <Table.Th>Пользователь</Table.Th>
              <Table.Th>Ссылок</Table.Th>
              <Table.Th>Переходов</Table.Th>
              <Table.Th>Последняя активность</Table.Th>
            </Table.Tr>
          </Table.Thead>
          <Table.Tbody>{rows}</Table.Tbody>
        </Table>
      </Card>
    </Stack>
  );
}

// ─── URLs Activity Tab ────────────────────────────────────────────────────────

type SortKey = 'title' | 'visitsToday' | 'visits7d' | 'visitsTotal';

function UrlsTab({ period, isAdmin }: { period: string; isAdmin: boolean }) {
  const [data, setData] = useState<UrlStatsResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [err, setErr] = useState<string | null>(null);
  const [sortKey, setSortKey] = useState<SortKey>('visitsTotal');
  const [sortDir, setSortDir] = useState<'asc' | 'desc'>('desc');
  const navigate = useNavigate();

  useEffect(() => {
    setLoading(true);
    setErr(null);
    api.get<UrlStatsResponse>('/api/dashboard/urls', { params: { period } })
      .then(setData)
      .catch((e: Error) => setErr(e.message ?? 'Ошибка загрузки'))
      .finally(() => setLoading(false));
  }, [period]);

  const sorted = useMemo(() => {
    if (!data?.urls) return [];
    return [...data.urls].sort((a, b) => {
      const av = a[sortKey], bv = b[sortKey];
      if (typeof av === 'string' && typeof bv === 'string') {
        return sortDir === 'asc' ? av.localeCompare(bv) : bv.localeCompare(av);
      }
      return sortDir === 'asc' ? Number(av) - Number(bv) : Number(bv) - Number(av);
    });
  }, [data, sortKey, sortDir]);

  function toggleSort(key: SortKey) {
    if (sortKey === key) setSortDir(d => d === 'asc' ? 'desc' : 'asc');
    else { setSortKey(key); setSortDir('desc'); }
  }

  const arrow = (key: SortKey) => sortKey === key ? (sortDir === 'asc' ? ' ↑' : ' ↓') : '';

  if (loading) return <Skeleton height={300} radius="md" />;
  if (err) return <Text c="red">{err}</Text>;
  if (!data) return null;

  const rows = sorted.map(u => (
    <Table.Tr key={u.shortCode} style={{ cursor: 'pointer' }} onClick={() => navigate(`/admin/urls/${u.shortCode}`)}>
      <Table.Td>{u.title || <Text fz="sm" c="dimmed">Без названия</Text>}</Table.Td>
      <Table.Td ta="right">{u.visitsToday.toLocaleString('ru-RU')}</Table.Td>
      <Table.Td ta="right">{u.visits7d.toLocaleString('ru-RU')}</Table.Td>
      <Table.Td ta="right">{u.visitsTotal.toLocaleString('ru-RU')}</Table.Td>
      <Table.Td>
        <Badge size="xs" color={u.status === 'active' ? 'green' : 'gray'}>
          {u.status === 'active' ? 'Активна' : 'Неактивна'}
        </Badge>
      </Table.Td>
    </Table.Tr>
  ));

  return (
    <Stack gap="lg">
      {isAdmin && (
        <Text fz="xs" c="dimmed">Все ссылки системы. Кликните для подробностей.</Text>
      )}
      <Card withBorder radius="md" p="md">
        <Table fz="sm" verticalSpacing="xs" style={{ tableLayout: 'fixed' }}>
          <Table.Thead>
            <Table.Tr>
              <Table.Th style={{ cursor: 'pointer' }} onClick={() => toggleSort('title')}>Название{arrow('title')}</Table.Th>
              <Table.Th ta="right" style={{ cursor: 'pointer' }} onClick={() => toggleSort('visitsToday')}>Сегодня{arrow('visitsToday')}</Table.Th>
              <Table.Th ta="right" style={{ cursor: 'pointer' }} onClick={() => toggleSort('visits7d')}>7 дней{arrow('visits7d')}</Table.Th>
              <Table.Th ta="right" style={{ cursor: 'pointer' }} onClick={() => toggleSort('visitsTotal')}>Всего{arrow('visitsTotal')}</Table.Th>
              <Table.Th>Статус</Table.Th>
            </Table.Tr>
          </Table.Thead>
          <Table.Tbody>{rows}</Table.Tbody>
        </Table>
      </Card>
    </Stack>
  );
}

// ─── Devices & Time Tab ───────────────────────────────────────────────────────

function DevicesTab({ period }: { period: string }) {
  const [data, setData] = useState<DevicesResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [err, setErr] = useState<string | null>(null);

  useEffect(() => {
    setLoading(true);
    setErr(null);
    api.get<DevicesResponse>('/api/dashboard/devices', { params: { period } })
      .then(setData)
      .catch((e: Error) => setErr(e.message ?? 'Ошибка загрузки'))
      .finally(() => setLoading(false));
  }, [period]);

  if (loading) return <Skeleton height={300} radius="md" />;
  if (err) return <Text c="red">{err}</Text>;
  if (!data) return null;

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
      <ResponsiveContainer width="100%" height={200}>
        <PieChart>
          <Pie data={items} dataKey="value" nameKey="name" cx="50%" cy="50%" innerRadius={50} outerRadius={80} paddingAngle={2}>
            {items.map((_, i) => <Cell key={i} fill={COLORS[i % COLORS.length]} />)}
          </Pie>
          <Tooltip formatter={(v: number) => v.toLocaleString('ru-RU')} />
          <Legend />
        </PieChart>
      </ResponsiveContainer>
    </Card>
  );

  return (
    <Stack gap="lg">
      <Grid>
        <Grid.Col span={{ base: 12, sm: 4 }}>{renderDonut(devicesArr, 'Устройства')}</Grid.Col>
        <Grid.Col span={{ base: 12, sm: 4 }}>{renderDonut(toValueArr(data.browsers ?? []), 'Браузеры')}</Grid.Col>
        <Grid.Col span={{ base: 12, sm: 4 }}>{renderDonut(toValueArr(data.os ?? []), 'Операционные системы')}</Grid.Col>
      </Grid>

      {data.heatmap && data.heatmap.length > 0 && (
        <Card withBorder radius="md" p="md">
          <ActivityHeatmap cells={data.heatmap} />
        </Card>
      )}
    </Stack>
  );
}

// ─── Dashboard Page ───────────────────────────────────────────────────────────

export function Dashboard() {
  const isAdmin = useIsAdmin();
  const [period, setPeriod] = useState('30');

  return (
    <Stack gap="lg">
      <Group justify="space-between" align="center">
        <Title order={2}>Дашборд</Title>
        <SegmentedControl
          value={period}
          onChange={setPeriod}
          data={PERIOD_OPTIONS}
          size="xs"
        />
      </Group>

      <Tabs defaultValue="overview">
        <Tabs.List mb="md">
          <Tabs.Tab value="overview" leftSection={<IconChartBar size={16} />}>Обзор</Tabs.Tab>
          {isAdmin && (
            <Tabs.Tab value="users" leftSection={<IconUsers size={16} />}>Активность пользователей</Tabs.Tab>
          )}
          <Tabs.Tab value="links" leftSection={<IconLink size={16} />}>По ссылкам</Tabs.Tab>
          <Tabs.Tab value="devices" leftSection={<IconDevices size={16} />}>Устройства и время</Tabs.Tab>
        </Tabs.List>

        <Tabs.Panel value="overview">
          <ErrorBoundary>
            <OverviewTab period={period} />
          </ErrorBoundary>
        </Tabs.Panel>

        {isAdmin && (
          <Tabs.Panel value="users">
            <ErrorBoundary>
              <UsersTab period={period} />
            </ErrorBoundary>
          </Tabs.Panel>
        )}

        <Tabs.Panel value="links">
          <ErrorBoundary>
            <UrlsTab period={period} isAdmin={isAdmin} />
          </ErrorBoundary>
        </Tabs.Panel>

        <Tabs.Panel value="devices">
          <ErrorBoundary>
            <DevicesTab period={period} />
          </ErrorBoundary>
        </Tabs.Panel>
      </Tabs>
    </Stack>
  );
}
