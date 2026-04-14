import { useState, useEffect } from 'react'
import { notificationsApi, SenderAnalyticsResponse, Sender, getErrorMessage } from '../api/client'
import { sourcesApi } from '../api/client'
import './Analytics.css'

export default function Analytics() {
  const [analytics, setAnalytics] = useState<SenderAnalyticsResponse | null>(null)
  const [senders, setSenders] = useState<Sender[]>([])
  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState('')
  const [selectedSender, setSelectedSender] = useState<string>('')
  const [fromDate, setFromDate] = useState<string>('')
  const [toDate, setToDate] = useState<string>('')

  const loadData = async () => {
    if (!selectedSender) return
    
    setIsLoading(true)
    try {
      const response = await notificationsApi.getAnalytics(
        selectedSender,
        fromDate || undefined,
        toDate || undefined
      )
      setAnalytics(response.data)
    } catch (err) {
      setError(getErrorMessage(err))
    } finally {
      setIsLoading(false)
    }
  }

  useEffect(() => {
    const loadSenders = async () => {
      try {
        const response = await sourcesApi.getSenders(1, 100)
        setSenders(response.data.items)
        if (response.data.items.length > 0) {
          setSelectedSender(response.data.items[0].sender_id)
        }
      } catch (err) {
        setError(getErrorMessage(err))
      }
    }
    loadSenders()
  }, [])

  useEffect(() => {
    if (selectedSender) {
      loadData()
    }
  }, [selectedSender, fromDate, toDate])

  const total = analytics ? 
    analytics.totals.queued + 
    analytics.totals.delivered + 
    analytics.totals.failed + 
    analytics.totals.skipped_by_preference : 0

  const getPercentage = (value: number) => {
    if (total === 0) return 0
    return Math.round((value / total) * 100)
  }

  const formatDate = (dateStr: string | null) => {
    if (!dateStr) return 'За всё время'
    return new Date(dateStr).toLocaleDateString('ru-RU', {
      day: 'numeric',
      month: 'short',
      year: 'numeric',
    })
  }

  if (isLoading && !analytics) {
    return (
      <div className="loading-screen">
        <div className="spinner"></div>
        <p>Загрузка аналитики...</p>
      </div>
    )
  }

  return (
    <div className="analytics-page">
      <div className="card">
        <div className="card-header">
          <h2 className="card-title">Аналитика по отправителю</h2>
          <div className="card-header-actions">
            <select
              className="form-select"
              value={selectedSender}
              onChange={e => setSelectedSender(e.target.value)}
              style={{ width: 'auto', minWidth: '200px' }}
            >
              {senders.map(sender => (
                <option key={sender.sender_id} value={sender.sender_id}>
                  {sender.name}
                </option>
              ))}
            </select>
          </div>
        </div>

        <div className="analytics-filters">
          <div className="form-group" style={{ marginBottom: 0 }}>
            <label className="form-label">От</label>
            <input
              type="date"
              className="form-input"
              value={fromDate}
              onChange={e => setFromDate(e.target.value)}
            />
          </div>
          <div className="form-group" style={{ marginBottom: 0 }}>
            <label className="form-label">До</label>
            <input
              type="date"
              className="form-input"
              value={toDate}
              onChange={e => setToDate(e.target.value)}
            />
          </div>
        </div>

        {error && <div className="alert alert-error">{error}</div>}

        {analytics && (
          <div className="analytics-content">
            <div className="analytics-period">
              Период: {formatDate(analytics.from)} — {formatDate(analytics.to)}
            </div>

            <div className="analytics-grid">
              <div className="analytics-card">
                <div className="analytics-value">{analytics.totals.delivered}</div>
                <div className="analytics-label">Доставлено</div>
                <div className="analytics-bar">
                  <div 
                    className="analytics-bar-fill analytics-bar-success" 
                    style={{ width: `${getPercentage(analytics.totals.delivered)}%` }}
                  />
                </div>
                <div className="analytics-percent">{getPercentage(analytics.totals.delivered)}%</div>
              </div>

              <div className="analytics-card">
                <div className="analytics-value">{analytics.totals.queued}</div>
                <div className="analytics-label">В очереди</div>
                <div className="analytics-bar">
                  <div 
                    className="analytics-bar-fill analytics-bar-warning" 
                    style={{ width: `${getPercentage(analytics.totals.queued)}%` }}
                  />
                </div>
                <div className="analytics-percent">{getPercentage(analytics.totals.queued)}%</div>
              </div>

              <div className="analytics-card">
                <div className="analytics-value">{analytics.totals.failed}</div>
                <div className="analytics-label">Ошибки</div>
                <div className="analytics-bar">
                  <div 
                    className="analytics-bar-fill analytics-bar-error" 
                    style={{ width: `${getPercentage(analytics.totals.failed)}%` }}
                  />
                </div>
                <div className="analytics-percent">{getPercentage(analytics.totals.failed)}%</div>
              </div>

              <div className="analytics-card">
                <div className="analytics-value">{analytics.totals.skipped_by_preference}</div>
                <div className="analytics-label">Пропущено (настройки)</div>
                <div className="analytics-bar">
                  <div 
                    className="analytics-bar-fill analytics-bar-neutral" 
                    style={{ width: `${getPercentage(analytics.totals.skipped_by_preference)}%` }}
                  />
                </div>
                <div className="analytics-percent">{getPercentage(analytics.totals.skipped_by_preference)}%</div>
              </div>
            </div>

            <div className="analytics-total">
              Всего обработано: <strong>{total}</strong>
            </div>
          </div>
        )}
      </div>
    </div>
  )
}