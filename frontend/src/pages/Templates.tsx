import { useState, useEffect } from 'react'
import { sourcesApi, Template, CreateTemplateRequest, Sender, getErrorMessage } from '../api/client'
import './Templates.css'

const channelLabels: Record<string, string> = {
  email: 'Email',
  sms: 'SMS',
  telegram: 'Telegram',
}

export default function Templates() {
  const [templates, setTemplates] = useState<Template[]>([])
  const [senders, setSenders] = useState<Sender[]>([])
  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState('')
  const [showModal, setShowModal] = useState(false)
  const [editingTemplate, setEditingTemplate] = useState<Template | null>(null)
  const [formData, setFormData] = useState<CreateTemplateRequest>({
    sender_id: '',
    name: '',
    channel: 'email',
    subject: null,
    body: '',
    variables: [],
  })
  const [formError, setFormError] = useState('')
  const [isSubmitting, setIsSubmitting] = useState(false)
  const [selectedSender, setSelectedSender] = useState<string>('')

  const loadData = async () => {
    try {
      const [templatesResponse, sendersResponse] = await Promise.all([
        sourcesApi.getTemplates(1, 100, selectedSender || undefined),
        sourcesApi.getSenders(1, 100),
      ])
      setTemplates(templatesResponse.data.items)
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
    setEditingTemplate(null)
    setFormData({
      sender_id: senders[0]?.sender_id || '',
      name: '',
      channel: 'email',
      subject: null,
      body: '',
      variables: [],
    })
    setFormError('')
    setShowModal(true)
  }

  const handleEdit = (template: Template) => {
    setEditingTemplate(template)
    setFormData({
      sender_id: template.sender_id,
      name: template.name,
      channel: template.channel,
      subject: template.subject,
      body: template.body,
      variables: template.variables,
    })
    setFormError('')
    setShowModal(true)
  }

  const handleDelete = async (templateId: string) => {
    if (!confirm('Вы уверены, что хотите удалить этот шаблон?')) return

    try {
      await sourcesApi.deleteTemplate(templateId)
      await loadData()
    } catch (err) {
      setError(getErrorMessage(err))
    }
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setFormError('')
    setIsSubmitting(true)

    try {
      if (editingTemplate) {
        await sourcesApi.updateTemplate(editingTemplate.template_id, formData)
      } else {
        await sourcesApi.createTemplate(formData)
      }
      setShowModal(false)
      await loadData()
    } catch (err) {
      setFormError(getErrorMessage(err))
    } finally {
      setIsSubmitting(false)
    }
  }

  const handleVariablesChange = (value: string) => {
    const vars = value.split(',').map(v => v.trim()).filter(v => v)
    setFormData({ ...formData, variables: vars })
  }

  const getSenderName = (senderId: string) => {
    const sender = senders.find(s => s.sender_id === senderId)
    return sender?.name || senderId
  }

  if (isLoading) {
    return (
      <div className="loading-screen">
        <div className="spinner"></div>
        <p>Загрузка шаблонов...</p>
      </div>
    )
  }

  return (
    <div className="templates-page">
      <div className="card">
        <div className="card-header">
          <h2 className="card-title">Шаблоны уведомлений</h2>
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
              Создать шаблон
            </button>
          </div>
        </div>

        {error && <div className="alert alert-error">{error}</div>}

        {senders.length === 0 ? (
          <div className="empty-state">
            <div className="empty-state-icon">📝</div>
            <p>Сначала создайте отправителя</p>
          </div>
        ) : templates.length === 0 ? (
          <div className="empty-state">
            <div className="empty-state-icon">📝</div>
            <p>У вас пока нет шаблонов</p>
            <button onClick={handleCreate} className="btn btn-primary" style={{ marginTop: '1rem' }}>
              Создать первый шаблон
            </button>
          </div>
        ) : (
          <table className="table">
            <thead>
              <tr>
                <th>Название</th>
                <th>Отправитель</th>
                <th>Канал</th>
                <th>Переменные</th>
                <th>Статус</th>
                <th>Действия</th>
              </tr>
            </thead>
            <tbody>
              {templates.map(template => (
                <tr key={template.template_id}>
                  <td>
                    <div>
                      <strong>{template.name}</strong>
                      {template.subject && (
                        <p className="template-subject">{template.subject}</p>
                      )}
                    </div>
                  </td>
                  <td>{getSenderName(template.sender_id)}</td>
                  <td>{channelLabels[template.channel]}</td>
                  <td>{template.variables.join(', ') || '-'}</td>
                  <td>
                    {template.active ? (
                      <span className="badge badge-success">Активен</span>
                    ) : (
                      <span className="badge badge-neutral">Неактивен</span>
                    )}
                  </td>
                  <td>
                    <div className="table-actions">
                      <button onClick={() => handleEdit(template)} className="btn btn-secondary btn-sm">
                        Изменить
                      </button>
                      <button onClick={() => handleDelete(template.template_id)} className="btn btn-danger btn-sm">
                        Удалить
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
                {editingTemplate ? 'Изменить шаблон' : 'Создать шаблон'}
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
                    disabled={!!editingTemplate}
                  >
                    {senders.map(sender => (
                      <option key={sender.sender_id} value={sender.sender_id}>
                        {sender.name}
                      </option>
                    ))}
                  </select>
                </div>

                <div className="form-group">
                  <label className="form-label">Название</label>
                  <input
                    type="text"
                    className="form-input"
                    value={formData.name}
                    onChange={e => setFormData({ ...formData, name: e.target.value })}
                    required
                  />
                </div>

                <div className="form-group">
                  <label className="form-label">Канал</label>
                  <select
                    className="form-select"
                    value={formData.channel}
                    onChange={e => setFormData({ ...formData, channel: e.target.value as any })}
                    disabled={!!editingTemplate}
                  >
                    <option value="email">Email</option>
                    <option value="sms">SMS</option>
                    <option value="telegram">Telegram</option>
                  </select>
                </div>

                {formData.channel === 'email' && (
                  <div className="form-group">
                    <label className="form-label">Тема письма</label>
                    <input
                      type="text"
                      className="form-input"
                      value={formData.subject || ''}
                      onChange={e => setFormData({ ...formData, subject: e.target.value || null })}
                    />
                  </div>
                )}

                <div className="form-group">
                  <label className="form-label">Текст</label>
                  <textarea
                    className="form-input"
                    value={formData.body}
                    onChange={e => setFormData({ ...formData, body: e.target.value })}
                    rows={5}
                    required
                  />
                </div>

                <div className="form-group">
                  <label className="form-label">Переменные (через запятую)</label>
                  <input
                    type="text"
                    className="form-input"
                    value={formData.variables.join(', ')}
                    onChange={e => handleVariablesChange(e.target.value)}
                    placeholder="name, email, code"
                  />
                  <p className="form-hint">Например: name, email, verification_code</p>
                </div>
              </div>
              <div className="modal-footer">
                <button type="button" onClick={() => setShowModal(false)} className="btn btn-secondary">
                  Отмена
                </button>
                <button type="submit" className="btn btn-primary" disabled={isSubmitting}>
                  {isSubmitting ? 'Сохранение...' : (editingTemplate ? 'Сохранить' : 'Создать')}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  )
}