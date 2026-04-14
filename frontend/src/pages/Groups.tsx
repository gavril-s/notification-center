import { useState, useEffect } from 'react'
import { sourcesApi, Group, CreateGroupRequest, UpdateGroupRequest, Sender, getErrorMessage } from '../api/client'
import './Groups.css'

export default function Groups() {
  const [groups, setGroups] = useState<Group[]>([])
  const [senders, setSenders] = useState<Sender[]>([])
  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState('')
  const [showModal, setShowModal] = useState(false)
  const [editingGroup, setEditingGroup] = useState<Group | null>(null)
  const [formData, setFormData] = useState<CreateGroupRequest>({
    sender_id: '',
    name: '',
  })
  const [formError, setFormError] = useState('')
  const [isSubmitting, setIsSubmitting] = useState(false)
  const [selectedSender, setSelectedSender] = useState<string>('')

  const loadData = async () => {
    try {
      const [groupsResponse, sendersResponse] = await Promise.all([
        sourcesApi.getGroups(1, 100, selectedSender || undefined),
        sourcesApi.getSenders(1, 100),
      ])
      setGroups(groupsResponse.data.items)
      setSenders(sendersResponse.data.items)
      
      if (!selectedSender && sendersResponse.data.items.length > 0) {
        setFormData(prev => ({ ...prev, sender_id: sendersResponse.data.items[0].sender_id }))
      }
    } catch (err) {
      setError(getErrorMessage(err))
    } finally {
      setIsLoading(false)
    }
  }

  useEffect(() => {
    loadData()
  }, [selectedSender])

  const handleCreate = () => {
    setEditingGroup(null)
    setFormData({
      sender_id: senders[0]?.sender_id || '',
      name: '',
    })
    setFormError('')
    setShowModal(true)
  }

  const handleEdit = (group: Group) => {
    setEditingGroup(group)
    setFormData({
      sender_id: group.sender_id,
      name: group.name,
    })
    setFormError('')
    setShowModal(true)
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setFormError('')
    setIsSubmitting(true)

    try {
      if (editingGroup) {
        const updateData: UpdateGroupRequest = {
          name: formData.name,
        }
        await sourcesApi.updateGroup(editingGroup.group_id, updateData)
      } else {
        await sourcesApi.createGroup(formData)
      }
      setShowModal(false)
      await loadData()
    } catch (err) {
      setFormError(getErrorMessage(err))
    } finally {
      setIsSubmitting(false)
    }
  }

  const getSenderName = (senderId: string) => {
    const sender = senders.find(s => s.sender_id === senderId)
    return sender?.name || senderId
  }

  if (isLoading) {
    return (
      <div className="loading-screen">
        <div className="spinner"></div>
        <p>Загрузка групп...</p>
      </div>
    )
  }

  return (
    <div className="groups-page">
      <div className="card">
        <div className="card-header">
          <h2 className="card-title">Группы получателей</h2>
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
            <button onClick={handleCreate} className="btn btn-primary">
              Создать группу
            </button>
          </div>
        </div>

        {error && <div className="alert alert-error">{error}</div>}

        {senders.length === 0 ? (
          <div className="empty-state">
            <div className="empty-state-icon">👥</div>
            <p>Сначала создайте отправителя</p>
          </div>
        ) : groups.length === 0 ? (
          <div className="empty-state">
            <div className="empty-state-icon">👥</div>
            <p>У вас пока нет групп получателей</p>
            <button onClick={handleCreate} className="btn btn-primary" style={{ marginTop: '1rem' }}>
              Создать первую группу
            </button>
          </div>
        ) : (
          <table className="table">
            <thead>
              <tr>
                <th>Название</th>
                <th>Отправитель</th>
                <th>Статус</th>
                <th>Действия</th>
              </tr>
            </thead>
            <tbody>
              {groups.map(group => (
                <tr key={group.group_id}>
                  <td><strong>{group.name}</strong></td>
                  <td>{getSenderName(group.sender_id)}</td>
                  <td>
                    {group.active ? (
                      <span className="badge badge-success">Активна</span>
                    ) : (
                      <span className="badge badge-neutral">Неактивна</span>
                    )}
                  </td>
                  <td>
                    <div className="table-actions">
                      <button onClick={() => handleEdit(group)} className="btn btn-secondary btn-sm">
                        Изменить
                      </button>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>

      {showModal && (
        <div className="modal-overlay" onClick={() => setShowModal(false)}>
          <div className="modal" onClick={e => e.stopPropagation()}>
            <div className="modal-header">
              <h3 className="modal-title">
                {editingGroup ? 'Изменить группу' : 'Создать группу'}
              </h3>
              <button onClick={() => setShowModal(false)} className="modal-close">&times;</button>
            </div>
            <form onSubmit={handleSubmit}>
              <div className="modal-body">
                {formError && <div className="alert alert-error">{formError}</div>}

                <div className="form-group">
                  <label className="form-label">Отправитель</label>
                  <select
                    className="form-select"
                    value={formData.sender_id}
                    onChange={e => setFormData({ ...formData, sender_id: e.target.value })}
                    required
                    disabled={!!editingGroup}
                  >
                    {senders.map(sender => (
                      <option key={sender.sender_id} value={sender.sender_id}>
                        {sender.name}
                      </option>
                    ))}
                  </select>
                </div>

                <div className="form-group">
                  <label className="form-label">Название группы</label>
                  <input
                    type="text"
                    className="form-input"
                    value={formData.name}
                    onChange={e => setFormData({ ...formData, name: e.target.value })}
                    required
                    placeholder="Например: Подписчики новостей"
                  />
                </div>
              </div>
              <div className="modal-footer">
                <button type="button" onClick={() => setShowModal(false)} className="btn btn-secondary">
                  Отмена
                </button>
                <button type="submit" className="btn btn-primary" disabled={isSubmitting}>
                  {isSubmitting ? 'Сохранение...' : (editingGroup ? 'Сохранить' : 'Создать')}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  )
}