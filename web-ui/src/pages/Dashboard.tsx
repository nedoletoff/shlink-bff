import { useEffect, useState } from 'react';
import {
  Container, Title, Text, Group, Stack, Grid, Card,
  Badge, Table, Anchor, Select, Tabs, Loader, Center,
  Avatar, SimpleGrid, Progress, Tooltip, ThemeIcon,
  RingProgress, Skeleton, Box,
} from '@mantine/core';
import {
  IconLink, IconClick, IconUsers, IconTrendingUp,
  IconDeviceDesktop, IconDeviceMobile, IconDeviceTablet,
  IconArrowUpRight, IconArrowDownRight, IconMinus,
} from '@tabler/icons-react';
import {
  AreaChart, Area, XAxis, YAxis, CartesianGrid,
  Tooltip as RTooltip, ResponsiveContainer,
} from 'recharts';
import { useNavigate } from 'react-router-dom';
import { useIsAdmin } from '../contexts/AuthContext';
import { api } from '../api/client';
import { formatDate } from '../utils/date';

// ─── types ──────────────────────────────────────────────────────────────────
interface ClickPoint  { date: string; clicks: number; }
interface TopLink {
  shortCode: string; shortUrl: string; longUrl: string;
  title: string; visitsTotal: number;
}
interface AdminUser {
  sub: string; username: string; email: string;
  role: string; status: string;
}
interface HeatCell { weekday: number; hour: number; value: number; }
interface NamedCount { name: string; count: number; }

interface DashboardData {
  overview: {
    linksCount: number;
    visitsTotal: number;
    topLinks: TopLink[];
    recentLinks: TopLink[];
  };
  users: AdminUser[] | null;
  visits: {
    clicksPerDay: ClickPoint[];
    clicksTotal: number;
  };
  devices: {
    devices: { desktop: number; mobile: number; tablet: number };
    browsers: NamedCount[];
    os: NamedCount[];
    heatmap: HeatCell[];
  };
}

// ─── helpers ─────────────────────────────────────────────────────────────────
const DAYS = ['Вс','Пн','Вт','Ср','Чт','Пт','Сб'];

function trendIcon(val: number, prev: number) {
  if (!prev) return <IconMinus size={14} />;
  return val >= prev
    ? <IconArrowUpRight size={14} color="var(--mantine-color-green-5)" />
    : <IconArrowDownRight size={14} color="var(--mantine-color-red-5)" />;
}

// ─── KPI Card ────────────────────────────────────────────────────────────────
function StatCard({
  icon, label, value, sub, color = 'blue', loading = false,
}: {
  icon: React.ReactNode; label: string; value: string | number;
  sub?: React.ReactNode; color?: string; loading?: boolean;
}) {
  return (
    <Card withBorder p="md" radius="md" style={{ position: 'relative', overflow: 'hidden' }}>
      <Box
        style={{
          position: 'absolute', top: -20, right: -20,
          width: 80, height: 80, borderRadius: '50%',
          background: `var(--mantine-color-${color}-1)`,
          opacity: 0.3,
        }}
      />
      <Group justify="space-between" mb={8}>
        <Text size="xs" c="dimmed" tt="uppercase" fw={600} style={{ letterSpacing: '0.04em' }}>
          {label}
        </Text>
        <ThemeIcon size="sm" variant="light" color={color} radius="sm">
          {icon}
        </ThemeIcon>
      </Group>
      {loading ? (
        <Skeleton height={28} width={80} radius="sm" />
      ) : (
        <Text fw={800} size="xl" ff="monospace">{value}</Text>
      )}
      {sub && <Group gap={4} mt={4}>{sub}</Group>}
    </Card>
  );
}

