import { useEffect, useState, useMemo } from 'react';
import {
  Grid, Card, Text, Title, Stack, Skeleton,
  Tabs, Group, SegmentedControl, Table, Badge,
  Center, Anchor, Box, Tooltip as MTooltip,
} from '@mantine/core';
import {
  IconClick, IconLink, IconUsers, IconChartBar,
  IconDevices, IconAlertTriangle,
} from '@tabler/icons-react';
import {
  LineChart, Line, BarChart, Bar, XAxis, YAxis, CartesianGrid,
  Tooltip, ResponsiveContainer, PieChart, Pie, Cell, Legend,
} from 'recharts';
import { useNavigate } from 'react-router-dom';
import { api } from '../api/client';
import { useAuth, useIsAdmin } from '../contexts/AuthContext';
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

export function Dashboard() {
  const isAdmin  = useIsAdmin();
  const [period, setPeriod] = useState('30');

  return (
    <Stack gap="lg">
      <Group justify="space-between">
        <Title order={2}>Дашборд</Title>
        <SegmentedControl
          data={PERIOD_OPTIONS}
          value={period}
          onChange={setPeriod}
          size="xs"
        />
      </Group>

      <Tabs defaultValue="overview">
        <Tabs.List>
          <Tabs.Tab value="overview"     leftSection={<IconChartBar size={16} />}>Обзор</Tabs.Tab>
          {isAdmin && (
            <Tabs.Tab value="users"      leftSection={<IconUsers size={16} />}>Пользователи</Tabs.Tab>
          )}
          <Tabs.Tab value="links"        leftSection={<IconLink size={16} />}>Ссылки</Tabs.Tab>
          <Tabs.Tab value="devices"      leftSection={<IconDevices size={16} />}>Устройства</Tabs.Tab>
        </Tabs.List>

        <Tabs.Panel value="overview" pt="md">
          <ErrorBoundary section="Обзор">
            <OverviewTab period={period} isAdmin={isAdmin} />
          </ErrorBoundary>
        </Tabs.Panel>

        {isAdmin && (
          <Tabs.Panel value="users" pt="md">
            <ErrorBoundary section="Активность пользователей">
              <UsersTab period={period} />
            </ErrorBoundary>
          </Tabs.Panel>
        )}

        <Tabs.Panel value="links" pt="md">
          <ErrorBoundary section="Активность по ссылкам">
            <LinksTab period={period} isAdmin={isAdmin} />
          </ErrorBoundary>
        </Tabs.Panel>

        <Tabs.Panel value="devices" pt="md">
          <ErrorBoundary section="Устройства и время">
            <DevicesTab period={period} />
          </ErrorBoundary>
        </Tabs.Panel>
      </Tabs>
    </Stack>
  );
}

