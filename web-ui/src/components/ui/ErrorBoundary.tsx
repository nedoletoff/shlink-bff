import React from 'react';
import { Center, Stack, Text, Button } from '@mantine/core';
import { IconAlertTriangle } from '@tabler/icons-react';

interface State {
  hasError: boolean;
  message:  string;
}

interface Props {
  children:    React.ReactNode;
  /** Название секции для сообщения об ошибке */
  section?:    string;
}

/**
 * React Error Boundary — оборачивает каждую вкладку / секцию.
 * Падавший компонент не крашит всю страницу.
 */
export class ErrorBoundary extends React.Component<Props, State> {
  constructor(props: Props) {
    super(props);
    this.state = { hasError: false, message: '' };
  }

  static getDerivedStateFromError(error: unknown): State {
    const message = error instanceof Error ? error.message : 'Неизвестная ошибка';
    return { hasError: true, message };
  }

  componentDidCatch(error: unknown, info: React.ErrorInfo) {
    console.error('[ErrorBoundary]', this.props.section ?? 'unknown', error, info);
  }

  handleReset = () => {
    this.setState({ hasError: false, message: '' });
  };

  render() {
    if (this.state.hasError) {
      return (
        <Center h={200}>
          <Stack align="center" gap="sm">
            <IconAlertTriangle size={32} color="var(--mantine-color-red-6)" />
            <Text c="red" fw={500}>
              {this.props.section
                ? `Ошибка в секции «${this.props.section}»`
                : 'Произошла ошибка'}
            </Text>
            <Text size="sm" c="dimmed">{this.state.message}</Text>
            <Button size="xs" variant="light" onClick={this.handleReset}>
              Повторить
            </Button>
          </Stack>
        </Center>
      );
    }
    return this.props.children;
  }
}
