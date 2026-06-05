import { useEffect, useState } from 'react';
import {
  Container,
  Title,
  Text,
  Group,
  Stack,
  Grid,
  Card,
  Badge,
  Table,
  Anchor,
  Select,
  Tabs,
  Loader,
  Center,
  Avatar,
  ThemeIcon,
  SimpleGrid,
  Progress,
  Tooltip,
} from '@mantine/core';
import {
  IconLink,
  IconClick,
  IconUsers,
  IconTrendingUp,
  IconDeviceDesktop,
  IconDeviceMobile,
  IconDeviceTablet,
  IconExternalLink,
} from '@tabler/icons-react';
import { useNavigate } from 'react-router-dom';
import { useIsAdmin } from '../contexts/AuthContext';
import { api } from '../api/client';

interface DashboardData {
  overview: {
    linksCount: number;
    visitsTotal: number;
    topLinks: Array<{
      shortCode: string;
      shortUrl: string;
      longUrl: string;
      title: string;
      visitsTotal: number;
    }>;
    recentLinks: Array<{
      shortCode: string;
      shortUrl: string;
      longUrl: string;
      title: string;
      visitsTotal: number;
    }>;
  };
  users: Array<{
    sub: string;
    username: string;
    email: string;
    role: string;
    status: string;
  }> | null;
  visits: {
    clicksPerDay: Array<{ date: string; clicks: number }>;
    clicksTotal: number;
  };
  devices: {
    devices: { desktop: number; mobile: number; tablet: number };
    browsers: Array<{ name: string; count: number }>;
    os: Array<{ name: string; count: number }>;
    heatmap: Array<{ weekday: number; hour: number; value: number }>;
  };
}

function StatCard({
  icon,
  label,
  value,
  color = 'blue',
}: {
  icon: React.ReactNode;
  label: string;
  value: string | number;
  color?: string;
}) {
  return (
    <Card withBorder p="md">
      <Group>
        <ThemeIcon size="lg" variant="light" color={color}>
          {icon}
        </ThemeIcon>
        <div>
          <Text size="xs" c="dimmed">
            {label}
          </Text>
          <Text fw={700} size="xl">
            {value}
          </Text>
        </div>
      </Group>
    </Card>
  );
}

function VisitsChart({ data }: { data: Array<{ date: string; clicks: number }> }) {
  if (!data || data.length === 0) {
    return (
      <Center h={120}>
        <Text c="dimmed" size="sm">
          Нет данных
        </Text>
      </Center>
    );
  }

  const max = Math.max(...data.map((d: { date: string; clicks: number }) => d.clicks), 1);
  const last14 = data.slice(-14);

  return (
    <Group align="flex-end" gap={3} h={80} style={{ overflow: 'hidden' }}>
      {last14.map((d: { date: string; clicks: number }) => (
        <Tooltip key={d.date} label={`${d.date}: ${d.clicks}`}>
          <div
            style={{
              flex: 1,
              height: `${Math.max((d.clicks / max) * 100, 2)}%`,
              background: 'var(--mantine-color-blue-5)',
              borderRadius: 2,
              minWidth: 4,
            }}
          />
        </Tooltip>
      ))}
    </Group>
  );
}

function HeatmapChart({
  data,
}: {
  data: Array<{ weekday: number; hour: number; value: number }>;
}) {
  const days = ['Пн', 'Вт', 'Ср', 'Чт', 'Пт', 'Сб', 'Вс'];
  const max = Math.max(...data.map((d: { weekday: number; hour: number; value: number }) => d.value), 1);
  const map = new Map(data.map((d: { weekday: number; hour: number; value: number }) => [`${d.weekday}-${d.hour}`, d.value]));

  return (
    <div style={{ overflowX: 'auto' }}>
      <div style={{ display: 'grid', gridTemplateColumns: `40px repeat(24, 1fr)`, gap: 2 }}>
        {days.map((day, wd) => (
          <>
            <Text key={`label-${wd}`} size="xs" c="dimmed" style={{ alignSelf: 'center' }}>
              {day}
            </Text>
            {Array.from({ length: 24 }, (_, hr) => {
              const v = map.get(`${wd}-${hr}`) ?? 0;
              const opacity = v > 0 ? 0.15 + (v / max) * 0.85 : 0.05;
              return (
                <Tooltip key={hr} label={`${day} ${hr}:00 — ${v} кликов`}>
                  <div
                    style={{
                      height: 14,
                      borderRadius: 2,
                      background: `rgba(34, 139, 230, ${opacity})`,
                    }}
                  />
                </Tooltip>
              );
            })}
          </>
        ))}
      </div>
    </div>
  );
}