// ───────────────────────────────────────────────────────────────────────
// Вкладка 1: Обзор
// ───────────────────────────────────────────────────────────────────────
function OverviewTab({ period, isAdmin }: { period: string; isAdmin: boolean }) {
  const [data,    setData]    = useState<OverviewResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [err,     setErr]     = useState<string | null>(null);

  useEffect(() => {
    setLoading(true); setErr(null);
    api.get<OverviewResponse>('/api/dashboard/overview', { params: { period } })
      .then(setData)
      .catch((e: unknown) => setErr(e instanceof Error ? e.message : 'Ошибка'))
      .finally(() => setLoading(false));
  }, [period]);

  if (loading) return <DashSkeleton rows={2} />;
  if (err)     return <ErrText msg={err} />;
  if (!data)   return null;

  return (
    <Stack gap="lg">
      {/* KPI-карточки */}
      <Grid>
        <Grid.Col span={{ base: 12, sm: 6, md: 3 }}>
          <KPICard label="Переходов за период"
            value={data.totalClicks.toLocaleString('ru-RU')} icon={<IconClick size={22} />} color="blue" />
        </Grid.Col>
        <Grid.Col span={{ base: 12, sm: 6, md: 3 }}>
          <KPICard label="Активных ссылок"
            value={data.activeLinks.toLocaleString('ru-RU')} icon={<IconLink size={22} />} color="green" />
        </Grid.Col>
        <Grid.Col span={{ base: 12, sm: 6, md: 3 }}>
          <KPICard label="Создано за период"
            value={data.createdPeriod.toLocaleString('ru-RU')} icon={<IconChartBar size={22} />} color="teal" />
        </Grid.Col>
        {data.uniqueVisitors != null && (
          <Grid.Col span={{ base: 12, sm: 6, md: 3 }}>
            <KPICard label="Уникальных посетителей"
              value={data.uniqueVisitors.toLocaleString('ru-RU')} icon={<IconUsers size={22} />} color="violet" />
          </Grid.Col>
        )}
      </Grid>

      {/* Активность по дням */}
      <Card withBorder radius="md" p="lg">
        <Text fw={600} mb="md">Активность по дням</Text>
        <ResponsiveContainer width="100%" height={240}>
          <BarChart data={data.clicksPerDay}>
            <CartesianGrid strokeDasharray="3 3" stroke="#373A40" />
            <XAxis dataKey="date" tick={{ fontSize: 11 }} />
            <YAxis tick={{ fontSize: 11 }} />
            <Tooltip />
            <Bar dataKey="clicks" fill="#4dabf7" radius={[3, 3, 0, 0]} />
          </BarChart>
        </ResponsiveContainer>
      </Card>

      {/* Топ-5 ссылок */}
      {data.topLinks.length > 0 && (
        <Card withBorder radius="md" p="lg">
          <Text fw={600} mb="md">Топ-{Math.min(data.topLinks.length, 5)} ссылок</Text>
          <ResponsiveContainer width="100%" height={200}>
            <BarChart
              layout="vertical"
              data={data.topLinks.slice(0, 5)}
              margin={{ left: 20 }}
            >
              <CartesianGrid strokeDasharray="3 3" stroke="#373A40" />
              <XAxis type="number" tick={{ fontSize: 11 }} />
              <YAxis
                type="category"
                dataKey="title"
                tick={{ fontSize: 11 }}
                width={120}
                tickFormatter={(v: string) => v.length > 18 ? v.slice(0, 18) + '…' : v}
              />
              <Tooltip />
              <Bar dataKey="visits" fill="#51cf66" radius={[0, 3, 3, 0]} />
            </BarChart>
          </ResponsiveContainer>
        </Card>
      )}
    </Stack>
  );
}

// ───────────────────────────────────────────────────────────────────────
// Вкладка 2: Активность пользователей (admin only)
// ───────────────────────────────────────────────────────────────────────
function UsersTab({ period }: { period: string }) {
  const navigate = useNavigate();
  const [data,    setData]    = useState<UserActivityResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [err,     setErr]     = useState<string | null>(null);

  useEffect(() => {
    setLoading(true); setErr(null);
    api.get<UserActivityResponse>('/api/dashboard/users', { params: { period } })
      .then(setData)
      .catch((e: unknown) => setErr(e instanceof Error ? e.message : 'Ошибка'))
      .finally(() => setLoading(false));
  }, [period]);

  if (loading) return <DashSkeleton rows={3} />;
  if (err)     return <ErrText msg={err} />;
  if (!data)   return null;

  const usernames = data.users.map(u => u.username);

  return (
    <Stack gap="lg">
      <Table striped highlightOnHover withTableBorder>
        <Table.Thead>
          <Table.Tr>
            <Table.Th>Пользователь</Table.Th>
            <Table.Th>Ссылок</Table.Th>
            <Table.Th>Переходов</Table.Th>
            <Table.Th>Последняя активность</Table.Th>
          </Table.Tr>
        </Table.Thead>
        <Table.Tbody>
          {data.users.map(u => (
            <Table.Tr
              key={u.sub}
              style={{ cursor: 'pointer' }}
              onClick={() => navigate(`/admin/users/${encodeURIComponent(u.sub)}`)}
            >
              <Table.Td><Text fw={500}>{u.username}</Text></Table.Td>
              <Table.Td>{u.linksCount.toLocaleString('ru-RU')}</Table.Td>
              <Table.Td>{u.visitsCount.toLocaleString('ru-RU')}</Table.Td>
              <Table.Td>
                <Text size="sm" c="dimmed">
                  {u.lastActivityAt ? formatDate(u.lastActivityAt) : '—'}
                </Text>
              </Table.Td>
            </Table.Tr>
          ))}
          {data.users.length === 0 && (
            <Table.Tr>
              <Table.Td colSpan={4}>
                <Center p="xl"><Text c="dimmed">Нет данных</Text></Center>
              </Table.Td>
            </Table.Tr>
          )}
        </Table.Tbody>
      </Table>

      {data.newLinksPerDay.length > 0 && (
        <Card withBorder radius="md" p="lg">
          <Text fw={600} mb="md">Новые ссылки по пользователям</Text>
          <ResponsiveContainer width="100%" height={240}>
            <BarChart data={data.newLinksPerDay}>
              <CartesianGrid strokeDasharray="3 3" stroke="#373A40" />
              <XAxis dataKey="date" tick={{ fontSize: 11 }} />
              <YAxis tick={{ fontSize: 11 }} />
              <Tooltip />
              <Legend />
              {usernames.map((name, i) => (
                <Bar key={name} dataKey={name} stackId="a"
                  fill={COLORS[i % COLORS.length]} />
              ))}
            </BarChart>
          </ResponsiveContainer>
        </Card>
      )}
    </Stack>
  );
}

