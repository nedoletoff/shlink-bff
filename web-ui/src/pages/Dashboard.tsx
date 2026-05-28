import { useEffect, useState } from 'react';
import {
  Grid, Card, Text, Group, Badge, Title, Stack, Loader, Center,
} from '@mantine/core';
import { IconClick, IconLink, IconTags } from '@tabler/icons-react';
import {
  LineChart, Line, XAxis, YAxis, CartesianGrid,
  Tooltip, ResponsiveContainer, PieChart, Pie, Cell, Legend,
} from 'recharts';
import { api } from '../api/client';
import type { DashboardResponse } from '../types/api';

const COLORS = ['#4dabf7', '#51cf66', '#ff6b6b', '#ffd43b', '#cc5de8'];

export function Dashboard() {
  const [data,    setData]    = useState<DashboardResponse | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    api.get<DashboardResponse>('/api/dashboard')
      .then(setData)
      .catch(() => setData(null))
      .finally(() => setLoading(false));
  }, []);

  if (loading) return <Center h={400}><Loader /></Center>;
  if (!data)   return <Text c="red">Ошибка загрузки дашборда</Text>;

  return (
    <Stack gap="lg">
      <Title order={2}>Dashboard</Title>

      <Grid>
        <Grid.Col span={{ base: 12, sm: 4 }}>
          <KPICard
            label="Всего кликов"
            value={data.totalClicks.toLocaleString('ru')}
            icon={<IconClick size={24} />}
            color="blue"
          />
        </Grid.Col>
        <Grid.Col span={{ base: 12, sm: 4 }}>
          <KPICard
            label="Активных ссылок"
            value={String(data.activeLinks)}
            icon={<IconLink size={24} />}
            color="green"
          />
        </Grid.Col>
        <Grid.Col span={{ base: 12, sm: 4 }}>
          <KPICard
            label="Топ тег"
            value={data.topTags[0]?.tag ?? '—'}
            icon={<IconTags size={24} />}
            color="grape"
          />
        </Grid.Col>
      </Grid>

      {/* График кликов по времени */}
      <Card withBorder radius="md" p="lg">
        <Text fw={600} mb="md">Клики по времени</Text>
        <ResponsiveContainer width="100%" height={260}>
          <LineChart data={data.clicksOverTime}>
            <CartesianGrid strokeDasharray="3 3" stroke="#373A40" />
            <XAxis dataKey="date" tick={{ fontSize: 12 }} />
            <YAxis tick={{ fontSize: 12 }} />
            <Tooltip />
            <Line
              type="monotone" dataKey="clicks"
              stroke="#4dabf7" strokeWidth={2} dot={false}
            />
          </LineChart>
        </ResponsiveContainer>
      </Card>

      {/* Топ тегов */}
      {data.topTags.length > 0 && (
        <Card withBorder radius="md" p="lg">
          <Text fw={600} mb="md">Распределение по тегам</Text>
          <ResponsiveContainer width="100%" height={220}>
            <PieChart>
              <Pie
                data={data.topTags}
                dataKey="count"
                nameKey="tag"
                cx="50%" cy="50%"
                outerRadius={80}
                label={({ tag, percent }: { tag: string; percent: number }) =>
                  `${tag} ${(percent * 100).toFixed(0)}%`
                }
              >
                {data.topTags.map((_, i) => (
                  <Cell key={i} fill={COLORS[i % COLORS.length]} />
                ))}
              </Pie>
              <Legend />
              <Tooltip />
            </PieChart>
          </ResponsiveContainer>
        </Card>
      )}
    </Stack>
  );
}

function KPICard({
  label, value, icon, color,
}: {
  label: string; value: string; icon: React.ReactNode; color: string;
}) {
  return (
    <Card withBorder radius="md" p="lg">
      <Group justify="space-between">
        <Stack gap={4}>
          <Text size="sm" c="dimmed">{label}</Text>
          <Text size="xl" fw={700}>{value}</Text>
        </Stack>
        <Badge size="xl" variant="light" color={color} p="xs">
          {icon}
        </Badge>
      </Group>
    </Card>
  );
}
