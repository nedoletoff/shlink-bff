import { FC } from 'react'

interface Props {
  username?: string
}

/**
 * LogoutButton — выход через oauth2-proxy /oauth2/sign_out.
 *
 * oauth2-proxy требует POST на /oauth2/sign_out для инвалидации cookie.
 * GET-редирект не удаляет сессию — используем скрытую форму.
 */
const LogoutButton: FC<Props> = ({ username }) => {
  const handleLogout = () => {
    const form = document.createElement('form')
    form.method = 'POST'
    form.action = '/oauth2/sign_out'

    const rd = document.createElement('input')
    rd.type = 'hidden'
    rd.name = 'rd'
    rd.value = '/oauth2/sign_in'
    form.appendChild(rd)

    document.body.appendChild(form)
    form.submit()
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
