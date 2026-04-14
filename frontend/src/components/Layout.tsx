import { NavLink, Outlet, useNavigate } from 'react-router-dom'
import { useAuth } from '../contexts/AuthContext'
import './Layout.css'

export default function Layout({ isOperator = false }: { isOperator?: boolean }) {
  const { user, logout } = useAuth()
  const navigate = useNavigate()

  const handleLogout = () => {
    logout()
    navigate('/login')
  }

  const recipientLinks = [
    { to: '/app/profile', label: 'Профиль' },
    { to: '/app/contacts', label: 'Контакты' },
    { to: '/app/preferences', label: 'Настройки' },
    { to: '/app/history', label: 'История' },
  ]

  const operatorLinks = [
    { to: '/operator/templates', label: 'Шаблоны' },
    { to: '/operator/groups', label: 'Группы' },
    { to: '/operator/campaigns', label: 'Рассылки' },
    { to: '/operator/history', label: 'История' },
    { to: '/operator/analytics', label: 'Аналитика' },
  ]

  const links = isOperator ? operatorLinks : recipientLinks

  return (
    <div className="layout">
      <header className="header">
        <div className="header-content">
          <h1 className="logo">Центр уведомлений</h1>
          <nav className="nav">
            {links.map(link => (
              <NavLink
                key={link.to}
                to={link.to}
                className={({ isActive }) => `nav-link ${isActive ? 'active' : ''}`}
              >
                {link.label}
              </NavLink>
            ))}
          </nav>
          <div className="user-menu">
            <span className="user-name">{user?.login}</span>
            {!isOperator && user?.role === 'source_operator' && (
              <NavLink to="/operator/templates" className="btn btn-secondary btn-sm">
                Оператор
              </NavLink>
            )}
            <button onClick={handleLogout} className="btn btn-secondary btn-sm">
              Выйти
            </button>
          </div>
        </div>
      </header>
      <main className="main">
        <div className="container">
          <Outlet />
        </div>
      </main>
    </div>
  )
}