// ─── Area chart ──────────────────────────────────────────────────────────────
function VisitsAreaChart({ data }: { data: ClickPoint[] }) {
  if (!data || data.length === 0)
    return <Center h={140}><Text c="dimmed" size="sm">Нет данных</Text></Center>;

  const formatted = data.map(d => ({
    date: d.date.slice(5),   // MM-DD
    clicks: d.clicks,
  }));

  return (
    <ResponsiveContainer width="100%" height={140}>
      <AreaChart data={formatted} margin={{ top: 4, right: 4, left: -20, bottom: 0 }}>
        <defs>
          <linearGradient id="clicksGrad" x1="0" y1="0" x2="0" y2="1">
            <stop offset="5%"  stopColor="#4dabf7" stopOpacity={0.35} />
            <stop offset="95%" stopColor="#4dabf7" stopOpacity={0} />
          </linearGradient>
        </defs>
        <CartesianGrid strokeDasharray="3 3" stroke="var(--mantine-color-default-border)" />
        <XAxis dataKey="date" tick={{ fontSize: 11 }} tickLine={false} axisLine={false} />
        <YAxis allowDecimals={false} tick={{ fontSize: 11 }} tickLine={false} axisLine={false} />
        <RTooltip
          contentStyle={{
            background: 'var(--mantine-color-dark-7)',
            border: '1px solid var(--mantine-color-dark-4)',
            borderRadius: 6,
            fontSize: 12,
          }}
          formatter={(v: number) => [v, 'Кликов']}
        />
        <Area type="monotone" dataKey="clicks" stroke="#4dabf7" fill="url(#clicksGrad)" strokeWidth={2} dot={false} />
      </AreaChart>
    </ResponsiveContainer>
  );
}

// ─── Heatmap ─────────────────────────────────────────────────────────────────
function HeatmapChart({ data }: { data: HeatCell[] }) {
  if (!data || data.length === 0)
    return <Center h={80}><Text c="dimmed" size="sm">Нет данных</Text></Center>;

  const maxVal = Math.max(...data.map(d => d.value), 1);
  const grid: Record<string, number> = {};
  for (const d of data) grid[`${d.weekday}-${d.hour}`] = d.value;

  return (
    <Box style={{ overflowX: 'auto' }}>
      <Box style={{ display: 'grid', gridTemplateColumns: '32px repeat(24, 1fr)', gap: 2, minWidth: 520 }}>
        <div />
        {Array.from({ length: 24 }, (_, h) => (
          <Text key={h} size="xs" c="dimmed" ta="center" style={{ lineHeight: 1.2 }}>
            {h % 3 === 0 ? h : ''}
          </Text>
        ))}
        {DAYS.map((day, wd) => (
          <>
            <Text key={`lbl-${wd}`} size="xs" c="dimmed" style={{ alignSelf: 'center', lineHeight: 1.2 }}>
              {day}
            </Text>
            {Array.from({ length: 24 }, (_, h) => {
              const v = grid[`${wd}-${h}`] ?? 0;
              const alpha = v === 0 ? 0 : 0.15 + (v / maxVal) * 0.85;
              return (
                <Tooltip key={h} label={`${day} ${h}:00 — ${v} кл.`} withArrow position="top">
                  <Box
                    style={{
                      height: 14,
                      borderRadius: 2,
                      background: `rgba(77,171,247,${alpha})`,
                    }}
                  />
                </Tooltip>
              );
            })}
          </>
        ))}
      </Box>
    </Box>
  );
}

