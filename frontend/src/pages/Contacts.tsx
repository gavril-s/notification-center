import { useState, useEffect } from 'react'
import { recipientsApi, Contact, CreateContactRequest, UpdateContactRequest, getErrorMessage } from '../api/client'
import './Contacts.css'

const channelLabels: Record<string, string> = {
  email: 'Email',
  sms: 'SMS',
  telegram: 'Telegram',
}

export default function Contacts() {
  const [contacts, setContacts] = useState<Contact[]>([])
  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState('')
  const [showModal, setShowModal] = useState(false)
  const [editingContact, setEditingContact] = useState<Contact | null>(null)
  const [formData, setFormData] = useState<CreateContactRequest>({
    channel: 'email',
    value: '',
  })
  const [formError, setFormError] = useState('')
  const [isSubmitting, setIsSubmitting] = useState(false)

  const loadContacts = async () => {
    try {
      const response = await recipientsApi.getContacts(1, 100)
      setContacts(response.data.items)
    } catch (err) {
      setError(getErrorMessage(err))
    } finally {
      setIsLoading(false)
    }
  }

  useEffect(() => {
    loadContacts()
  }, [])

  const handleCreate = () => {
    setEditingContact(null)
    setFormData({ channel: 'email', value: '' })
    setFormError('')
    setShowModal(true)
  }

  const handleEdit = (contact: Contact) => {
    setEditingContact(contact)
    setFormData({ channel: contact.channel, value: contact.value })
    setFormError('')
    setShowModal(true)
  }

  const handleDelete = async (contactId: string) => {
    if (!confirm('Вы уверены, что хотите удалить этот контакт?')) return

    try {
      await recipientsApi.deleteContact(contactId)
      await loadContacts()
    } catch (err) {
      setError(getErrorMessage(err))
    }
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setFormError('')
    setIsSubmitting(true)

    try {
      if (editingContact) {
        const updateData: UpdateContactRequest = {
          value: formData.value,
        }
        await recipientsApi.updateContact(editingContact.contact_id, updateData)
      } else {
        await recipientsApi.createContact(formData)
      }
      setShowModal(false)
      await loadContacts()
    } catch (err) {
      setFormError(getErrorMessage(err))
    } finally {
      setIsSubmitting(false)
    }
  }

  const handleToggleEnabled = async (contact: Contact) => {
    try {
      await recipientsApi.updateContact(contact.contact_id, { enabled: !contact.enabled })
      await loadContacts()
    } catch (err) {
      setError(getErrorMessage(err))
    }
  }

  if (isLoading) {
    return (
      <div className="loading-screen">
        <div className="spinner"></div>
        <p>Загрузка контактов...</p>
      </div>
    )
  }

  return (
    <div className="contacts-page">
      <div className="card">
        <div className="card-header">
          <h2 className="card-title">Мои контакты</h2>
          <button onClick={handleCreate} className="btn btn-primary">
            Добавить контакт
          </button>
        </div>

        {error && <div className="alert alert-error">{error}</div>}

        {contacts.length === 0 ? (
          <div className="empty-state">
            <div className="empty-state-icon">📱</div>
            <p>У вас пока нет контактов</p>
            <button onClick={handleCreate} className="btn btn-primary" style={{ marginTop: '1rem' }}>
              Добавить первый контакт
            </button>
          </div>
        ) : (
          <table className="table">
            <thead>
              <tr>
                <th>Канал</th>
                <th>Значение</th>
                <th>Статус</th>
                <th>Подтверждён</th>
                <th>Действия</th>
              </tr>
            </thead>
            <tbody>
              {contacts.map(contact => (
                <tr key={contact.contact_id}>
                  <td>{channelLabels[contact.channel]}</td>
                  <td>{contact.value}</td>
                  <td>
                    <label className="checkbox-label">
                      <input
                        type="checkbox"
                        checked={contact.enabled}
                        onChange={() => handleToggleEnabled(contact)}
                      />
                      {contact.enabled ? 'Активен' : 'Отключён'}
                    </label>
                  </td>
                  <td>
                    {contact.verified ? (
                      <span className="badge badge-success">Да</span>
                    ) : (
                      <span className="badge badge-warning">Нет</span>
                    )}
                  </td>
                  <td>
                    <div className="table-actions">
                      <button onClick={() => handleEdit(contact)} className="btn btn-secondary btn-sm">
                        Изменить
                      </button>
                      <button onClick={() => handleDelete(contact.contact_id)} className="btn btn-danger btn-sm">
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
                {editingContact ? 'Изменить контакт' : 'Добавить контакт'}
              </h3>
              <button onClick={() => setShowModal(false)} className="modal-close">&times;</button>
            </div>
            <form onSubmit={handleSubmit}>
              <div className="modal-body">
                {formError && <div className="alert alert-error">{formError}</div>}

                <div className="form-group">
                  <label className="form-label">Канал</label>
                  <select
                    className="form-select"
                    value={formData.channel}
                    onChange={e => setFormData({ ...formData, channel: e.target.value as 'email' | 'sms' | 'telegram' })}
                    disabled={!!editingContact}
                  >
                    <option value="email">Email</option>
                    <option value="sms">SMS</option>
                    <option value="telegram">Telegram</option>
                  </select>
                </div>

                <div className="form-group">
                  <label className="form-label">Значение</label>
                  <input
                    type={formData.channel === 'email' ? 'email' : 'text'}
                    className="form-input"
                    value={formData.value}
                    onChange={e => setFormData({ ...formData, value: e.target.value })}
                    placeholder={
                      formData.channel === 'email' ? 'example@mail.ru' :
                      formData.channel === 'sms' ? '+79001234567' : '@username'
                    }
                    required
                  />
                </div>
              </div>
              <div className="modal-footer">
                <button type="button" onClick={() => setShowModal(false)} className="btn btn-secondary">
                  Отмена
                </button>
                <button type="submit" className="btn btn-primary" disabled={isSubmitting}>
                  {isSubmitting ? 'Сохранение...' : (editingContact ? 'Сохранить' : 'Добавить')}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  )
}