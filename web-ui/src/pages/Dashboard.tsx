import { useEffect, useState } from 'react';
import {
  Container, Title, Text, Group, Stack, Grid, Card,
  Badge, Table, Select, Tabs, Loader, Center,
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

// Shlink proxy response shape for /api/shlink/short-urls
interface ShlinkURLsResponse {
  shortURLs: {
    data: TopLink[];
    pagination: { currentPage: number; pagesCount: number; itemsPerPage: number; itemsInCurrentPage: number; totalItems: number; };
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
    api.get<ShlinkURLsResponse>('/api/shlink/short-urls', {
      params: { page: 1, itemsPerPage: 10, orderBy: 'visits', period },
    })
      .then(r => setUrls(r.shortURLs?.data ?? []))
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
            icon={<IconTrendingUp size={14} />} label="Посещений (период)" color="orange" loading={loading}
            value={visits?.clicksTotal ?? 0}
          />
        )}
      </SimpleGrid>

      <Grid gutter="md" mb="lg">
        {/* Visits chart */}
        <Grid.Col span={{ base: 12, md: 8 }}>
          <Card withBorder p="md" radius="md" h="100%">
            <Text fw={600} size="sm" mb="sm">Переходы по дням</Text>
            {loading
              ? <Skeleton height={140} radius="sm" />
              : <VisitsAreaChart data={visits?.clicksPerDay ?? []} />
            }
          </Card>
        </Grid.Col>

        {/* Devices */}
        <Grid.Col span={{ base: 12, md: 4 }}>
          <Card withBorder p="md" radius="md" h="100%">
            <Text fw={600} size="sm" mb="sm">Устройства</Text>
            {loading ? <Skeleton height={140} radius="sm" /> : (() => {
              const d = devices?.devices;
              if (!d) return <Text c="dimmed" size="sm">Нет данных</Text>;
              const total = (d.desktop + d.mobile + d.tablet) || 1;
              return (
                <Stack gap="xs">
                  {([
                    { icon: <IconDeviceDesktop size={14} />, label: 'Desktop', val: d.desktop, color: 'blue' },
                    { icon: <IconDeviceMobile  size={14} />, label: 'Mobile',  val: d.mobile,  color: 'teal' },
                    { icon: <IconDeviceTablet  size={14} />, label: 'Tablet',  val: d.tablet,  color: 'grape' },
                  ] as const).map(({ icon, label, val, color }) => (
                    <Box key={label}>
                      <Group justify="space-between" mb={2}>
                        <Group gap={4}>{icon}<Text size="xs">{label}</Text></Group>
                        <Text size="xs" c="dimmed">{val} ({Math.round(val / total * 100)}%)</Text>
                      </Group>
                      <Progress value={val / total * 100} color={color} size="xs" radius="xl" />
                    </Box>
                  ))}
                </Stack>
              );
            })()}
          </Card>
        </Grid.Col>
      </Grid>

      {/* Main tabs */}
      <Tabs defaultValue="top">
        <Tabs.List mb="md">
          <Tabs.Tab value="top">Топ ссылок</Tabs.Tab>
          <Tabs.Tab value="recent">Последние</Tabs.Tab>
          {devices?.browsers?.length ? <Tabs.Tab value="browsers">Браузеры</Tabs.Tab> : null}
          {devices?.heatmap?.length  ? <Tabs.Tab value="heatmap">Тепловая карта</Tabs.Tab> : null}
          {isAdmin && users?.length  ? <Tabs.Tab value="users">Пользователи</Tabs.Tab> : null}
        </Tabs.List>

        <Tabs.Panel value="top">
          <UrlsTab period={period} />
        </Tabs.Panel>

        <Tabs.Panel value="recent">
          {loading ? <Center h={100}><Loader size="sm" /></Center> : (
            overview?.recentLinks?.length
              ? (
                <Table highlightOnHover withTableBorder>
                  <Table.Thead>
                    <Table.Tr>
                      <Table.Th>Ссылка</Table.Th>
                      <Table.Th>Назначение</Table.Th>
                      <Table.Th>Создана</Table.Th>
                      <Table.Th ta="right">Переходов</Table.Th>
                    </Table.Tr>
                  </Table.Thead>
                  <Table.Tbody>
                    {overview.recentLinks.map(u => (
                      <Table.Tr key={u.shortCode}>
                        <Table.Td><Text size="sm" ff="monospace" c="blue">{u.shortCode}</Text></Table.Td>
                        <Table.Td><Text size="sm" truncate maw={320} c="dimmed">{u.title || u.longUrl}</Text></Table.Td>
                        <Table.Td><Text size="sm" c="dimmed">—</Text></Table.Td>
                        <Table.Td ta="right"><Badge variant="light" size="sm">{u.visitsTotal}</Badge></Table.Td>
                      </Table.Tr>
                    ))}
                  </Table.Tbody>
                </Table>
              )
              : <Text c="dimmed" size="sm">Нет данных</Text>
          )}
        </Tabs.Panel>

        <Tabs.Panel value="browsers">
          {loading ? <Center h={100}><Loader size="sm" /></Center> : (
            <Stack gap="xs" maw={400}>
              {(devices?.browsers ?? []).map(b => (
                <Group key={b.name} justify="space-between">
                  <Text size="sm">{b.name}</Text>
                  <Badge variant="light" size="sm">{b.count}</Badge>
                </Group>
              ))}
            </Stack>
          )}
        </Tabs.Panel>

        <Tabs.Panel value="heatmap">
          {loading ? <Center h={100}><Loader size="sm" /></Center> : (
            <HeatmapChart cells={devices?.heatmap ?? []} />
          )}
        </Tabs.Panel>

        {isAdmin && (
          <Tabs.Panel value="users">
            {loading ? <Center h={100}><Loader size="sm" /></Center> : (
              users?.length
                ? (
                  <Table highlightOnHover withTableBorder>
                    <Table.Thead>
                      <Table.Tr>
                        <Table.Th>Пользователь</Table.Th>
                        <Table.Th>Email</Table.Th>
                        <Table.Th>Роль</Table.Th>
                        <Table.Th>Статус</Table.Th>
                      </Table.Tr>
                    </Table.Thead>
                    <Table.Tbody>
                      {users.map(u => (
                        <Table.Tr key={u.sub}>
                          <Table.Td>
                            <Group gap="xs">
                              <Avatar size="sm" radius="xl" color="blue">
                                {u.username?.[0]?.toUpperCase() ?? '?'}
                              </Avatar>
                              <Text size="sm">{u.username}</Text>
                            </Group>
                          </Table.Td>
                          <Table.Td><Text size="sm" c="dimmed">{u.email}</Text></Table.Td>
                          <Table.Td><Badge size="sm" variant="light">{u.role}</Badge></Table.Td>
                          <Table.Td>
                            <Badge
                              size="sm"
                              color={u.status === 'active' ? 'green' : u.status === 'disabled' ? 'red' : 'yellow'}
                            >
                              {u.status}
                            </Badge>
                          </Table.Td>
                        </Table.Tr>
                      ))}
                    </Table.Tbody>
                  </Table>
                )
                : <Text c="dimmed" size="sm">Нет пользователей</Text>
            )}
          </Tabs.Panel>
        )}
      </Tabs>
    </Container>
  );
}