function UrlsTab({ period, isAdmin }: { period: string; isAdmin: boolean }) {
  const navigate = useNavigate();
  const [data, setData] = useState<DashboardData | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    setLoading(true);
    api
      .get<DashboardData>(`/api/dashboard?period=${period}`)
      .then((d: DashboardData) => setData(d))
      .catch(console.error)
      .finally(() => setLoading(false));
  }, [period]);

  if (loading)
    return (
      <Center h={200}>
        <Loader />
      </Center>
    );
  if (!data) return null;

  const urls = data.overview.topLinks;

  return (
    <Stack>
      <Table highlightOnHover>
        <Table.Thead>
          <Table.Tr>
            <Table.Th>Ссылка</Table.Th>
            <Table.Th>Назначение</Table.Th>
            <Table.Th>Визиты</Table.Th>
          </Table.Tr>
        </Table.Thead>
        <Table.Tbody>
          {urls.map((u) => (
            <Table.Tr key={u.shortCode} style={{ cursor: 'pointer' }} onClick={() => navigate(isAdmin ? `/admin/urls/${u.shortCode}` : `/links/${u.shortCode}`)}>
              <Table.Td>
                <Group gap="xs">
                  <Text size="sm" ff="monospace">
                    {u.shortCode}
                  </Text>
                  <Anchor
                    href={u.shortUrl}
                    target="_blank"
                    size="xs"
                    onClick={(e) => e.stopPropagation()}
                  >
                    <IconExternalLink size={12} />
                  </Anchor>
                </Group>
              </Table.Td>
              <Table.Td>
                <Text size="sm" truncate maw={300}>
                  {u.title || u.longUrl}
                </Text>
              </Table.Td>
              <Table.Td>
                <Badge variant="light">{u.visitsTotal}</Badge>
              </Table.Td>
            </Table.Tr>
          ))}
        </Table.Tbody>
      </Table>
    </Stack>
  );
}

