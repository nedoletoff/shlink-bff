import { useCallback, useEffect, useState } from 'react';
import {
  Stack, Title, Text, Card, Group, Button, TextInput,
  Switch, NumberInput, Loader, Center, Badge, Divider,
  Alert, Box,
} from '@mantine/core';
import { notifications } from '@mantine/notifications';
import { IconInfoCircle, IconPlugConnected, IconPlugConnectedX } from '@tabler/icons-react';
import { api } from '../../api/client';
import type { ShlinkSettings } from '../../types/api';

interface SettingsForm {
  shortCodeLength:  number;
  allowCustomSlugs: boolean;
  userSlugPrefix:   boolean;
  domain:           string;
}

function FieldRow({ label, description, children }: {
  label: string;
  description?: string;
  children: React.ReactNode;
}) {
  return (
    <Group justify="space-between" align="flex-start" wrap="nowrap" gap="xl">
      <Box style={{ flex: 1, minWidth: 0 }}>
        <Text size="sm" fw={500}>{label}</Text>
        {description && <Text size="xs" c="dimmed" mt={2}>{description}</Text>}
      </Box>
      <Box style={{ flexShrink: 0, minWidth: 180 }}>{children}</Box>
    </Group>
  );
}

export function Settings() {
  const [settings, setSettings] = useState<ShlinkSettings | null>(null);
  const [form,     setForm]     = useState<SettingsForm | null>(null);
  const [loading,  setLoading]  = useState(true);
  const [saving,   setSaving]   = useState(false);
  const [dirty,    setDirty]    = useState(false);

  const fetchSettings = useCallback(() => {
    setLoading(true);
    api.get<ShlinkSettings>('/api/admin/settings')
      .then(s => {
        setSettings(s);
        setForm({
          shortCodeLength:  s.shortCodeLength,
          allowCustomSlugs: s.allowCustomSlugs,
          userSlugPrefix:   s.userSlugPrefix,
          domain:           s.domain,
        });
        setDirty(false);
      })
      .catch(() => setSettings(null))
      .finally(() => setLoading(false));
  }, []);

  useEffect(() => { fetchSettings(); }, [fetchSettings]);

  const patch = <K extends keyof SettingsForm>(key: K, val: SettingsForm[K]) => {
    setForm(f => f ? { ...f, [key]: val } : f);
    setDirty(true);
  };

  const handleSave = async () => {
    if (!form) return;
    setSaving(true);
    try {
      await api.patch('/api/admin/settings', form);
      notifications.show({ message: 'Настройки сохранены', color: 'green' });
      fetchSettings();
    } catch {
      /* shown */
    } finally {
      setSaving(false);
    }
  };

  return (
    <Stack gap="xl">
      <Group justify="space-between" wrap="nowrap">
        <div>
          <Title order={2} fw={700}>Настройки</Title>
          <Text size="sm" c="dimmed">Конфигурация сервиса</Text>
        </div>
        {settings && (
          <Badge
            size="sm"
            variant="light"
            color={settings.connected ? 'green' : 'red'}
            leftSection={settings.connected
              ? <IconPlugConnected size={12} />
              : <IconPlugConnectedX size={12} />
            }
          >
            Shlink {settings.shlinkVersion} — {settings.connected ? 'подключён' : 'недоступен'}
          </Badge>
        )}
      </Group>

      {loading ? (
        <Center h={200}><Loader /></Center>
      ) : !form ? (
        <Alert icon={<IconInfoCircle size={16} />} color="red">
          Не удалось загрузить настройки. Проверьте подключение к Shlink.
        </Alert>
      ) : (
        <Stack gap="md">
          {/* Shlink info */}
          <Card withBorder radius="md" p="lg">
            <Text fw={600} mb="md" size="sm" tt="uppercase" c="dimmed" style={{ letterSpacing: '0.05em' }}>
              Подключение
            </Text>
            <Stack gap="md">
              <FieldRow label="Домен" description="Базовый домен, используемый для коротких ссылок">
                <TextInput
                  size="sm"
                  value={form.domain}
                  onChange={e => patch('domain', e.currentTarget.value)}
                  placeholder="https://s.example.com"
                />
              </FieldRow>
            </Stack>
          </Card>

          {/* Short URL settings */}
          <Card withBorder radius="md" p="lg">
            <Text fw={600} mb="md" size="sm" tt="uppercase" c="dimmed" style={{ letterSpacing: '0.05em' }}>
              Короткие ссылки
            </Text>
            <Stack gap="lg">
              <FieldRow
                label="Длина кода"
                description="Количество символов в автоматически сгенерированном коде (4–10)"
              >
                <NumberInput
                  size="sm"
                  value={form.shortCodeLength}
                  onChange={v => patch('shortCodeLength', Number(v) || 5)}
                  min={4} max={10}
                />
              </FieldRow>

              <Divider />

              <FieldRow
                label="Кастомные слаги"
                description="Разрешить пользователям задавать собственный код ссылки"
              >
                <Switch
                  checked={form.allowCustomSlugs}
                  onChange={e => patch('allowCustomSlugs', e.currentTarget.checked)}
                  label={form.allowCustomSlugs ? 'Включено' : 'Выключено'}
                />
              </FieldRow>

              <Divider />

              <FieldRow
                label="Префикс пользователя"
                description="Добавлять slug-префикс пользователя к генерируемым кодам"
              >
                <Switch
                  checked={form.userSlugPrefix}
                  onChange={e => patch('userSlugPrefix', e.currentTarget.checked)}
                  label={form.userSlugPrefix ? 'Включено' : 'Выключено'}
                />
              </FieldRow>
            </Stack>
          </Card>

          {dirty && (
            <Group justify="flex-end">
              <Button variant="default" onClick={fetchSettings}>Сбросить</Button>
              <Button onClick={handleSave} loading={saving}>Сохранить изменения</Button>
            </Group>
          )}
        </Stack>
      )}
    </Stack>
  );
}
