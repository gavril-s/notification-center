import { useEffect } from 'react'
import { useAuth } from '../contexts/AuthContext'
import { useNavigate } from 'react-router-dom'
import './RecipientProfile.css'

export default function RecipientProfile() {
  const { user, isLoading } = useAuth()
  const navigate = useNavigate()

  useEffect(() => {
    if (!isLoading && !user) {
      navigate('/login')
    }
  }, [user, isLoading, navigate])

  if (isLoading || !user) {
    return (
      <div className="loading-screen">
        <div className="spinner"></div>
        <p>Загрузка...</p>
      </div>
    )
  }

  const roleLabels: Record<string, string> = {
    recipient_user: 'Получатель',
    source_operator: 'Оператор',
    admin: 'Администратор',
  }

  return (
    <div className="profile-page">
      <div className="card">
        <div className="card-header">
          <h2 className="card-title">Профиль пользователя</h2>
        </div>

        <div className="profile-info">
          <div className="profile-field">
            <label className="profile-label">Логин</label>
            <p className="profile-value">{user?.login}</p>
          </div>

          <div className="profile-field">
            <label className="profile-label">Роль</label>
            <p className="profile-value">{roleLabels[user?.role || ''] || user?.role}</p>
          </div>

          <div className="profile-field">
            <label className="profile-label">ID пользователя</label>
            <p className="profile-value profile-value-mono">{user?.user_id}</p>
          </div>
        </div>

        <div className="profile-section">
          <h3 className="profile-section-title">Навигация</h3>
          <div className="profile-nav">
            <a href="/app/contacts" className="profile-nav-item">
              <span className="profile-nav-icon">📱</span>
              <div>
                <strong>Мои контакты</strong>
                <p>Управление email, телефоном, Telegram</p>
              </div>
            </a>
            <a href="/app/preferences" className="profile-nav-item">
              <span className="profile-nav-icon">⚙️</span>
              <div>
                <strong>Настройки уведомлений</strong>
                <p>Тихие часы, блокировка каналов</p>
              </div>
            </a>
            <a href="/app/history" className="profile-nav-item">
              <span className="profile-nav-icon">📋</span>
              <div>
                <strong>История уведомлений</strong>
                <p>Просмотр отправленных уведомлений</p>
              </div>
            </a>
          </div>
        </div>
      </div>
    </div>
  )
}