import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { recipientsApi, getErrorMessage } from '../api/client'
import { useAuth } from '../contexts/AuthContext'
import './Login.css'

export default function Login() {
  const [isRegister, setIsRegister] = useState(false)
  const [login, setLogin] = useState('')
  const [password, setPassword] = useState('')
  const [confirmPassword, setConfirmPassword] = useState('')
  const [error, setError] = useState('')
  const [isLoading, setIsLoading] = useState(false)
  const { login: authLogin } = useAuth()
  const navigate = useNavigate()

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setError('')

    if (isRegister && password !== confirmPassword) {
      setError('Пароли не совпадают')
      return
    }

    setIsLoading(true)

    try {
      if (isRegister) {
        const response = await recipientsApi.register({ login, password })
        await authLogin(response.data.access_token, response.data.refresh_token)
        navigate('/app/profile')
      } else {
        const response = await recipientsApi.login({ login, password })
        await authLogin(response.data.access_token, response.data.refresh_token)
        
        const userResponse = await recipientsApi.getMe()
        if (userResponse.data.role === 'source_operator' || userResponse.data.role === 'admin') {
          navigate('/operator/templates')
        } else {
          navigate('/app/profile')
        }
      }
    } catch (err) {
      setError(getErrorMessage(err))
    } finally {
      setIsLoading(false)
    }
  }

  return (
    <div className="login-page">
      <div className="login-card">
        <div className="login-header">
          <h1>Центр уведомлений</h1>
          <p>{isRegister ? 'Создайте аккаунт' : 'Войдите в систему'}</p>
        </div>

        <form onSubmit={handleSubmit} className="login-form">
          {error && <div className="alert alert-error">{error}</div>}

          <div className="form-group">
            <label className="form-label" htmlFor="login">Логин</label>
            <input
              id="login"
              type="text"
              className="form-input"
              value={login}
              onChange={(e) => setLogin(e.target.value)}
              required
              autoComplete="username"
            />
          </div>

          <div className="form-group">
            <label className="form-label" htmlFor="password">Пароль</label>
            <input
              id="password"
              type="password"
              className="form-input"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              required
              autoComplete={isRegister ? 'new-password' : 'current-password'}
            />
          </div>

          {isRegister && (
            <div className="form-group">
              <label className="form-label" htmlFor="confirmPassword">Подтверждение пароля</label>
              <input
                id="confirmPassword"
                type="password"
                className="form-input"
                value={confirmPassword}
                onChange={(e) => setConfirmPassword(e.target.value)}
                required
                autoComplete="new-password"
              />
            </div>
          )}

          <button type="submit" className="btn btn-primary btn-full" disabled={isLoading}>
            {isLoading ? 'Загрузка...' : (isRegister ? 'Зарегистрироваться' : 'Войти')}
          </button>
        </form>

        <div className="login-footer">
          <button
            type="button"
            className="btn-link"
            onClick={() => {
              setIsRegister(!isRegister)
              setError('')
            }}
          >
            {isRegister ? 'Уже есть аккаунт? Войти' : 'Нет аккаунта? Зарегистрироваться'}
          </button>
        </div>
      </div>
    </div>
  )
}