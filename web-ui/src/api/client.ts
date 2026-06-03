import { notifications } from '@mantine/notifications';

// Все запросы идут на относительный /api/* — браузер никогда не знает Shlink API key.
// В prod: nginx → unified-backend.
// В dev: vite proxy → localhost:8080.

export class APIError extends Error {
  constructor(
    public readonly status: number,
    message: string,
  ) {
    super(message);
    this.name = 'APIError';
  }
}

export interface RequestOptions extends RequestInit {
  params?: Record<string, string | number | boolean | undefined>;
}

async function request<T>(path: string, options: RequestOptions = {}): Promise<T> {
  const { params, ...init } = options;

  let url = path;
  if (params) {
    const sp = new URLSearchParams();
    for (const [k, v] of Object.entries(params)) {
      if (v !== undefined && v !== '') sp.set(k, String(v));
    }
    const qs = sp.toString();
    if (qs) url = `${path}?${qs}`;
  }

  const resp = await fetch(url, {
    credentials: 'include', // cookie-based session через oauth2-proxy
    headers: {
      'Content-Type': 'application/json',
      ...init.headers,
    },
    ...init,
  });

  if (resp.status === 401) {
    // Сессия истекла — редирект через oauth2-proxy start (не sign_in напрямую)
    // Используем replace чтобы не ломать history
    window.location.replace('/oauth2/start?rd=' + encodeURIComponent(window.location.pathname));
    throw new APIError(401, 'Unauthorized — redirecting to login');
  }

  if (resp.status === 403) {
    const body = await resp.json().catch(() => ({ error: 'Forbidden' }));
    const msg = (body as { error?: string })?.error ?? 'Forbidden';
    // НЕ редиректим при 403 — выбрасываем ошибку, AuthContext поймает
    throw new APIError(403, msg);
  }

  if (!resp.ok) {
    const body = await resp.json().catch(() => ({ error: `HTTP ${resp.status}` }));
    const msg = (body as { error?: string })?.error ?? `HTTP ${resp.status}`;
    notifications.show({ title: 'Ошибка', message: msg, color: 'red' });
    throw new APIError(resp.status, msg);
  }

  if (resp.status === 204) return undefined as T;

  return resp.json() as Promise<T>;
}

export const api = {
  get: <T>(path: string, options?: RequestOptions) =>
    request<T>(path, { method: 'GET', ...options }),

  post: <T>(path: string, body: unknown, options?: RequestOptions) =>
    request<T>(path, { method: 'POST', body: JSON.stringify(body), ...options }),

  patch: <T>(path: string, body: unknown, options?: RequestOptions) =>
    request<T>(path, { method: 'PATCH', body: JSON.stringify(body), ...options }),

  put: <T>(path: string, body: unknown, options?: RequestOptions) =>
    request<T>(path, { method: 'PUT', body: JSON.stringify(body), ...options }),

  delete: <T>(path: string, options?: RequestOptions) =>
    request<T>(path, { method: 'DELETE', ...options }),
};
