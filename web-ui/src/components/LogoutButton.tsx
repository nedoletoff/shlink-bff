import { FC } from 'react'

interface Props {
  username?: string
}

/**
 * LogoutButton — выход через oauth2-proxy /oauth2/sign_out.
 *
 * oauth2-proxy при запросе на /oauth2/sign_out:
 *   1. Удаляет сессионный cookie.
 *   2. Редиректит на Keycloak end_session (если настроен OIDC logout).
 *   3. После выхода редиректит на rd-адрес (должен быть в whitelist_domains).
 *
 * Настройка rd в docker-compose: убедись, что shlink-create.local
 * находится в whitelist_domains в oauth2-proxy/shlink.cfg.
 */
const LogoutButton: FC<Props> = ({ username }) => {
  const handleLogout = () => {
    window.location.href = '/oauth2/sign_out?rd=%2F'
  }

  return (
    <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
      {username && (
        <span style={{ fontSize: 13, color: 'var(--color-text-muted, #666)' }}>
          {username}
        </span>
      )}
      <button
        onClick={handleLogout}
        title="Выйти из аккаунта"
        style={{
          padding: '4px 12px',
          fontSize: 13,
          cursor: 'pointer',
          border: '1px solid currentColor',
          borderRadius: 4,
          background: 'transparent',
        }}
      >
        Выйти
      </button>
    </div>
  )
}

export default LogoutButton
