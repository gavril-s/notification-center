import { useState, useEffect } from 'react'
import { sourcesApi, Campaign, CreateCampaignRequest, Sender, Template, Group, getErrorMessage } from '../api/client'
import './Campaigns.css'

const channelLabels: Record<string, string> = {
  email: 'Email',
  sms: 'SMS',
  telegram: 'Telegram',
}

const recurrenceLabels: Record<string, string> = {
  once: 'Однократно',
  daily: 'Ежедневно',
  weekly: 'Еженедельно',
  cron: 'По расписанию (cron)',
}

export default function Campaigns() {
  const [campaigns, setCampaigns] = useState<Campaign[]>([])
  const [senders, setSenders] = useState<Sender[]>([])
  const [templates, setTemplates] = useState<Template[]>([])
  const [groups, setGroups] = useState<Group[]>([])
  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState('')
  const [showModal, setShowModal] = useState(false)
  const [editingCampaign, setEditingCampaign] = useState<Campaign | null>(null)
  const [formData, setFormData] = useState<CreateCampaignRequest>({
    sender_id: '',
    group_id: '',
    template_id: '',
    name: '',
    scheduled_at: null,
    channels: ['email'],
    recurrence_rule: { kind: 'once', value: null },
  })
  const [formError, setFormError] = useState('')
  const [isSubmitting, setIsSubmitting] = useState(false)
  const [selectedSender, setSelectedSender] = useState<string>('')

  const loadData = async () => {
    try {
      const [campaignsResponse, sendersResponse, templatesResponse, groupsResponse] = await Promise.all([
        sourcesApi.getCampaigns(1, 100, selectedSender || undefined),
        sourcesApi.getSenders(1, 100),
        sourcesApi.getTemplates(1, 100),
        sourcesApi.getGroups(1, 100),
      ])
      setCampaigns(campaignsResponse.data.items)
      setSenders(sendersResponse.data.items)
      setTemplates(templatesResponse.data.items)
      setGroups(groupsResponse.data.items)
      
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

  const filteredTemplates = templates.filter(t => !selectedSender || t.sender_id === selectedSender)
  const filteredGroups = groups.filter(g => !selectedSender || g.sender_id === selectedSender)

  const handleCreate = () => {
    setEditingCampaign(null)
    setFormData({
      sender_id: senders[0]?.sender_id || '',
      group_id: filteredGroups[0]?.group_id || '',
      template_id: filteredTemplates[0]?.template_id || '',
      name: '',
      scheduled_at: null,
      channels: ['email'],
      recurrence_rule: { kind: 'once', value: null },
    })
    setFormError('')
    setShowModal(true)
  }

  const handleEdit = (campaign: Campaign) => {
    setEditingCampaign(campaign)
    setFormData({
      sender_id: campaign.sender_id,
      group_id: campaign.group_id,
      template_id: campaign.template_id,
      name: campaign.name,
      scheduled_at: campaign.scheduled_at,
      channels: campaign.channels,
      recurrence_rule: campaign.recurrence_rule,
    })
    setFormError('')
    setShowModal(true)
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setFormError('')
    setIsSubmitting(true)

    try {
      if (editingCampaign) {
        await sourcesApi.updateCampaign(editingCampaign.campaign_id, formData)
      } else {
        await sourcesApi.createCampaign(formData)
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

  const getTemplateName = (templateId: string) => {
    const template = templates.find(t => t.template_id === templateId)
    return template?.name || templateId
  }

  const getGroupName = (groupId: string) => {
    const group = groups.find(g => g.group_id === groupId)
    return group?.name || groupId
  }

  if (isLoading) {
    return (
      <div className="loading-screen">
        <div className="spinner"></div>
        <p>Загрузка рассылок...</p>
      </div>
    )
  }

  return (
    <div className="campaigns-page">
      <div className="card">
        <div className="card-header">
          <h2 className="card-title">Рассылки</h2>
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
              Создать рассылку
            </button>
          </div>
        </div>

        {error && <div className="alert alert-error">{error}</div>}

        {senders.length === 0 ? (
          <div className="empty-state">
            <div className="empty-state-icon">📨</div>
            <p>Сначала создайте отправителя</p>
          </div>
        ) : campaigns.length === 0 ? (
          <div className="empty-state">
            <div className="empty-state-icon">📨</div>
            <p>У вас пока нет рассылок</p>
            <button onClick={handleCreate} className="btn btn-primary" style={{ marginTop: '1rem' }}>
              Создать первую рассылку
            </button>
          </div>
        ) : (
          <table className="table">
            <thead>
              <tr>
                <th>Название</th>
                <th>Отправитель</th>
                <th>Шаблон</th>
                <th>Группа</th>
                <th>Канал</th>
                <th>Повтор</th>
                <th>Статус</th>
                <th>Действия</th>
              </tr>
            </thead>
            <tbody>
              {campaigns.map(campaign => (
                <tr key={campaign.campaign_id}>
                  <td><strong>{campaign.name}</strong></td>
                  <td>{getSenderName(campaign.sender_id)}</td>
                  <td>{getTemplateName(campaign.template_id)}</td>
                  <td>{getGroupName(campaign.group_id)}</td>
                  <td>{channelLabels[campaign.channels[0]]}</td>
                  <td>{recurrenceLabels[campaign.recurrence_rule.kind]}</td>
                  <td>
                    {campaign.active ? (
                      <span className="badge badge-success">Активна</span>
                    ) : (
                      <span className="badge badge-neutral">Неактивна</span>
                    )}
                  </td>
                  <td>
                    <div className="table-actions">
                      <button onClick={() => handleEdit(campaign)} className="btn btn-secondary btn-sm">
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
                {editingCampaign ? 'Изменить рассылку' : 'Создать рассылку'}
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
                    onChange={e => {
                      setFormData({ 
                        ...formData, 
                        sender_id: e.target.value,
                        template_id: '',
                        group_id: '',
                      })
                    }}
                    required
                    disabled={!!editingCampaign}
                  >
                    {senders.map(sender => (
                      <option key={sender.sender_id} value={sender.sender_id}>
                        {sender.name}
                      </option>
                    ))}
                  </select>
                </div>

                <div className="form-group">
                  <label className="form-label">Название рассылки</label>
                  <input
                    type="text"
                    className="form-input"
                    value={formData.name}
                    onChange={e => setFormData({ ...formData, name: e.target.value })}
                    required
                    placeholder="Например: Еженедельная рассылка"
                  />
                </div>

                <div className="form-group">
                  <label className="form-label">Шаблон</label>
                  <select
                    className="form-select"
                    value={formData.template_id}
                    onChange={e => setFormData({ ...formData, template_id: e.target.value })}
                    required
                    disabled={!!editingCampaign}
                  >
                    <option value="">Выберите шаблон</option>
                    {filteredTemplates.map(template => (
                      <option key={template.template_id} value={template.template_id}>
                        {template.name} ({channelLabels[template.channel]})
                      </option>
                    ))}
                  </select>
                </div>

                <div className="form-group">
                  <label className="form-label">Группа получателей</label>
                  <select
                    className="form-select"
                    value={formData.group_id}
                    onChange={e => setFormData({ ...formData, group_id: e.target.value })}
                    required
                    disabled={!!editingCampaign}
                  >
                    <option value="">Выберите группу</option>
                    {filteredGroups.map(group => (
                      <option key={group.group_id} value={group.group_id}>
                        {group.name}
                      </option>
                    ))}
                  </select>
                </div>

                <div className="form-group">
                  <label className="form-label">Канал</label>
                  <select
                    className="form-select"
                    value={formData.channels[0]}
                    onChange={e => setFormData({ ...formData, channels: [e.target.value as any] })}
                    disabled={!!editingCampaign}
                  >
                    <option value="email">Email</option>
                    <option value="sms">SMS</option>
                    <option value="telegram">Telegram</option>
                  </select>
                </div>

                <div className="form-group">
                  <label className="form-label">Повторение</label>
                  <select
                    className="form-select"
                    value={formData.recurrence_rule.kind}
                    onChange={e => setFormData({ 
                      ...formData, 
                      recurrence_rule: { 
                        ...formData.recurrence_rule, 
                        kind: e.target.value as any 
                      } 
                    })}
                  >
                    <option value="once">Однократно</option>
                    <option value="daily">Ежедневно</option>
                    <option value="weekly">Еженедельно</option>
                    <option value="cron">По расписанию (cron)</option>
                  </select>
                </div>

                <div className="form-group">
                  <label className="form-label">Запланировать на</label>
                  <input
                    type="datetime-local"
                    className="form-input"
                    value={formData.scheduled_at ? formData.scheduled_at.slice(0, 16) : ''}
                    onChange={e => setFormData({ 
                      ...formData, 
                      scheduled_at: e.target.value ? new Date(e.target.value).toISOString() : null 
                    })}
                  />
                </div>
              </div>
              <div className="modal-footer">
                <button type="button" onClick={() => setShowModal(false)} className="btn btn-secondary">
                  Отмена
                </button>
                <button type="submit" className="btn btn-primary" disabled={isSubmitting}>
                  {isSubmitting ? 'Сохранение...' : (editingCampaign ? 'Сохранить' : 'Создать')}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  )
}