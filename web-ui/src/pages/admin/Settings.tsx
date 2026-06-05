import { useEffect, useState } from 'react';
import {
  Stack, Title, Card, Text, Switch, NumberInput,
  Button, Group, Badge, Divider, Skeleton, Center, TextInput,
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

  const [codeLen,    setCodeLen]    = useState<number>(6);
  const [allowSlugs, setAllowSlugs] = useState(true);
  const [slugPrefix, setSlugPrefix] = useState(false);
  const [domain,     setDomain]     = useState('');

  useEffect(() => {
    api.get<ShlinkSettings>('/api/admin/settings')
      .then(s => {
        setSettings(s);
        setCodeLen(s.shortCodeLength);
        setAllowSlugs(s.allowCustomSlugs);
        setSlugPrefix(s.userSlugPrefix);
        setDomain(s.domain ?? '');
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
        domain:           domain.trim() || undefined,
      });
      notifications.show({ message: RU.settings.saved, color: 'green' });
      setSettings(prev => prev ? {
        ...prev,
        shortCodeLength: codeLen,
        allowCustomSlugs: allowSlugs,
        userSlugPrefix: slugPrefix,
        domain,
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

      <Card withBorder radius="md" p="lg">
        <Text fw={600} mb="md">{RU.settings.generation}</Text>
        <Stack gap="md">
          <NumberInput
            label={RU.settings.shortCodeLength}
            description={`${RU.settings.shortCodeHint} (3–32)`}
            min={3}
            max={32}
            value={codeLen}
            onChange={v => setCodeLen(Number(v))}
            style={{ maxWidth: 220 }}
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

      <Card withBorder radius="md" p="lg">
        <Text fw={600} mb="md">{RU.settings.domain}</Text>
        <Stack gap="sm">
          <TextInput
            label={RU.settings.domainHint}
            placeholder="https://s.example.com"
            value={domain}
            onChange={e => setDomain(e.currentTarget.value)}
          />
          <Text size="xs" c="dimmed">
            Этот домен используется в начале коротких ссылок вместо внутреннего docker-адреса.
          </Text>
        </Stack>
      </Card>

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
            <Badge color={settings.connected ? 'green' : 'red'} variant="dot">
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