// ───────────────────────────────────────────────────────────────────────
// Вкладка 3: Активность по ссылкам
// ───────────────────────────────────────────────────────────────────────
type SortKey = 'visitsToday' | 'visits7d' | 'visitsTotal';

function LinksTab({ period, isAdmin }: { period: string; isAdmin: boolean }) {
  const navigate = useNavigate();
  const [data,    setData]    = useState<UrlStatsResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [err,     setErr]     = useState<string | null>(null);
  const [sortBy,  setSortBy]  = useState<SortKey>('visitsTotal');
  const [tagFilter, setTagFilter] = useState('');
  const [userFilter, setUserFilter] = useState('');

  useEffect(() => {
    setLoading(true); setErr(null);
    api.get<UrlStatsResponse>('/api/dashboard/links', { params: { period } })
      .then(setData)
      .catch((e: unknown) => setErr(e instanceof Error ? e.message : 'Ошибка'))
      .finally(() => setLoading(false));
  }, [period]);

  const sorted = useMemo(() => {
    if (!data) return [];
    let rows = [...data.urls];
    if (tagFilter)  rows = rows.filter(r => r.tags.includes(tagFilter));
    if (userFilter) rows = rows.filter(r => r.ownerUsername === userFilter);
    return rows.sort((a, b) => b[sortBy] - a[sortBy]);
  }, [data, sortBy, tagFilter, userFilter]);

  if (loading) return <DashSkeleton rows={2} />;
  if (err)     return <ErrText msg={err} />;
  if (!data)   return null;

  const allTags  = Array.from(new Set(data.urls.flatMap(u => u.tags)));
  const allUsers = isAdmin ? Array.from(new Set(data.urls.map(u => u.ownerUsername).filter(Boolean))) as string[] : [];

  const SortTh = ({ col, label }: { col: SortKey; label: string }) => (
    <Table.Th
      style={{ cursor: 'pointer', userSelect: 'none' }}
      onClick={() => setSortBy(col)}
    >
      {label} {sortBy === col ? '↓' : ''}
    </Table.Th>
  );

  return (
    <Stack gap="md">
      <Group>
        <select
          value={tagFilter}
          onChange={e => setTagFilter(e.target.value)}
          style={{ padding: '4px 8px', borderRadius: 4, fontSize: 13 }}
        >
          <option value="">Все теги</option>
          {allTags.map(t => <option key={t} value={t}>{t}</option>)}
        </select>
        {isAdmin && (
          <select
            value={userFilter}
            onChange={e => setUserFilter(e.target.value)}
            style={{ padding: '4px 8px', borderRadius: 4, fontSize: 13 }}
          >
            <option value="">Все пользователи</option>
            {allUsers.map(u => <option key={u} value={u}>{u}</option>)}
          </select>
        )}
      </Group>

      <Table striped highlightOnHover withTableBorder fz="sm" verticalSpacing="xs">
        <Table.Thead>
          <Table.Tr>
            <Table.Th>Название</Table.Th>
            <SortTh col="visitsToday" label="Сегодня" />
            <SortTh col="visits7d"    label="7 дней" />
            <SortTh col="visitsTotal" label="Всего" />
            <Table.Th>Статус</Table.Th>
          </Table.Tr>
        </Table.Thead>
        <Table.Tbody>
          {sorted.map(r => (
            <Table.Tr
              key={r.shortCode}
              style={{ cursor: 'pointer' }}
              onClick={() => navigate(
                isAdmin ? `/admin/urls/${r.shortCode}` : `/links/${r.shortCode}`
              )}
            >
              <Table.Td>
                <Text size="sm" fw={500} truncate maw={260}>
                  {r.title || r.shortCode}
                </Text>
                {r.ownerUsername && isAdmin && (
                  <Text size="xs" c="dimmed">{r.ownerUsername}</Text>
                )}
              </Table.Td>
              <Table.Td>{r.visitsToday.toLocaleString('ru-RU')}</Table.Td>
              <Table.Td>{r.visits7d.toLocaleString('ru-RU')}</Table.Td>
              <Table.Td>{r.visitsTotal.toLocaleString('ru-RU')}</Table.Td>
              <Table.Td>
                <Badge
                  size="xs"
                  color={r.status === 'active' ? 'green' : 'gray'}
                  variant="light"
                >
                  {r.status === 'active' ? 'активна' : 'неактивна'}
                </Badge>
              </Table.Td>
            </Table.Tr>
          ))}
          {sorted.length === 0 && (
            <Table.Tr>
              <Table.Td colSpan={5}>
                <Center p="xl"><Text c="dimmed">Нет данных</Text></Center>
              </Table.Td>
            </Table.Tr>
          )}
        </Table.Tbody>
      </Table>
    </Stack>
  );
}

