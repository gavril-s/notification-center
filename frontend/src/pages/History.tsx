import { useState, useEffect } from 'react'
import { notificationsApi, NotificationHistoryItem, getErrorMessage } from '../api/client'
import './History.css'

const statusLabels: Record<string, { label: string; class: string }> = {
  draft: { label: 'Черновик', class: 'badge-neutral' },
  scheduled: { label: 'Запланировано', class: 'badge-info' },
  queued: { label: 'В очереди', class: 'badge-warning' },
  processing: { label: 'Отправляется', class: 'badge-warning' },
  sent: { label: 'Отправлено', class: 'badge-success' },
  delivered: { label: 'Доставлено', class: 'badge-success' },
  failed: { label: 'Ошибка', class: 'badge-error' },
  cancelled: { label: 'Отменено', class: 'badge-error' },
  skipped_by_preference: { label: 'Пропущено', class: 'badge-neutral' },
}

const channelLabels: Record<string, string> = {
  email: 'Email',
  sms: 'SMS',
  telegram: 'Telegram',
}

export default function History() {
  const [history, setHistory] = useState<NotificationHistoryItem[]>([])
  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState('')
  const [page, setPage] = useState(1)
  const [total, setTotal] = useState(0)
  const size = 20

  const loadHistory = async (pageNum: number) => {
    setIsLoading(true)
    try {
      const response = await notificationsApi.getHistory({
        mode: 'recipient',
        page: pageNum,
        size,
      })
      setHistory(response.data.items)
      setTotal(response.data.total)
      setPage(pageNum)
    } catch (err) {
      setError(getErrorMessage(err))
    } finally {
      setIsLoading(false)
    }
  }

  useEffect(() => {
    loadHistory(1)
  }, [])

  const formatDate = (dateStr: string) => {
    return new Date(dateStr).toLocaleString('ru-RU', {
      day: 'numeric',
      month: 'short',
      year: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
    })
  }

  const totalPages = Math.ceil(total / size)

  if (isLoading && history.length === 0) {
    return (
      <div className="loading-screen">
        <div className="spinner"></div>
        <p>Загрузка истории...</p>
      </div>
    )
  }

  return (
    <div className="history-page">
      <div className="card">
        <div className="card-header">
          <h2 className="card-title">История уведомлений</h2>
        </div>

        {error && <div className="alert alert-error">{error}</div>}

        {history.length === 0 ? (
          <div className="empty-state">
            <div className="empty-state-icon">📋</div>
            <p>У вас пока нет уведомлений</p>
          </div>
        ) : (
          <>
            <table className="table">
              <thead>
                <tr>
                  <th>Дата</th>
                  <th>Канал</th>
                  <th>Статус</th>
                </tr>
              </thead>
              <tbody>
                {history.map(item => (
                  <tr key={item.notification_id}>
                    <td>{formatDate(item.created_at)}</td>
                    <td>{channelLabels[item.channel]}</td>
                    <td>
                      <span className={`badge ${statusLabels[item.status]?.class || 'badge-neutral'}`}>
                        {statusLabels[item.status]?.label || item.status}
                      </span>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>

            {totalPages > 1 && (
              <div className="pagination">
                <button
                  className="pagination-btn"
                  onClick={() => loadHistory(page - 1)}
                  disabled={page === 1}
                >
                  Предыдущая
                </button>
                <span className="pagination-info">
                  Страница {page} из {totalPages}
                </span>
                <button
                  className="pagination-btn"
                  onClick={() => loadHistory(page + 1)}
                  disabled={page >= totalPages}
                >
                  Следующая
                </button>
              </div>
            )}
          </>
        )}
      </div>
    </div>
  )
}