import { useState, useEffect } from 'react'
import { notificationsApi, NotificationHistoryItem, Sender, getErrorMessage } from '../api/client'
import { sourcesApi } from '../api/client'
import './OperatorHistory.css'

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

export default function OperatorHistory() {
  const [history, setHistory] = useState<NotificationHistoryItem[]>([])
  const [senders, setSenders] = useState<Sender[]>([])
  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState('')
  const [page, setPage] = useState(1)
  const [total, setTotal] = useState(0)
  const [selectedSender, setSelectedSender] = useState<string>('')
  const [selectedStatus, setSelectedStatus] = useState<string>('')
  const size = 20

  const loadData = async (pageNum: number) => {
    setIsLoading(true)
    try {
      const [historyResponse, sendersResponse] = await Promise.all([
        notificationsApi.getHistory({
          mode: 'operator',
          sender_id: selectedSender || undefined,
          status: selectedStatus || undefined,
          page: pageNum,
          size,
        }),
        sourcesApi.getSenders(1, 100),
      ])
      setHistory(historyResponse.data.items)
      setTotal(historyResponse.data.total)
      setSenders(sendersResponse.data.items)
      setPage(pageNum)
    } catch (err) {
      setError(getErrorMessage(err))
    } finally {
      setIsLoading(false)
    }
  }

  useEffect(() => {
    loadData(1)
  }, [selectedSender, selectedStatus])

  const formatDate = (dateStr: string) => {
    return new Date(dateStr).toLocaleString('ru-RU', {
      day: 'numeric',
      month: 'short',
      year: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
    })
  }

  const getSenderName = (senderId: string) => {
    const sender = senders.find(s => s.sender_id === senderId)
    return sender?.name || senderId
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
    <div className="operator-history-page">
      <div className="card">
        <div className="card-header">
          <h2 className="card-title">История отправок</h2>
          <div className="card-header-actions">
            <select
              className="form-select"
              value={selectedSender}
              onChange={e => setSelectedSender(e.target.value)}
              style={{ width: 'auto', minWidth: '200px' }}
            >
              <option value="">Все отправители</option>
              {senders.map(sender => (
                <option key={sender.sender_id} value={sender.sender_id}>
                  {sender.name}
                </option>
              ))}
            </select>
            <select
              className="form-select"
              value={selectedStatus}
              onChange={e => setSelectedStatus(e.target.value)}
              style={{ width: 'auto', minWidth: '150px' }}
            >
              <option value="">Все статусы</option>
              {Object.entries(statusLabels).map(([key, { label }]) => (
                <option key={key} value={key}>
                  {label}
                </option>
              ))}
            </select>
          </div>
        </div>

        {error && <div className="alert alert-error">{error}</div>}

        {history.length === 0 ? (
          <div className="empty-state">
            <div className="empty-state-icon">📋</div>
            <p>История пуста</p>
          </div>
        ) : (
          <>
            <table className="table">
              <thead>
                <tr>
                  <th>Дата</th>
                  <th>Отправитель</th>
                  <th>Канал</th>
                  <th>Статус</th>
                </tr>
              </thead>
              <tbody>
                {history.map(item => (
                  <tr key={item.notification_id}>
                    <td>{formatDate(item.created_at)}</td>
                    <td>{getSenderName(item.sender_id)}</td>
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
                  onClick={() => loadData(page - 1)}
                  disabled={page === 1}
                >
                  Предыдущая
                </button>
                <span className="pagination-info">
                  Страница {page} из {totalPages}
                </span>
                <button
                  className="pagination-btn"
                  onClick={() => loadData(page + 1)}
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