import axios, { AxiosError, InternalAxiosRequestConfig } from 'axios'

const API_BASE_URL = '/api'

export const api = axios.create({
  baseURL: API_BASE_URL,
  headers: {
    'Content-Type': 'application/json',
  },
})

let isRefreshing = false
let refreshSubscribers: ((token: string) => void)[] = []

function subscribeTokenRefresh(callback: (token: string) => void) {
  refreshSubscribers.push(callback)
}

function onTokenRefreshed(token: string) {
  refreshSubscribers.forEach(callback => callback(token))
  refreshSubscribers = []
}

api.interceptors.request.use((config: InternalAxiosRequestConfig) => {
  const token = localStorage.getItem('access_token')
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

api.interceptors.response.use(
  (response) => response,
  async (error: AxiosError) => {
    const originalRequest = error.config as InternalAxiosRequestConfig & { _retry?: boolean }
    
    if (error.response?.status === 401 && !originalRequest._retry) {
      if (isRefreshing) {
        return new Promise((resolve) => {
          subscribeTokenRefresh((token: string) => {
            if (originalRequest.headers) {
              originalRequest.headers.Authorization = `Bearer ${token}`
            }
            resolve(api(originalRequest))
          })
        })
      }

      originalRequest._retry = true
      isRefreshing = true

      const refreshToken = localStorage.getItem('refresh_token')
      if (!refreshToken) {
        localStorage.removeItem('access_token')
        localStorage.removeItem('refresh_token')
        window.location.href = '/login'
        return Promise.reject(error)
      }

      try {
        const response = await axios.post(`${API_BASE_URL}/recipients/refresh`, {
          refresh_token: refreshToken,
        })

        const { access_token, refresh_token } = response.data
        localStorage.setItem('access_token', access_token)
        localStorage.setItem('refresh_token', refresh_token)

        if (originalRequest.headers) {
          originalRequest.headers.Authorization = `Bearer ${access_token}`
        }

        onTokenRefreshed(access_token)
        isRefreshing = false

        return api(originalRequest)
      } catch (refreshError) {
        isRefreshing = false
        localStorage.removeItem('access_token')
        localStorage.removeItem('refresh_token')
        window.location.href = '/login'
        return Promise.reject(refreshError)
      }
    }

    return Promise.reject(error)
  }
)

export interface AuthTokensResponse {
  user_id: string
  access_token: string
  refresh_token: string
}

export interface RegisterRequest {
  login: string
  password: string
  claim_contact_token?: string | null
}

export interface LoginRequest {
  login: string
  password: string
}

export interface RefreshRequest {
  refresh_token: string
}

export interface RecipientProfile {
  user_id: string
  login: string
  role: 'recipient_user' | 'source_operator' | 'admin'
}

export interface Contact {
  contact_id: string
  channel: 'email' | 'sms' | 'telegram'
  value: string
  enabled: boolean
  verified: boolean
  deleted_at: string | null
}

export interface ContactListResponse {
  items: Contact[]
  page: number
  size: number
  total: number
}

export interface CreateContactRequest {
  channel: 'email' | 'sms' | 'telegram'
  value: string
}

export interface UpdateContactRequest {
  value?: string
  enabled?: boolean
}

export interface Preference {
  preference_id: string
  contact_id: string
  sender_id: string | null
  scope_type: 'global' | 'sender' | 'campaign' | 'group'
  scope_id: string | null
  enabled: boolean
  quiet_from: string | null
  quiet_to: string | null
  blocked_channels: ('email' | 'sms' | 'telegram')[]
}

export interface PreferenceListResponse {
  items: Preference[]
  page: number
  size: number
  total: number
}

export interface PreferenceUpsertRequest {
  contact_id: string
  sender_id?: string | null
  scope_type: 'global' | 'sender' | 'campaign' | 'group'
  scope_id?: string | null
  enabled: boolean
  quiet_from?: string | null
  quiet_to?: string | null
  blocked_channels: ('email' | 'sms' | 'telegram')[]
}

export interface UnsubscribeRequest {
  token: string
  selected_scope: 'sender' | 'campaign_or_group'
}

export interface UnsubscribeResponse {
  contact_id: string
  applied_scope: 'sender' | 'campaign_or_group'
  sender_id: string | null
  campaign_id: string | null
  group_id: string | null
}

export interface Sender {
  sender_id: string
  name: string
  description: string | null
  active: boolean
}

export interface SenderListResponse {
  items: Sender[]
  page: number
  size: number
  total: number
}

export interface CreateSenderRequest {
  name: string
  description?: string | null
}

export interface Template {
  template_id: string
  sender_id: string
  name: string
  channel: 'email' | 'sms' | 'telegram'
  subject: string | null
  body: string
  variables: string[]
  active: boolean
}

export interface TemplateListResponse {
  items: Template[]
  page: number
  size: number
  total: number
}

export interface CreateTemplateRequest {
  sender_id: string
  name: string
  channel: 'email' | 'sms' | 'telegram'
  subject?: string | null
  body: string
  variables: string[]
}

export interface Group {
  group_id: string
  sender_id: string
  name: string
  active: boolean
}

export interface GroupListResponse {
  items: Group[]
  page: number
  size: number
  total: number
}

export interface CreateGroupRequest {
  sender_id: string
  name: string
}

export interface UpdateGroupRequest {
  name?: string
  active?: boolean
}

export interface Campaign {
  campaign_id: string
  sender_id: string
  group_id: string
  template_id: string
  name: string
  scheduled_at: string | null
  channels: ('email' | 'sms' | 'telegram')[]
  recurrence_rule: {
    kind: 'once' | 'daily' | 'weekly' | 'cron'
    value: string | null
  }
  active: boolean
}

export interface CampaignListResponse {
  items: Campaign[]
  page: number
  size: number
  total: number
}

export interface CreateCampaignRequest {
  sender_id: string
  group_id: string
  template_id: string
  name: string
  scheduled_at?: string | null
  channels: ('email' | 'sms' | 'telegram')[]
  recurrence_rule: {
    kind: 'once' | 'daily' | 'weekly' | 'cron'
    value?: string | null
  }
}

export interface NotificationHistoryItem {
  notification_id: string
  contact_id: string
  channel: 'email' | 'sms' | 'telegram'
  status: 'draft' | 'scheduled' | 'queued' | 'processing' | 'sent' | 'delivered' | 'failed' | 'cancelled' | 'skipped_by_preference'
  sender_id: string
  campaign_id: string | null
  created_at: string
  delivered_at: string | null
}

export interface NotificationHistoryResponse {
  items: NotificationHistoryItem[]
  page: number
  size: number
  total: number
}

export interface NotificationDetails extends NotificationHistoryItem {
  group_id: string | null
  subject: string | null
  rendered_content: string
  unsubscribe_url: string | null
}

export interface SenderAnalyticsResponse {
  sender_id: string
  from: string | null
  to: string | null
  totals: {
    queued: number
    delivered: number
    failed: number
    skipped_by_preference: number
  }
}

export interface ApiError {
  error: {
    code: string
    message: string
    details: Record<string, unknown>
    request_id: string
  }
}

export function isApiError(error: unknown): error is ApiError {
  return typeof error === 'object' && error !== null && 'error' in error
}

export function getErrorMessage(error: unknown): string {
  if (isApiError(error)) {
    return error.error.message
  }
  if (axios.isAxiosError(error)) {
    return error.response?.data?.error?.message || error.message
  }
  return 'Произошла ошибка. Попробуйте позже.'
}

export const recipientsApi = {
  register: (data: RegisterRequest) => 
    api.post<AuthTokensResponse>('/recipients/register', data),
  
  login: (data: LoginRequest) => 
    api.post<AuthTokensResponse>('/recipients/login', data),
  
  refresh: (data: RefreshRequest) => 
    api.post<AuthTokensResponse>('/recipients/refresh', data),
  
  getMe: () => 
    api.get<RecipientProfile>('/recipients/me'),
  
  getContacts: (page = 1, size = 20) => 
    api.get<ContactListResponse>('/recipients/contacts', { params: { page, size } }),
  
  createContact: (data: CreateContactRequest) => 
    api.post<Contact>('/recipients/contacts', data),
  
  updateContact: (contactId: string, data: UpdateContactRequest) => 
    api.put<Contact>(`/recipients/contacts/${contactId}`, data),
  
  deleteContact: (contactId: string) => 
    api.delete(`/recipients/contacts/${contactId}`),
  
  getPreferences: (page = 1, size = 20) => 
    api.get<PreferenceListResponse>('/recipients/preferences', { params: { page, size } }),
  
  upsertPreference: (data: PreferenceUpsertRequest) => 
    api.put<Preference>('/recipients/preferences', data),
  
  unsubscribe: (data: UnsubscribeRequest) => 
    api.post<UnsubscribeResponse>('/recipients/unsubscribe', data),
}

export const sourcesApi = {
  getSenders: (page = 1, size = 20) => 
    api.get<SenderListResponse>('/sources/senders', { params: { page, size } }),
  
  createSender: (data: CreateSenderRequest) => 
    api.post<Sender>('/sources/senders', data),
  
  getTemplates: (page = 1, size = 20, senderId?: string) => 
    api.get<TemplateListResponse>('/sources/templates', { params: { page, size, sender_id: senderId } }),
  
  createTemplate: (data: CreateTemplateRequest) => 
    api.post<Template>('/sources/templates', data),
  
  updateTemplate: (templateId: string, data: CreateTemplateRequest) => 
    api.put<Template>(`/sources/templates/${templateId}`, data),
  
  deleteTemplate: (templateId: string) => 
    api.delete(`/sources/templates/${templateId}`),
  
  getGroups: (page = 1, size = 20, senderId?: string) => 
    api.get<GroupListResponse>('/sources/groups', { params: { page, size, sender_id: senderId } }),
  
  createGroup: (data: CreateGroupRequest) => 
    api.post<Group>('/sources/groups', data),
  
  updateGroup: (groupId: string, data: UpdateGroupRequest) => 
    api.put<Group>(`/sources/groups/${groupId}`, data),
  
  upsertGroupMembers: (groupId: string, contactIds: string[]) => 
    api.post(`/sources/groups/${groupId}/members`, { contact_ids: contactIds }),
  
  getCampaigns: (page = 1, size = 20, senderId?: string) => 
    api.get<CampaignListResponse>('/sources/campaigns', { params: { page, size, sender_id: senderId } }),
  
  createCampaign: (data: CreateCampaignRequest) => 
    api.post<Campaign>('/sources/campaigns', data),
  
  updateCampaign: (campaignId: string, data: CreateCampaignRequest) => 
    api.put<Campaign>(`/sources/campaigns/${campaignId}`, data),
}

export const notificationsApi = {
  getHistory: (params: {
    mode?: 'recipient' | 'operator'
    sender_id?: string
    campaign_id?: string
    status?: string
    page?: number
    size?: number
  }) => 
    api.get<NotificationHistoryResponse>('/notifications/history', { params }),
  
  getNotification: (notificationId: string) => 
    api.get<NotificationDetails>(`/notifications/${notificationId}`),
  
  getAnalytics: (senderId: string, from?: string, to?: string) => 
    api.get<SenderAnalyticsResponse>('/notifications/analytics', { 
      params: { sender_id: senderId, from, to } 
    }),
}