// ───────────────────────────────────────────────────────────────────────
// Вкладка 4: Устройства и время
// ───────────────────────────────────────────────────────────────────────
function DevicesTab({ period }: { period: string }) {
  const [data,    setData]    = useState<DevicesResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [err,     setErr]     = useState<string | null>(null);

  useEffect(() => {
    setLoading(true); setErr(null);
    api.get<DevicesResponse>('/api/dashboard/devices', { params: { period } })
      .then(setData)
      .catch((e: unknown) => setErr(e instanceof Error ? e.message : 'Ошибка'))
      .finally(() => setLoading(false));
  }, [period]);

  if (loading) return <DashSkeleton rows={2} />;
  if (err)     return <ErrText msg={err} />;
  if (!data)   return null;

  const deviceData = [
    { name: 'Desktop', value: data.devices.desktop },
    { name: 'Mobile',  value: data.devices.mobile  },
    { name: 'Tablet',  value: data.devices.tablet  },
  ].filter(d => d.value > 0);

  return (
    <Stack gap="lg">
      <Grid>
        <Grid.Col span={{ base: 12, md: 4 }}>
          <DonutCard title="Устройства" data={deviceData} />
        </Grid.Col>
        <Grid.Col span={{ base: 12, md: 4 }}>
          <DonutCard
            title="Браузеры"
            data={data.browsers.map(b => ({ name: b.name, value: b.count }))}
          />
        </Grid.Col>
        <Grid.Col span={{ base: 12, md: 4 }}>
          <DonutCard
            title="Операционные системы"
            data={data.os.map(o => ({ name: o.name, value: o.count }))}
          />
        </Grid.Col>
      </Grid>

      {data.heatmap && data.heatmap.length > 0 && (
        <ActivityHeatmap data={data.heatmap} />
      )}
    </Stack>
  );
}

// ────────── Утилиты ─────────────────────────────────────────────────────────────
function KPICard({ label, value, icon, color }: {
  label: string; value: string; icon: React.ReactNode; color: string;
}) {
  return (
    <Card withBorder radius="md" p="lg" h="100%">
      <Group justify="space-between">
        <Stack gap={2}>
          <Text size="sm" c="dimmed">{label}</Text>
          <Text size="xl" fw={700}>{value}</Text>
        </Stack>
        <Text c={color}>{icon}</Text>
      </Group>
    </Card>
  );
}