export function Dashboard() {
  const navigate = useNavigate();
  const isAdmin = useIsAdmin();
  const [data, setData] = useState<DashboardData | null>(null);
  const [loading, setLoading] = useState(true);
  const [period, setPeriod] = useState('30d');

  useEffect(() => {
    setLoading(true);
    api
      .get<DashboardData>(`/api/dashboard?period=${period}`)
      .then((d: DashboardData) => setData(d))
      .catch(console.error)
      .finally(() => setLoading(false));
  }, [period]);

  if (loading)
    return (
      <Center h={400}>
        <Loader size="lg" />
      </Center>
    );

  if (!data) return null;

  const { overview, users, visits, devices } = data;
  const devTotal = devices.devices.desktop + devices.devices.mobile + devices.devices.tablet || 1;

  return (
    <Container size="xl" py="xl">
      <Stack gap="xl">
        {/* Header */}
        <Group justify="space-between">
          <Title order={2}>Дашборд</Title>
          <Select
            size="xs"
            value={period}
            onChange={(v) => setPeriod(v ?? '30d')}
            data={[
              { value: '7d', label: '7 дней' },
              { value: '30d', label: '30 дней' },
              { value: '90d', label: '90 дней' },
            ]}
          />
        </Group>

        {/* Stats */}
        <SimpleGrid cols={{ base: 2, md: 4 }}>
          <StatCard
            icon={<IconLink size={18} />}
            label="Ссылок"
            value={overview.linksCount}
            color="blue"
          />
          <StatCard
            icon={<IconClick size={18} />}
            label="Всего кликов"
            value={overview.visitsTotal}
            color="teal"
          />
          <StatCard
            icon={<IconTrendingUp size={18} />}
            label={`Кликов за период`}
            value={visits.clicksTotal}
            color="violet"
          />
          {isAdmin && (
            <StatCard
              icon={<IconUsers size={18} />}
              label="Пользователей"
              value={users?.length ?? 0}
              color="orange"
            />
          )}
        </SimpleGrid>

        {/* Visits chart */}
        <Card withBorder p="md">
          <Text fw={600} mb="sm">
            Клики по дням
          </Text>
          <VisitsChart data={visits.clicksPerDay} />
        </Card>

        <Grid>
          {/* Devices */}
          <Grid.Col span={{ base: 12, md: 4 }}>
            <Card withBorder p="md" h="100%">
              <Text fw={600} mb="md">
                Устройства
              </Text>
              <Stack gap="xs">
                <Group justify="space-between">
                  <Group gap="xs">
                    <IconDeviceDesktop size={16} />
                    <Text size="sm">Desktop</Text>
                  </Group>
                  <Text size="sm" fw={500}>
                    {devices.devices.desktop}
                  </Text>
                </Group>
                <Progress
                  value={(devices.devices.desktop / devTotal) * 100}
                  color="blue"
                  size="sm"
                />
                <Group justify="space-between">
                  <Group gap="xs">
                    <IconDeviceMobile size={16} />
                    <Text size="sm">Mobile</Text>
                  </Group>
                  <Text size="sm" fw={500}>
                    {devices.devices.mobile}
                  </Text>
                </Group>
                <Progress
                  value={(devices.devices.mobile / devTotal) * 100}
                  color="teal"
                  size="sm"
                />
                <Group justify="space-between">
                  <Group gap="xs">
                    <IconDeviceTablet size={16} />
                    <Text size="sm">Tablet</Text>
                  </Group>
                  <Text size="sm" fw={500}>
                    {devices.devices.tablet}
                  </Text>
                </Group>
                <Progress
                  value={(devices.devices.tablet / devTotal) * 100}
                  color="violet"
                  size="sm"
                />
              </Stack>
            </Card>
          </Grid.Col>

          {/* Browsers */}
          <Grid.Col span={{ base: 12, md: 4 }}>
            <Card withBorder p="md" h="100%">
              <Text fw={600} mb="md">
                Браузеры
              </Text>
              <Stack gap="xs">
                {devices.browsers.slice(0, 5).map((b) => (
                  <Group key={b.name} justify="space-between">
                    <Text size="sm">{b.name}</Text>
                    <Badge variant="light" size="sm">
                      {b.count}
                    </Badge>
                  </Group>
                ))}
              </Stack>
            </Card>
          </Grid.Col>

          {/* Users (admin only) */}
          {isAdmin && users && (
            <Grid.Col span={{ base: 12, md: 4 }}>
              <Card withBorder p="md" h="100%">
                <Text fw={600} mb="md">
                  Пользователи
                </Text>
                <Stack gap="xs">
                  {users.slice(0, 5).map((u) => (
                    <Group
                      key={u.sub}
                      justify="space-between"
                      style={{ cursor: 'pointer' }}
                      onClick={() => navigate(`/admin/users/${u.sub}`)}
                    >
                      <Group gap="xs">
                        <Avatar size="sm" radius="xl">
                          {u.username[0]?.toUpperCase()}
                        </Avatar>
                        <div>
                          <Text size="sm" fw={500}>
                            {u.username}
                          </Text>
                          <Text size="xs" c="dimmed">
                            {u.email}
                          </Text>
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
              </Card>
            </Grid.Col>
          )}
        </Grid>

        {/* Heatmap */}
        <Card withBorder p="md">
          <Text fw={600} mb="md">
            Активность по дням и часам
          </Text>
          <HeatmapChart data={devices.heatmap} />
        </Card>

        {/* Tabs */}
        <Tabs defaultValue="urls">
          <Tabs.List>
            <Tabs.Tab value="urls">Топ ссылок</Tabs.Tab>
            {isAdmin && <Tabs.Tab value="recent">Последние</Tabs.Tab>}
          </Tabs.List>
          <Tabs.Panel value="urls" pt="md">
            <UrlsTab period={period} isAdmin={isAdmin} />
          </Tabs.Panel>
          {isAdmin && (
            <Tabs.Panel value="recent" pt="md">
              <Stack>
                <Table highlightOnHover>
                  <Table.Thead>
                    <Table.Tr>
                      <Table.Th>Ссылка</Table.Th>
                      <Table.Th>Назначение</Table.Th>
                      <Table.Th>Создана</Table.Th>
                    </Table.Tr>
                  </Table.Thead>
                  <Table.Tbody>
                    {overview.recentLinks.map((u) => (
                      <Table.Tr
                        key={u.shortCode}
                        style={{ cursor: 'pointer' }}
                        onClick={() => navigate(`/admin/urls/${u.shortCode}`)}
                      >
                        <Table.Td>
                          <Text size="sm" ff="monospace">
                            {u.shortCode}
                          </Text>
                        </Table.Td>
                        <Table.Td>
                          <Text size="sm" truncate maw={300}>
                            {u.title || u.longUrl}
                          </Text>
                        </Table.Td>
                        <Table.Td>
                          <Badge variant="light">{u.visitsTotal} visits</Badge>
                        </Table.Td>
                      </Table.Tr>
                    ))}
                  </Table.Tbody>
                </Table>
              </Stack>
            </Tabs.Panel>
          )}
        </Tabs>
      </Stack>
    </Container>
  );
}