// ─── Top URLs tab ─────────────────────────────────────────────────────────────
function UrlsTab({ period, isAdmin }: { period: string; isAdmin: boolean }) {
  const navigate = useNavigate();
  const [urls, setUrls] = useState<TopLink[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    setLoading(true);
    api.get<{ urls: TopLink[] }>('/api/shlink/short-urls', {
      params: { page: 1, itemsPerPage: 10, orderBy: 'visits', period },
    })
      .then(r => setUrls((r as any).shortUrls?.data ?? []))
      .catch(() => setUrls([]))
      .finally(() => setLoading(false));
  }, [period]);

  if (loading) return <Center h={100}><Loader size="sm" /></Center>;
  if (!urls.length) return <Text c="dimmed" size="sm">Нет данных</Text>;

  return (
    <Table highlightOnHover withTableBorder withColumnBorders={false}>
      <Table.Thead>
        <Table.Tr>
          <Table.Th>Ссылка</Table.Th>
          <Table.Th>Назначение</Table.Th>
          <Table.Th ta="right">Переходов</Table.Th>
        </Table.Tr>
      </Table.Thead>
      <Table.Tbody>
        {urls.slice(0, 10).map(u => (
          <Table.Tr
            key={u.shortCode}
            style={{ cursor: 'pointer' }}
            onClick={() => navigate(`/links/${u.shortCode}`)}
          >
            <Table.Td>
              <Text size="sm" ff="monospace" c="blue">{u.shortCode}</Text>
            </Table.Td>
            <Table.Td>
              <Text size="sm" truncate maw={340} c="dimmed">{u.title || u.longUrl}</Text>
            </Table.Td>
            <Table.Td ta="right">
              <Badge variant="light" color="blue" size="sm">{u.visitsTotal}</Badge>
            </Table.Td>
          </Table.Tr>
        ))}
      </Table.Tbody>
    </Table>
  );
}

