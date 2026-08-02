import type { Backup, BatchMonitorResult, Notification, Repo, Settings, SettingsSaveResult, SyncResult, SyncStatus } from './types'

async function request<T>(path: string, options: RequestInit = {}): Promise<T> {
  const res = await fetch(`/api${path}`, {
    headers: { 'Content-Type': 'application/json' },
    ...options,
  })
  if (!res.ok) {
    let msg = `请求失败 (${res.status})`
    try {
      const data = await res.json()
      if (data && data.error) msg = data.error
    } catch {
      /* ignore */
    }
    throw new Error(msg)
  }
  if (res.status === 204) return undefined as T
  return res.json() as Promise<T>
}

export const api = {
  health: () => request<{ status: string; time: string }>('/health'),

  getSettings: () => request<Settings>('/settings'),
  saveSettings: (s: Settings) =>
    request<SettingsSaveResult>('/settings', { method: 'PUT', body: JSON.stringify(s) }),

  listRepos: (params?: { query?: string; language?: string; monitored?: string }) => {
    const q = new URLSearchParams()
    if (params?.query) q.set('query', params.query)
    if (params?.language) q.set('language', params.language)
    if (params?.monitored !== undefined) q.set('monitored', params.monitored)
    const suffix = q.toString() ? `?${q.toString()}` : ''
    return request<Repo[]>(`/repos${suffix}`)
  },
  addRepo: (fullName: string) =>
    request<{ full_name: string }>('/repos', { method: 'POST', body: JSON.stringify({ full_name: fullName }) }),
  setMonitored: (id: number, monitored: boolean, ignorePattern?: string) => {
    const body: { monitored: boolean; ignore_pattern?: string } = { monitored }
    if (ignorePattern !== undefined) body.ignore_pattern = ignorePattern
    return request<{ monitored: boolean }>(`/repos/${id}`, {
      method: 'PATCH',
      body: JSON.stringify(body),
    })
  },
  batchMonitor: (ids: number[], monitored: boolean) =>
    request<BatchMonitorResult>('/repos/batch-monitor', {
      method: 'POST',
      body: JSON.stringify({ ids, monitored }),
    }),
  deleteRepo: (id: number) => request<void>(`/repos/${id}`, { method: 'DELETE' }),
  syncStars: () => request<SyncResult>('/repos/sync-stars', { method: 'POST' }),
  syncStarsStatus: () => request<SyncStatus>('/repos/sync-stars/status'),

  listNotifications: (params?: { limit?: number; query?: string; status?: string }) => {
    const q = new URLSearchParams()
    if (params?.limit) q.set('limit', String(params.limit))
    if (params?.query) q.set('query', params.query)
    if (params?.status) q.set('status', params.status)
    const suffix = q.toString() ? `?${q.toString()}` : ''
    return request<Notification[]>(`/notifications${suffix}`)
  },

  retryNotification: (id: number) =>
    request<{ status: string }>(`/notifications/${id}/retry`, { method: 'POST' }),

  backup: () => request<Backup>('/backup'),
  restore: (b: Backup) =>
    request<{ status: string }>('/restore', { method: 'POST', body: JSON.stringify(b) }),

  testNotification: (title: string, message: string) =>
    request<{ status: string }>('/test-notification', {
      method: 'POST',
      body: JSON.stringify({ title, message }),
    }),

  runMonitor: () => request<{ status: string }>('/monitor/run', { method: 'POST' }),
}