function DonutCard({ title, data }: { title: string; data: { name: string; value: number }[] }) {
  return (
    <Card withBorder radius="md" p="lg">
      <Text fw={600} mb="md">{title}</Text>
      <ResponsiveContainer width="100%" height={200}>
        <PieChart>
          <Pie data={data} dataKey="value" nameKey="name"
            cx="50%" cy="50%" innerRadius={50} outerRadius={80}
          >
            {data.map((_, i) => <Cell key={i} fill={COLORS[i % COLORS.length]} />)}
          </Pie>
          <Legend />
          <Tooltip formatter={(v: number) => v.toLocaleString('ru-RU')} />
        </PieChart>
      </ResponsiveContainer>
    </Card>
  );
}

// ───────────────────────────────────────────────────────────────────────
// Heatmap: активность по часам недели
// ───────────────────────────────────────────────────────────────────────
const WEEKDAYS = ['Вс', 'Пн', 'Вт', 'Ср', 'Чт', 'Пт', 'Сб'];

function ActivityHeatmap({ data }: { data: HeatmapCell[] }) {
  const max = Math.max(...data.map(d => d.value), 1);

  // grid[weekday][hour] = value
  const grid = Array.from({ length: 7 }, (_, wd) =>
    Array.from({ length: 24 }, (_, h) =>
      data.find(d => d.weekday === wd && d.hour === h)?.value ?? 0,
    ),
  );

  return (
    <Card withBorder radius="md" p="lg">
      <Text fw={600} mb="md">Активность по часам недели</Text>
      <Box style={{ overflowX: 'auto' }}>
        <table style={{ borderCollapse: 'separate', borderSpacing: 2 }}>
          <thead>
            <tr>
              <th style={{ width: 28 }} />
              {Array.from({ length: 24 }, (_, h) => (
                <th
                  key={h}
                  style={{
                    fontSize: 10,
                    fontWeight: 400,
                    textAlign: 'center',
                    width: 20,
                    color: 'var(--mantine-color-dimmed)',
                  }}
                >
                  {h % 3 === 0 ? h : ''}
                </th>
              ))}
            </tr>
          </thead>
          <tbody>
            {grid.map((row, wd) => (
              <tr key={wd}>
                <td
                  style={{
                    fontSize: 11,
                    paddingRight: 6,
                    color: 'var(--mantine-color-dimmed)',
                    userSelect: 'none',
                  }}
                >
                  {WEEKDAYS[wd]}
                </td>
                {row.map((val, h) => {
                  const alpha = val === 0 ? 0.07 : 0.15 + (val / max) * 0.8;
                  return (
                    <td key={h} style={{ padding: 0 }}>
                      <MTooltip
                        label={`${WEEKDAYS[wd]} ${String(h).padStart(2, '0')}:00 — ${val.toLocaleString('ru-RU')} переходов`}
                        withArrow
                        position="top"
                      >
                        <Box
                          style={{
                            width: 18,
                            height: 18,
                            borderRadius: 3,
                            background: `oklch(0.55 0.15 192 / ${alpha.toFixed(2)})`,
                          }}
                        />
                      </MTooltip>
                    </td>
                  );
                })}
              </tr>
            ))}
          </tbody>
        </table>
      </Box>
    </Card>
  );
}

function DashSkeleton({ rows }: { rows: number }) {
  return (
    <Stack gap="md">
      <Grid>
        {Array.from({ length: 4 }).map((_, i) => (
          <Grid.Col key={i} span={{ base: 12, sm: 6, md: 3 }}>
            <Skeleton height={88} radius="md" />
          </Grid.Col>
        ))}
      </Grid>
      {Array.from({ length: rows }).map((_, i) => (
        <Skeleton key={i} height={240} radius="md" />
      ))}
    </Stack>
  );
}

function ErrText({ msg }: { msg: string }) {
  return (
    <Center h={200}>
      <Stack align="center" gap="xs">
        <IconAlertTriangle size={28} color="var(--mantine-color-red-6)" />
        <Text c="red" size="sm">{msg}</Text>
      </Stack>
    </Center>
  );
}
