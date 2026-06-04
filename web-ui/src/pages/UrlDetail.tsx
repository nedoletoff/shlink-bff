import { useEffect, useState } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import {
  Stack, Title, Text, Card, Group, Badge, Anchor,
  ActionIcon, Tooltip, SegmentedControl, Table,
  Skeleton, Center, Grid, CopyButton, Pagination,
} from '@mantine/core';
import {
  IconArrowLeft, IconCopy, IconCheck,
  IconEdit, IconTrash,
} from '@tabler/icons-react';
import {
  LineChart, Line, XAxis, YAxis, CartesianGrid,
  Tooltip as RTooltip, ResponsiveContainer,
  PieChart, Pie, Cell, Legend,
} from 'recharts';
import { api } from '../api/client';
import { useIsAdmin } from '../contexts/AuthContext';
import { ErrorBoundary } from '../components/ui/ErrorBoundary';
import { formatDate, formatDateTime } from '../utils/date';
import type { UrlDetailResponse } from '../types/api';

const COLORS  = ['#4dabf7', '#51cf66', '#ff6b6b', '#ffd43b', '#cc5de8'];
const PERIODS = [
  { label: '7 д',  value: '7'  },
  { label: '30 д', value: '30' },
  { label: '90 д', value: '90' },
];
const VISITS_PER_PAGE = 20;

