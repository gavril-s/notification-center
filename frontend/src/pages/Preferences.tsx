import { useState, useEffect } from 'react'
import { recipientsApi, Preference, PreferenceUpsertRequest, Contact, getErrorMessage } from '../api/client'
import './Preferences.css'

const channelLabels: Record<string, string> = {
  email: 'Email',
  sms: 'SMS',
  telegram: 'Telegram',
}

const scopeTypeLabels: Record<string, string> = {
  global: 'Глобальные',
  sender: 'Отправитель',
  campaign: 'Рассылка',
  group: 'Группа',
}

export default function Preferences() {
  const [preferences, setPreferences] = useState<Preference[]>([])
  const [contacts, setContacts] = useState<Contact[]>([])
  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState('')
  const [showModal, setShowModal] = useState(false)
  const [formData, setFormData] = useState<PreferenceUpsertRequest>({
    contact_id: '',
    scope_type: 'global',
    enabled: true,
    blocked_channels: [],
    quiet_from: null,
    quiet_to: null,
    sender_id: null,
    scope_id: null,
  })
  const [formError, setFormError] = useState('')
  const [isSubmitting, setIsSubmitting] = useState(false)

  const loadData = async () => {
    try {
      const [prefResponse, contactsResponse] = await Promise.all([
        recipientsApi.getPreferences(1, 100),
        recipientsApi.getContacts(1, 100),
      ])
      setPreferences(prefResponse.data.items)
      setContacts(contactsResponse.data.items)
    } catch (err) {
      setError(getErrorMessage(err))
    } finally {
      setIsLoading(false)
    }
  }

  useEffect(() => {
    loadData()
  }, [])

  const handleCreate = () => {
    setFormData({
      contact_id: contacts[0]?.contact_id || '',
      scope_type: 'global',
      enabled: true,
      blocked_channels: [],
      quiet_from: null,
      quiet_to: null,
      sender_id: null,
      scope_id: null,
    })
    setFormError('')
    setShowModal(true)
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setFormError('')
    setIsSubmitting(true)

    try {
      await recipientsApi.upsertPreference(formData)
      setShowModal(false)
      await loadData()
    } catch (err) {
      setFormError(getErrorMessage(err))
    } finally {
      setIsSubmitting(false)
    }
  }

  const getContactValue = (contactId: string) => {
    const contact = contacts.find(c => c.contact_id === contactId)
    return contact ? `${channelLabels[contact.channel]}: ${contact.value}` : contactId
  }

  if (isLoading) {
    return (
      <div className="loading-screen">
        <div className="spinner"></div>
        <p>Загрузка настроек...</p>
      </div>
    )
  }

  return (
    <div className="preferences-page">
      <div className="card">
        <div className="card-header">
          <h2 className="card-title">Настройки уведомлений</h2>
          <button onClick={handleCreate} className="btn btn-primary">
            Добавить настройку
          </button>
        </div>

        {error && <div className="alert alert-error">{error}</div>}

        {contacts.length === 0 ? (
          <div className="empty-state">
            <div className="empty-state-icon">⚙️</div>
            <p>Сначала добавьте контакт в разделе «Контакты»</p>
          </div>
        ) : preferences.length === 0 ? (
          <div className="empty-state">
            <div className="empty-state-icon">⚙️</div>
            <p>У вас пока нет настроек уведомлений</p>
            <button onClick={handleCreate} className="btn btn-primary" style={{ marginTop: '1rem' }}>
              Добавить первую настройку
            </button>
          </div>
        ) : (
          <table className="table">
            <thead>
              <tr>
                <th>Контакт</th>
                <th>Тип области</th>
                <th>Статус</th>
                <th>Тихие часы</th>
                <th>Заблокированные каналы</th>
              </tr>
            </thead>
            <tbody>
              {preferences.map(pref => (
                <tr key={pref.preference_id}>
                  <td>{getContactValue(pref.contact_id)}</td>
                  <td>{scopeTypeLabels[pref.scope_type]}</td>
                  <td>
                    {pref.enabled ? (
                      <span className="badge badge-success">Включены</span>
                    ) : (
                      <span className="badge badge-error">Отключены</span>
                    )}
                  </td>
                  <td>
                    {pref.quiet_from && pref.quiet_to
                      ? `${pref.quiet_from} - ${pref.quiet_to}`
                      : 'Не настроены'}
                  </td>
                  <td>
                    {pref.blocked_channels.length > 0
                      ? pref.blocked_channels.map(ch => channelLabels[ch]).join(', ')
                      : 'Нет'}
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
              <h3 className="modal-title">Добавить настройку</h3>
              <button onClick={() => setShowModal(false)} className="modal-close">&times;</button>
            </div>
            <form onSubmit={handleSubmit}>
              <div className="modal-body">
                {formError && <div className="alert alert-error">{formError}</div>}

                <div className="form-group">
                  <label className="form-label">Контакт</label>
                  <select
                    className="form-select"
                    value={formData.contact_id}
                    onChange={e => setFormData({ ...formData, contact_id: e.target.value })}
                    required
                  >
                    <option value="">Выберите контакт</option>
                    {contacts.map(contact => (
                      <option key={contact.contact_id} value={contact.contact_id}>
                        {channelLabels[contact.channel]}: {contact.value}
                      </option>
                    ))}
                  </select>
                </div>

                <div className="form-group">
                  <label className="form-label">Тип области</label>
                  <select
                    className="form-select"
                    value={formData.scope_type}
                    onChange={e => setFormData({ ...formData, scope_type: e.target.value as any })}
                  >
                    <option value="global">Глобальные</option>
                    <option value="sender">Отправитель</option>
                    <option value="campaign">Рассылка</option>
                    <option value="group">Группа</option>
                  </select>
                </div>

                <div className="form-group">
                  <label className="checkbox-label">
                    <input
                      type="checkbox"
                      checked={formData.enabled}
                      onChange={e => setFormData({ ...formData, enabled: e.target.checked })}
                    />
                    Включить уведомления
                  </label>
                </div>

                <div className="form-group">
                  <label className="form-label">Тихие часы (начало)</label>
                  <input
                    type="time"
                    className="form-input"
                    value={formData.quiet_from || ''}
                    onChange={e => setFormData({ ...formData, quiet_from: e.target.value || null })}
                  />
                </div>

                <div className="form-group">
                  <label className="form-label">Тихие часы (конец)</label>
                  <input
                    type="time"
                    className="form-input"
                    value={formData.quiet_to || ''}
                    onChange={e => setFormData({ ...formData, quiet_to: e.target.value || null })}
                  />
                </div>

                <div className="form-group">
                  <label className="form-label">Заблокированные каналы</label>
                  <div className="checkbox-group">
                    {(['email', 'sms', 'telegram'] as const).map(channel => (
                      <label key={channel} className="checkbox-label">
                        <input
                          type="checkbox"
                          checked={formData.blocked_channels.includes(channel)}
                          onChange={e => {
                            if (e.target.checked) {
                              setFormData({
                                ...formData,
                                blocked_channels: [...formData.blocked_channels, channel],
                              })
                            } else {
                              setFormData({
                                ...formData,
                                blocked_channels: formData.blocked_channels.filter(ch => ch !== channel),
                              })
                            }
                          }}
                        />
                        {channelLabels[channel]}
                      </label>
                    ))}
                  </div>
                </div>
              </div>
              <div className="modal-footer">
                <button type="button" onClick={() => setShowModal(false)} className="btn btn-secondary">
                  Отмена
                </button>
                <button type="submit" className="btn btn-primary" disabled={isSubmitting}>
                  {isSubmitting ? 'Сохранение...' : 'Сохранить'}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  )
}