// ─── Main component ──────────────────────────────────────────────────────────
export function Dashboard() {
  const navigate  = useNavigate();
  const isAdmin   = useIsAdmin();

  const [data,    setData]    = useState<DashboardData | null>(null);
  const [loading, setLoading] = useState(true);
  const [period,  setPeriod]  = useState('30');

  useEffect(() => {
    setLoading(true);
    api.get<DashboardData>('/api/dashboard', { params: { period } })
      .then(setData)
      .catch(() => setData(null))
      .finally(() => setLoading(false));
  }, [period]);

  const overview = data?.overview;
  const visits   = data?.visits;
  const devices  = data?.devices;
  const users    = data?.users;

  const devTotal = devices
    ? (devices.devices.desktop + devices.devices.mobile + devices.devices.tablet) || 1
    : 1;

  // Сравниваем первую и вторую половины периода для trend
  const clickArr  = visits?.clicksPerDay ?? [];
  const half      = Math.floor(clickArr.length / 2);
  const prevHalf  = clickArr.slice(0, half).reduce((s, d) => s + d.clicks, 0);
  const currHalf  = clickArr.slice(half).reduce((s, d) => s + d.clicks, 0);

  return (
    <Container fluid px={0}>
      <Stack gap="lg">
        {/* Header */}
        <Group justify="space-between" wrap="nowrap">
          <div>
            <Title order={2} fw={700}>Дашборд</Title>
            <Text size="sm" c="dimmed">Статистика и аналитика</Text>
          </div>
          <Select
            size="xs"
            value={period}
            onChange={v => v && setPeriod(v)}
            data={[
              { value: '7',  label: '7 дней' },
              { value: '30', label: '30 дней' },
              { value: '90', label: '90 дней' },
            ]}
            w={110}
            styles={{ input: { fontWeight: 600 } }}
          />
        </Group>

        {/* KPI */}
        <SimpleGrid cols={{ base: 2, sm: 4 }} spacing="sm">
          <StatCard
            icon={<IconLink size={14} />}
            label="Активных ссылок"
            value={loading ? '…' : (overview?.linksCount ?? 0)}
            color="blue"
            loading={loading}
          />
          <StatCard
            icon={<IconClick size={14} />}
            label="Переходов всего"
            value={loading ? '…' : (overview?.visitsTotal ?? 0)}
            color="teal"
            loading={loading}
          />
          <StatCard
            icon={<IconTrendingUp size={14} />}
            label={`Кликов за ${period} д.`}
            value={loading ? '…' : (visits?.clicksTotal ?? 0)}
            color="violet"
            loading={loading}
            sub={
              !loading && [
                trendIcon(currHalf, prevHalf),
                <Text key="t" size="xs" c={currHalf >= prevHalf ? 'green' : 'red'}>
                  {prevHalf > 0
                    ? `${Math.abs(Math.round((currHalf - prevHalf) / prevHalf * 100))}% к пред. периоду`
                    : 'Нет данных для сравнения'
                  }
                </Text>,
              ]
            }
          />
          {isAdmin && (
            <StatCard
              icon={<IconUsers size={14} />}
              label="Пользователей"
              value={loading ? '…' : (users?.length ?? 0)}
              color="orange"
              loading={loading}
            />
          )}
        </SimpleGrid>

        {/* Clicks area chart */}
        <Card withBorder p="md" radius="md">
          <Group justify="space-between" mb="sm">
            <Text fw={600} size="sm">Клики по дням</Text>
            {!loading && visits && (
              <Badge variant="light" color="blue" size="sm">
                {visits.clicksTotal} всего
              </Badge>
            )}
          </Group>
          {loading ? <Skeleton height={140} radius="sm" /> : <VisitsAreaChart data={visits?.clicksPerDay ?? []} />}
        </Card>

        {/* Devices + Browsers + Users */}
        <Grid gutter="sm">
          {/* Devices */}
          <Grid.Col span={{ base: 12, md: devices && isAdmin ? 4 : 6 }}>
            <Card withBorder p="md" radius="md" h="100%">
              <Text fw={600} size="sm" mb="md">Устройства</Text>
              {loading ? (
                <Stack gap="xs">{[1,2,3].map(i => <Skeleton key={i} height={20} radius="sm" />)}</Stack>
              ) : (
                <Stack gap="md">
                  {[
                    { label: 'Desktop', val: devices?.devices.desktop ?? 0, icon: <IconDeviceDesktop size={14} />, color: 'blue'   },
                    { label: 'Mobile',  val: devices?.devices.mobile  ?? 0, icon: <IconDeviceMobile  size={14} />, color: 'teal'   },
                    { label: 'Tablet',  val: devices?.devices.tablet  ?? 0, icon: <IconDeviceTablet  size={14} />, color: 'violet' },
                  ].map(({ label, val, icon, color }) => (
                    <div key={label}>
                      <Group justify="space-between" mb={4}>
                        <Group gap={6}>{icon}<Text size="sm">{label}</Text></Group>
                        <Text size="sm" fw={600} ff="monospace">{val}</Text>
                      </Group>
                      <Progress value={(val / devTotal) * 100} color={color} size="xs" radius="xl" />
                    </div>
                  ))}
                </Stack>
              )}
            </Card>
          </Grid.Col>

          {/* Browsers */}
          <Grid.Col span={{ base: 12, md: devices && isAdmin ? 4 : 6 }}>
            <Card withBorder p="md" radius="md" h="100%">
              <Text fw={600} size="sm" mb="md">Браузеры</Text>
              {loading ? (
                <Stack gap="xs">{[1,2,3,4].map(i => <Skeleton key={i} height={18} radius="sm" />)}</Stack>
              ) : (
                <Stack gap="xs">
                  {(devices?.browsers ?? []).slice(0, 6).map((b, i) => {
                    const total = (devices?.browsers ?? []).reduce((s, x) => s + x.count, 0) || 1;
                    return (
                      <div key={b.name}>
                        <Group justify="space-between" mb={2}>
                          <Text size="sm">{b.name}</Text>
                          <Text size="xs" c="dimmed" ff="monospace">{b.count}</Text>
                        </Group>
                        <Progress value={(b.count / total) * 100} color={['blue','teal','violet','orange','green','red'][i % 6]} size="xs" radius="xl" />
                      </div>
                    );
                  })}
                </Stack>
              )}
            </Card>
          </Grid.Col>

          {/* Users (admin only) */}
          {isAdmin && (
            <Grid.Col span={{ base: 12, md: 4 }}>
              <Card withBorder p="md" radius="md" h="100%">
                <Group justify="space-between" mb="md">
                  <Text fw={600} size="sm">Пользователи</Text>
                  <Anchor size="xs" onClick={() => navigate('/admin/users')}>Все →</Anchor>
                </Group>
                {loading ? (
                  <Stack gap="xs">{[1,2,3].map(i => <Skeleton key={i} height={36} radius="sm" />)}</Stack>
                ) : (
                  <Stack gap="xs">
                    {(users ?? []).slice(0, 5).map(u => (
                      <Group
                        key={u.sub}
                        justify="space-between"
                        style={{ cursor: 'pointer' }}
                        onClick={() => navigate(`/admin/users/${u.sub}`)}
                        p={4}
                        styles={{ root: { borderRadius: 6, '&:hover': { background: 'var(--mantine-color-default-hover)' } } }}
                      >
                        <Group gap="xs">
                          <Avatar size="xs" radius="xl" color={u.role === 'admin' ? 'red' : 'blue'}>
                            {u.username[0]?.toUpperCase()}
                          </Avatar>
                          <div>
                            <Text size="xs" fw={500}>{u.username}</Text>
                            <Text size="xs" c="dimmed">{u.email}</Text>
                          </div>
                        </Group>
                        <Badge
                          variant="light"
                          color={u.status === 'active' ? 'green' : 'gray'}
                          size="xs"
                        >
                          {u.role}
                        </Badge>
                      </Group>
                    ))}
                  </Stack>
                )}
              </Card>
            </Grid.Col>
          )}
        </Grid>

        {/* Heatmap */}
        <Card withBorder p="md" radius="md">
          <Text fw={600} size="sm" mb="md">Тепловая карта активности</Text>
          {loading
            ? <Skeleton height={120} radius="sm" />
            : <HeatmapChart data={devices?.heatmap ?? []} />
          }
        </Card>

        {/* Tabs: top / recent */}
        <Card withBorder p={0} radius="md" style={{ overflow: 'hidden' }}>
          <Tabs defaultValue="top">
            <Tabs.List px="md" pt="xs">
              <Tabs.Tab value="top">Топ ссылок</Tabs.Tab>
              {isAdmin && <Tabs.Tab value="recent">Последние</Tabs.Tab>}
            </Tabs.List>

            <Box p="md">
              <Tabs.Panel value="top">
                <UrlsTab period={period} isAdmin={isAdmin} />
              </Tabs.Panel>

              {isAdmin && (
                <Tabs.Panel value="recent">
                  {loading ? (
                    <Center h={80}><Loader size="sm" /></Center>
                  ) : (
                    <Table highlightOnHover withTableBorder={false}>
                      <Table.Thead>
                        <Table.Tr>
                          <Table.Th>Код</Table.Th>
                          <Table.Th>Назначение</Table.Th>
                          <Table.Th>Создана</Table.Th>
                          <Table.Th ta="right">Кликов</Table.Th>
                        </Table.Tr>
                      </Table.Thead>
                      <Table.Tbody>
                        {(overview?.recentLinks ?? []).map(u => (
                          <Table.Tr
                            key={u.shortCode}
                            style={{ cursor: 'pointer' }}
                            onClick={() => navigate(`/admin/urls/${u.shortCode}`)}
                          >
                            <Table.Td><Text size="sm" ff="monospace" c="blue">{u.shortCode}</Text></Table.Td>
                            <Table.Td><Text size="sm" truncate maw={280} c="dimmed">{u.title || u.longUrl}</Text></Table.Td>
                            <Table.Td><Text size="xs" c="dimmed">{formatDate(u.longUrl)}</Text></Table.Td>
                            <Table.Td ta="right"><Badge variant="light" size="xs">{u.visitsTotal}</Badge></Table.Td>
                          </Table.Tr>
                        ))}
                      </Table.Tbody>
                    </Table>
                  )}
                </Tabs.Panel>
              )}
            </Box>
          </Tabs>
        </Card>
      </Stack>
    </Container>
  );
}
