import { useEffect, useState } from 'react';
import {
  Stack, Title, Card, Text, Switch, NumberInput,
  Button, Group, Badge, Divider, Skeleton, Center,
} from '@mantine/core';
import { notifications } from '@mantine/notifications';
import { api } from '../../api/client';
import { RU } from '../../i18n/ru';

interface ShlinkSettings {
  shortCodeLength:  number;
  allowCustomSlugs: boolean;
  userSlugPrefix:   boolean;
  domain:           string;
  shlinkVersion:    string;
  connected:        boolean;
}

export function Settings() {
  const [settings, setSettings] = useState<ShlinkSettings | null>(null);
  const [loading,  setLoading]  = useState(true);
  const [saving,   setSaving]   = useState(false);

  // Локальное состояние редактируемых полей
  const [codeLen,       setCodeLen]       = useState<number>(6);
  const [allowSlugs,    setAllowSlugs]    = useState(true);
  const [slugPrefix,    setSlugPrefix]    = useState(false);

  useEffect(() => {
    api.get<ShlinkSettings>('/api/admin/settings')
      .then(s => {
        setSettings(s);
        setCodeLen(s.shortCodeLength);
        setAllowSlugs(s.allowCustomSlugs);
        setSlugPrefix(s.userSlugPrefix);
      })
      .catch(() => setSettings(null))
      .finally(() => setLoading(false));
  }, []);

  const handleSave = async () => {
    setSaving(true);
    try {
      await api.patch('/api/admin/settings', {
        shortCodeLength:  codeLen,
        allowCustomSlugs: allowSlugs,
        userSlugPrefix:   slugPrefix,
      });
      notifications.show({ message: RU.settings.saved, color: 'green' });
      // Обновляем локальное состояние
      setSettings(prev => prev ? {
        ...prev, shortCodeLength: codeLen, allowCustomSlugs: allowSlugs, userSlugPrefix: slugPrefix,
      } : prev);
    } catch {
      // handled
    } finally {
      setSaving(false);
    }
  };

  if (loading) {
    return (
      <Stack gap="lg">
        <Skeleton height={32} width={200} />
        {[1, 2, 3].map(i => <Skeleton key={i} height={120} radius="md" />)}
      </Stack>
    );
  }

  if (!settings) {
    return <Center h={200}><Text c="dimmed">Не удалось загрузить настройки</Text></Center>;
  }

  return (
    <Stack gap="lg">
      <Title order={2}>{RU.settings.title}</Title>

      {/* Блок: Генерация ссылок */}
      <Card withBorder radius="md" p="lg">
        <Text fw={600} mb="md">{RU.settings.generation}</Text>
        <Stack gap="md">
          <NumberInput
            label={RU.settings.shortCodeLength}
            description={RU.settings.shortCodeHint}
            min={4} max={10}
            value={codeLen}
            onChange={v => setCodeLen(Number(v))}
            style={{ maxWidth: 200 }}
          />
          <Switch
            label={RU.settings.allowCustomSlugs}
            checked={allowSlugs}
            onChange={e => setAllowSlugs(e.currentTarget.checked)}
          />
          <Switch
            label={RU.settings.userSlugPrefix}
            description="FEATURE_USER_SLUG_PREFIX"
            checked={slugPrefix}
            onChange={e => setSlugPrefix(e.currentTarget.checked)}
          />
        </Stack>
      </Card>

      {/* Блок: Домен */}
      <Card withBorder radius="md" p="lg">
        <Text fw={600} mb="md">{RU.settings.domain}</Text>
        <Group gap="sm" align="center">
          <Text size="sm" c="dimmed">{RU.settings.domainHint}:</Text>
          <Text ff="monospace" fw={500}>{settings.domain || '—'}</Text>
        </Group>
      </Card>

      {/* Блок: API */}
      <Card withBorder radius="md" p="lg">
        <Text fw={600} mb="md">{RU.settings.apiSection}</Text>
        <Stack gap="xs">
          <Group gap="sm">
            <Text size="sm" c="dimmed" w={180}>{RU.settings.shlinkVersion}:</Text>
            <Text ff="monospace" size="sm">{settings.shlinkVersion || '—'}</Text>
          </Group>
          <Divider />
          <Group gap="sm">
            <Text size="sm" c="dimmed" w={180}>{RU.settings.connectionStatus}:</Text>
            <Badge
              color={settings.connected ? 'green' : 'red'}
              variant="dot"
            >
              {settings.connected ? RU.settings.connected : RU.settings.disconnected}
            </Badge>
          </Group>
        </Stack>
      </Card>

      <Group>
        <Button onClick={handleSave} loading={saving}>{RU.save}</Button>
      </Group>
    </Stack>
  );
}