export function UrlDetail() {
  const { shortCode } = useParams<{ shortCode: string }>();
  const navigate      = useNavigate();
  const isAdmin       = useIsAdmin();

  const [data,    setData]    = useState<UrlDetailResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [err,     setErr]     = useState<string | null>(null);
  const [period,  setPeriod]  = useState('30');
  const [vPage,   setVPage]   = useState(1);

  useEffect(() => {
    if (!shortCode) return;
    setLoading(true); setErr(null);
    api.get<UrlDetailResponse>(`/api/shlink/short-urls/${shortCode}/stats`, {
      params: { period },
    })
      .then(setData)
      .catch((e: unknown) => setErr(e instanceof Error ? e.message : 'Ошибка загрузки'))
      .finally(() => setLoading(false));
  }, [shortCode, period]);

  if (loading) return (
    <Stack gap="lg">
      <Skeleton height={32} width={300} />
      <Skeleton height={48} />
      <Skeleton height={260} radius="md" />
      <Grid><Grid.Col span={4}><Skeleton height={200} radius="md" /></Grid.Col>
        <Grid.Col span={4}><Skeleton height={200} radius="md" /></Grid.Col>
        <Grid.Col span={4}><Skeleton height={200} radius="md" /></Grid.Col>
      </Grid>
      <Skeleton height={300} radius="md" />
    </Stack>
  );

  if (err) return (
    <Center h={300}>
      <Text c="red">{err}</Text>
    </Center>
  );

  if (!data) return null;

  const deviceData = [
    { name: 'Desktop', value: data.devices.desktop },
    { name: 'Mobile',  value: data.devices.mobile  },
    { name: 'Tablet',  value: data.devices.tablet  },
  ].filter(d => d.value > 0);

  const visitsPage = data.visits.slice(
    (vPage - 1) * VISITS_PER_PAGE,
    vPage * VISITS_PER_PAGE,
  );
  const totalVPages = Math.max(1, Math.ceil(data.visits.length / VISITS_PER_PAGE));

  return (
    <Stack gap="lg">
      {/* Заголовок */}
      <Group>
        <ActionIcon variant="subtle" onClick={() => navigate(-1)}>
          <IconArrowLeft size={18} />
        </ActionIcon>
        <Title order={2}>{data.title || data.shortCode}</Title>
        <SegmentedControl
          data={PERIODS}
          value={period}
          onChange={v => { setPeriod(v); setVPage(1); }}
          size="xs"
          ml="auto"
        />
      </Group>

      {/* Инфо-строка */}
      <Card withBorder radius="md" p="md">
        <Group wrap="wrap" gap="lg">
          <Stack gap={2}>
            <Text size="xs" c="dimmed">Короткая ссылка</Text>
            <Group gap={6}>
              <Anchor href={data.shortUrl} target="_blank" size="sm" fw={500}>
                {data.shortUrl}
              </Anchor>
              <CopyButton value={data.shortUrl} timeout={2000}>
                {({ copied, copy }) => (
                  <Tooltip label={copied ? 'Скопировано!' : 'Копировать'} withArrow>
                    <ActionIcon size="xs" variant="subtle"
                      color={copied ? 'teal' : 'gray'} onClick={copy}>
                      {copied ? <IconCheck size={12} /> : <IconCopy size={12} />}
                    </ActionIcon>
                  </Tooltip>
                )}
              </CopyButton>
            </Group>
          </Stack>

          <Stack gap={2}>
            <Text size="xs" c="dimmed">Куда ведёт</Text>
            <Text size="sm" truncate="end" maw={300} title={data.longUrl}>{data.longUrl}</Text>
          </Stack>

          <Stack gap={2}>
            <Text size="xs" c="dimmed">Создана</Text>
            <Text size="sm">{formatDate(data.dateCreated)}</Text>
          </Stack>

          {isAdmin && data.ownerUsername && (
            <Stack gap={2}>
              <Text size="xs" c="dimmed">Владелец</Text>
              <Text size="sm">{data.ownerUsername}</Text>
            </Stack>
          )}

          <Group ml="auto" gap={8}>
            <Tooltip label="Редактировать" withArrow>
              <ActionIcon variant="subtle">
                <IconEdit size={16} />
              </ActionIcon>
            </Tooltip>
            <Tooltip label="Удалить" withArrow>
              <ActionIcon variant="subtle" color="red">
                <IconTrash size={16} />
              </ActionIcon>
            </Tooltip>
          </Group>
        </Group>
      </Card>

      {/* График переходов по дням */}
      <ErrorBoundary section="График">
        <Card withBorder radius="md" p="lg">
          <Text fw={600} mb="md">Переходы по дням</Text>
          <ResponsiveContainer width="100%" height={240}>
            <LineChart data={data.clicksPerDay}>
              <CartesianGrid strokeDasharray="3 3" stroke="#373A40" />
              <XAxis dataKey="date" tick={{ fontSize: 11 }} />
              <YAxis tick={{ fontSize: 11 }} />
              <RTooltip />
              <Line type="monotone" dataKey="clicks"
                stroke="#4dabf7" strokeWidth={2} dot={false} />
            </LineChart>
          </ResponsiveContainer>
        </Card>
      </ErrorBoundary>

      {/* Donut-диаграммы */}
      <ErrorBoundary section="Диаграммы">
        <Grid>
          <Grid.Col span={{ base: 12, md: 4 }}>
            <DonutCard title="Устройства" data={deviceData} />
          </Grid.Col>
          <Grid.Col span={{ base: 12, md: 4 }}>
            <DonutCard title="Браузеры"
              data={data.browsers.map(b => ({ name: b.name, value: b.count }))} />
          </Grid.Col>
          <Grid.Col span={{ base: 12, md: 4 }}>
            <DonutCard title="Операционные системы"
              data={data.os.map(o => ({ name: o.name, value: o.count }))} />
          </Grid.Col>
        </Grid>
      </ErrorBoundary>

      {/* Таблица переходов */}
      <ErrorBoundary section="Таблица переходов">
        <Card withBorder radius="md" p={0}>
          <Group px="lg" py="md" justify="space-between">
            <Text fw={600}>Переходы</Text>
            <Text size="sm" c="dimmed">Всего: {data.visitsTotal.toLocaleString('ru-RU')}</Text>
          </Group>
          <Table striped highlightOnHover>
            <Table.Thead>
              <Table.Tr>
                <Table.Th>Дата/время</Table.Th>
                <Table.Th>Устройство</Table.Th>
                <Table.Th>OS</Table.Th>
                <Table.Th>Страна</Table.Th>
                <Table.Th>Реферер</Table.Th>
              </Table.Tr>
            </Table.Thead>
            <Table.Tbody>
              {visitsPage.map((v, i) => (
                <Table.Tr key={i}>
                  <Table.Td><Text size="xs" ff="monospace">{formatDateTime(v.date)}</Text></Table.Td>
                  <Table.Td><Text size="sm">{v.device || '—'}</Text></Table.Td>
                  <Table.Td><Text size="sm">{v.os || '—'}</Text></Table.Td>
                  <Table.Td><Text size="sm">{v.country || '—'}</Text></Table.Td>
                  <Table.Td>
                    <Text size="xs" c="dimmed" truncate="end" maw={160}>{v.referer || '—'}</Text>
                  </Table.Td>
                </Table.Tr>
              ))}
              {data.visits.length === 0 && (
                <Table.Tr>
                  <Table.Td colSpan={5}>
                    <Center p="xl"><Text c="dimmed">Переходов нет</Text></Center>
                  </Table.Td>
                </Table.Tr>
              )}
            </Table.Tbody>
          </Table>
          {totalVPages > 1 && (
            <Group justify="flex-end" p="md">
              <Pagination total={totalVPages} value={vPage} onChange={setVPage} size="sm" />
            </Group>
          )}
        </Card>
      </ErrorBoundary>
    </Stack>
  );
}

function DonutCard({ title, data }: { title: string; data: { name: string; value: number }[] }) {
  return (
    <Card withBorder radius="md" p="lg">
      <Text fw={600} mb="md">{title}</Text>
      <ResponsiveContainer width="100%" height={180}>
        <PieChart>
          <Pie data={data} dataKey="value" nameKey="name"
            cx="50%" cy="50%" innerRadius={45} outerRadius={70}
          >
            {data.map((_, i) => <Cell key={i} fill={COLORS[i % COLORS.length]} />)}
          </Pie>
          <Legend />
          <RTooltip formatter={(v: number) => v.toLocaleString('ru-RU')} />
        </PieChart>
      </ResponsiveContainer>
    </Card>
  );
}
