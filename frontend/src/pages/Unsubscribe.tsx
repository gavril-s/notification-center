import { useState, useEffect } from 'react'
import { useSearchParams, useNavigate } from 'react-router-dom'
import { recipientsApi, getErrorMessage } from '../api/client'
import './Unsubscribe.css'

export default function Unsubscribe() {
  const [searchParams] = useSearchParams()
  const navigate = useNavigate()
  const [token, setToken] = useState<string>('')
  const [availableScopes, setAvailableScopes] = useState<string[]>([])
  const [selectedScope, setSelectedScope] = useState<string>('')
  const [isLoading, setIsLoading] = useState(false)
  const [error, setError] = useState('')
  const [success, setSuccess] = useState(false)

  useEffect(() => {
    const tokenParam = searchParams.get('token')
    const scopesParam = searchParams.get('available_scopes')
    
    if (!tokenParam) {
      setError('Отсутствует токен отписки')
      return
    }

    setToken(tokenParam)

    if (scopesParam) {
      try {
        const scopes = JSON.parse(scopesParam)
        setAvailableScopes(scopes)
        if (scopes.length === 1) {
          setSelectedScope(scopes[0])
        }
      } catch {
        setAvailableScopes(['sender'])
      }
    } else {
      setAvailableScopes(['sender'])
    }
  }, [searchParams])

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setError('')
    setIsLoading(true)

    try {
      await recipientsApi.unsubscribe({
        token,
        selected_scope: selectedScope as 'sender' | 'campaign_or_group',
      })
      setSuccess(true)
    } catch (err) {
      setError(getErrorMessage(err))
    } finally {
      setIsLoading(false)
    }
  }

  const scopeLabels: Record<string, string> = {
    sender: 'Отписаться от всех писем от этого отправителя',
    campaign_or_group: 'Отписаться от конкретной рассылки',
  }

  if (success) {
    return (
      <div className="unsubscribe-page">
        <div className="unsubscribe-card">
          <div className="unsubscribe-success">
            <div className="success-icon">✓</div>
            <h1>Вы успешно отписались</h1>
            <p>Вы больше не будете получать уведомления от этого отправителя.</p>
            <button onClick={() => navigate('/login')} className="btn btn-primary">
              Перейти на главную
            </button>
          </div>
        </div>
      </div>
    )
  }

  if (error && !token) {
    return (
      <div className="unsubscribe-page">
        <div className="unsubscribe-card">
          <div className="unsubscribe-error">
            <div className="error-icon">✕</div>
            <h1>Ошибка</h1>
            <p>{error}</p>
          </div>
        </div>
      </div>
    )
  }

  return (
    <div className="unsubscribe-page">
      <div className="unsubscribe-card">
        <h1>Отписка от уведомлений</h1>
        <p className="unsubscribe-subtitle">
          Вы хотите отписаться от получения уведомлений. Пожалуйста, выберите тип отписки:
        </p>

        {error && <div className="alert alert-error">{error}</div>}

        <form onSubmit={handleSubmit} className="unsubscribe-form">
          <div className="scope-options">
            {availableScopes.map(scope => (
              <label key={scope} className="scope-option">
                <input
                  type="radio"
                  name="scope"
                  value={scope}
                  checked={selectedScope === scope}
                  onChange={e => setSelectedScope(e.target.value)}
                />
                <span className="scope-label">{scopeLabels[scope] || scope}</span>
              </label>
            ))}
          </div>

          <button type="submit" className="btn btn-primary btn-full" disabled={isLoading || !selectedScope}>
            {isLoading ? 'Подождите...' : 'Отписаться'}
          </button>
        </form>

        <div className="unsubscribe-footer">
          <p>
            Если вы хотите изменить другие настройки уведомлений,{' '}
            <a href="/login">войдите в систему</a>
          </p>
        </div>
      </div>
    </div>
  )
}