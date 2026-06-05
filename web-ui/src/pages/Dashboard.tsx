import { useEffect, useState } from 'react';
import {
  Container, Title, Text, Group, Stack, Grid, Card,
  Badge, Table, Anchor, Select, Tabs, Loader, Center,
  Avatar, SimpleGrid, Progress, Tooltip, ThemeIcon,
  Skeleton, Box,
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

// ─── types ────────────────────────────────────────────────────────────────────────────
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

// ─── helpers ──────────────────────────────────────────────────────────────────────────────
const DAYS = ['Вс','Пн','Вт','Ср','Чт','Пт','Сб'];

function trendIcon(val: number, prev: number) {
  if (!prev) return <IconMinus size={14} />;
  return val >= prev
    ? <IconArrowUpRight size={14} color="var(--mantine-color-green-5)" />
    : <IconArrowDownRight size={14} color="var(--mantine-color-red-5)" />;
}

// ─── KPI Card ──────────────────────────────────────────────────────────────────────────────
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

// ─── Area chart ──────────────────────────────────────────────────────────────────────────────
function VisitsAreaChart({ data }: { data: ClickPoint[] }) {
  if (!data || data.length === 0)
    return <Center h={140}><Text c="dimmed" size="sm">Нет данных</Text></Center>;
  return (
    <ResponsiveContainer width="100%" height={140}>
      <AreaChart data={data} margin={{ top: 4, right: 8, bottom: 0, left: -24 }}>
        <defs>
          <linearGradient id="visitsGrad" x1="0" y1="0" x2="0" y2="1">
            <stop offset="5%"  stopColor="var(--mantine-color-blue-5)"  stopOpacity={0.3} />
            <stop offset="95%" stopColor="var(--mantine-color-blue-5)"  stopOpacity={0.02} />
          </linearGradient>
        </defs>
        <CartesianGrid strokeDasharray="3 3" stroke="var(--mantine-color-gray-2)" />
        <XAxis dataKey="date" tick={{ fontSize: 10 }} tickFormatter={d => d.slice(5)} />
        <YAxis tick={{ fontSize: 10 }} allowDecimals={false} />
        <RTooltip
          contentStyle={{ fontSize: 12, borderRadius: 6 }}
          formatter={(v: number) => [v, 'Переходов']}
        />
        <Area
          type="monotone" dataKey="clicks"
          stroke="var(--mantine-color-blue-5)" strokeWidth={2}
          fill="url(#visitsGrad)"
        />
      </AreaChart>
    </ResponsiveContainer>
  );
}

// ─── Heatmap ─────────────────────────────────────────────────────────────────────────────────
function HeatmapChart({ cells }: { cells: HeatCell[] }) {
  const maxVal = Math.max(...cells.map(c => c.value), 1);
  const grid: number[][] = Array.from({ length: 7 }, () => new Array(24).fill(0));
  cells.forEach(c => { grid[c.weekday][c.hour] = c.value; });

  return (
    <Box style={{ overflowX: 'auto' }}>
      <Box style={{ display: 'grid', gridTemplateColumns: 'auto repeat(24, 1fr)', gap: 2, minWidth: 580 }}>
        <Box />
        {Array.from({ length: 24 }, (_, h) => (
          <Text key={h} size="xs" c="dimmed" ta="center" style={{ fontSize: 9 }}>{h}</Text>
        ))}
        {grid.map((row, wd) => (
          <>
            <Text key={`d${wd}`} size="xs" c="dimmed" style={{ fontSize: 10, paddingRight: 4, alignSelf: 'center' }}>
              {DAYS[wd]}
            </Text>
            {row.map((val, hr) => {
              const intensity = val ? Math.max(0.15, val / maxVal) : 0;
              return (
                <Tooltip key={`${wd}-${hr}`} label={`${DAYS[wd]} ${hr}:00 — ${val} переходов`} withArrow>
                  <Box
                    style={{
                      height: 14, borderRadius: 2,
                      background: val
                        ? `rgba(34,139,230,${intensity})`
                        : 'var(--mantine-color-gray-1)',
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

// ─── Top URLs tab ────────────────────────────────────────────────────────────────────────────────
function UrlsTab({ period }: { period: string }) {
  const navigate = useNavigate();
  const [urls, setUrls] = useState<TopLink[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    setLoading(true);
    api.get<{ urls: TopLink[] }>('/api/shlink/short-urls', {
      params: { page: 1, itemsPerPage: 10, orderBy: 'visits', period },
    })
      .then((r: { shortUrls?: { data: TopLink[] } }) => setUrls(r.shortUrls?.data ?? []))
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

// ─── Main component ──────────────────────────────────────────────────────────────────────────────
export function Dashboard() {
  const navigate  = useNavigate();
  const isAdmin   = useIsAdmin();

  const [data,    setData]    = useState<DashboardData | null>(null);
  const [loading, setLoading] = useState(true);
  const [period,  setPeriod]  = useState('30');

  useEffect(() => {
    setLoading(true);
    api.get<DashboardData>('/api/dashboard', { params: { period } })
      .then(d => setData(d))
      .catch(() => setData(null))
      .finally(() => setLoading(false));
  }, [period]);

  const overview = data?.overview;
  const visits   = data?.visits;
  const devices  = data?.devices;
  const users    = data?.users;

  return (
    <Container size="xl" py="lg">
      <Group justify="space-between" mb="lg">
        <Stack gap={2}>
          <Title order={2}>Дашборд</Title>
          <Text c="dimmed" size="sm">Статистика за период</Text>
        </Stack>
        <Select
          size="xs"
          value={period}
          onChange={v => v && setPeriod(v)}
          data={[
            { value: '7',  label: '7 дней' },
            { value: '14', label: '14 дней' },
            { value: '30', label: '30 дней' },
            { value: '90', label: '90 дней' },
          ]}
          w={120}
        />
      </Group>

      {/* KPI row */}
      <SimpleGrid cols={{ base: 2, sm: 2, md: isAdmin ? 4 : 2 }} mb="lg">
        <StatCard
          icon={<IconLink size={14} />} label="Ссылок" color="blue" loading={loading}
          value={overview?.linksCount ?? 0}
        />
        <StatCard
          icon={<IconClick size={14} />} label="Переходов" color="teal" loading={loading}
          value={overview?.visitsTotal ?? 0}
          sub={visits && <>
            {trendIcon(visits.clicksTotal, 0)}
            <Text size="xs" c="dimmed">за {period} д.</Text>
          </>}
        />
        {isAdmin && (
          <StatCard
            icon={<IconUsers size={14} />} label="Пользователей" color="grape" loading={loading}
            value={users?.length ?? 0}
          />
        )}
        {isAdmin && (
          <StatCard
            icon={<IconTrendingUp size={14} />} label="Активных" color="orange" loading={loading}
            value={users?.filter(u => u.status === 'active').length ?? 0}
          />
        )}
      </SimpleGrid>

      {/* Visits chart */}
      <Card withBorder mb="lg" p="md">
        <Text fw={600} size="sm" mb="xs">Переходы по дням</Text>
        {loading
          ? <Skeleton height={140} radius="sm" />
          : <VisitsAreaChart data={visits?.clicksPerDay ?? []} />
        }
      </Card>

      {/* Device breakdown */}
      <Grid mb="lg">
        <Grid.Col span={{ base: 12, md: devices && isAdmin ? 4 : 6 }}>
          <Card withBorder p="md" h="100%">
            <Text fw={600} size="sm" mb="xs">Устройства</Text>
            {loading ? <Skeleton height={80} /> : devices ? (
              <Stack gap="xs">
                {([
                  ['desktop', 'Компьютер', <IconDeviceDesktop size={14} />],
                  ['mobile',  'Мобильный', <IconDeviceMobile  size={14} />],
                  ['tablet',  'Планшет',   <IconDeviceTablet  size={14} />],
                ] as const).map(([key, label, icon]) => {
                  const val  = devices.devices[key as keyof typeof devices.devices] ?? 0;
                  const total = Object.values(devices.devices).reduce((a, b) => a + b, 0);
                  const pct  = total ? Math.round(val / total * 100) : 0;
                  return (
                    <Group key={key} gap="xs">
                      {icon}
                      <Text size="xs" w={80}>{label}</Text>
                      <Progress value={pct} flex={1} size="sm" color="blue" />
                      <Text size="xs" c="dimmed" w={36} ta="right">{pct}%</Text>
                    </Group>
                  );
                })}
              </Stack>
            ) : <Text c="dimmed" size="sm">Нет данных</Text>}
          </Card>
        </Grid.Col>

        <Grid.Col span={{ base: 12, md: devices && isAdmin ? 4 : 6 }}>
          <Card withBorder p="md" h="100%">
            <Text fw={600} size="sm" mb="xs">Браузеры</Text>
            {loading ? <Skeleton height={80} /> : devices?.browsers?.length ? (
              <Stack gap={4}>
                {devices.browsers.slice(0, 5).map(b => (
                  <Group key={b.name} gap="xs" justify="space-between">
                    <Text size="xs">{b.name}</Text>
                    <Badge variant="light" size="xs">{b.count}</Badge>
                  </Group>
                ))}
              </Stack>
            ) : <Text c="dimmed" size="sm">Нет данных</Text>}
          </Card>
        </Grid.Col>

        {isAdmin && (
          <Grid.Col span={{ base: 12, md: 4 }}>
            <Card withBorder p="md" h="100%">
              <Text fw={600} size="sm" mb="xs">Операционные системы</Text>
              {loading ? <Skeleton height={80} /> : devices?.os?.length ? (
                <Stack gap={4}>
                  {devices.os.slice(0, 5).map(o => (
                    <Group key={o.name} gap="xs" justify="space-between">
                      <Text size="xs">{o.name}</Text>
                      <Badge variant="light" size="xs" color="grape">{o.count}</Badge>
                    </Group>
                  ))}
                </Stack>
              ) : <Text c="dimmed" size="sm">Нет данных</Text>}
            </Card>
          </Grid.Col>
        )}
      </Grid>

      {/* Heatmap */}
      {devices?.heatmap?.length ? (
        <Card withBorder mb="lg" p="md">
          <Text fw={600} size="sm" mb="xs">Активность по часам / дням</Text>
          <HeatmapChart cells={devices.heatmap} />
        </Card>
      ) : null}

      {/* Users table (admin) */}
      {isAdmin && users && (
        <Card withBorder mb="lg" p="md">
          <Text fw={600} size="sm" mb="xs">Пользователи</Text>
          <Table highlightOnHover withTableBorder withColumnBorders={false}>
            <Table.Thead>
              <Table.Tr>
                <Table.Th>Пользователь</Table.Th>
                <Table.Th>Email</Table.Th>
                <Table.Th>Роль</Table.Th>
                <Table.Th>Статус</Table.Th>
                <Table.Th>Действия</Table.Th>
              </Table.Tr>
            </Table.Thead>
            <Table.Tbody>
              {users.map(u => (
                <Table.Tr key={u.sub}>
                  <Table.Td>
                    <Group gap="xs">
                      <Avatar size="sm" radius="xl" color="blue">
                        {u.username.slice(0, 2).toUpperCase()}
                      </Avatar>
                      <Text size="sm">{u.username}</Text>
                    </Group>
                  </Table.Td>
                  <Table.Td><Text size="sm" c="dimmed">{u.email}</Text></Table.Td>
                  <Table.Td><Badge variant="light" size="sm">{u.role}</Badge></Table.Td>
                  <Table.Td>
                    <Badge
                      variant="light" size="sm"
                      color={u.status === 'active' ? 'green' : 'red'}
                    >
                      {u.status === 'active' ? 'Активен' : 'Отключён'}
                    </Badge>
                  </Table.Td>
                  <Table.Td>
                    <Anchor
                      size="xs"
                      onClick={() => navigate(`/admin/users/${u.sub}`)}
                    >
                      Профиль
                    </Anchor>
                  </Table.Td>
                </Table.Tr>
              ))}
            </Table.Tbody>
          </Table>
        </Card>
      )}

      {/* Links tabs */}
      <Card withBorder p="md">
        <Tabs defaultValue="top">
          <Tabs.List mb="md">
            <Tabs.Tab value="top">Популярные</Tabs.Tab>
            {isAdmin && <Tabs.Tab value="recent">Последние</Tabs.Tab>}
          </Tabs.List>
          <Tabs.Panel value="top">
            <UrlsTab period={period} />
          </Tabs.Panel>
          {isAdmin && (
            <Tabs.Panel value="recent">
              {overview?.recentLinks && (
                <Table highlightOnHover withTableBorder withColumnBorders={false}>
                  <Table.Thead>
                    <Table.Tr>
                      <Table.Th>Ссылка</Table.Th>
                      <Table.Th>Назначение</Table.Th>
                      <Table.Th ta="right">Переходов</Table.Th>
                    </Table.Tr>
                  </Table.Thead>
                  <Table.Tbody>
                    {overview.recentLinks.map(u => (
                      <Table.Tr
                        key={u.shortCode}
                        style={{ cursor: 'pointer' }}
                        onClick={() => navigate(`/links/${u.shortCode}`)}
                      >
                        <Table.Td>
                          <Text size="sm" ff="monospace" c="blue">{u.shortCode}</Text>
                        </Table.Td>
                        <Table.Td>
                          <Text size="sm" truncate maw={340} c="dimmed">
                            {u.title || u.longUrl}
                          </Text>
                        </Table.Td>
                        <Table.Td ta="right">
                          <Badge variant="light" color="teal" size="sm">{u.visitsTotal}</Badge>
                        </Table.Td>
                      </Table.Tr>
                    ))}
                  </Table.Tbody>
                </Table>
              )}
            </Tabs.Panel>
          )}
        </Tabs>
      </Card>

      <Text size="xs" c="dimmed" ta="center" mt="lg">
        Данные обновляются при каждом открытии дашборда · Период: {period} дней
        {data && <> · {formatDate(new Date().toISOString())}</>}
      </Text>
    </Container>
  );
}
