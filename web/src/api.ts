import type { Notification, Repo, Settings, SettingsSaveResult, SyncResult } from './types'

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
  setMonitored: (id: number, monitored: boolean) =>
    request<{ monitored: boolean }>(`/repos/${id}`, {
      method: 'PATCH',
      body: JSON.stringify({ monitored }),
    }),
  deleteRepo: (id: number) => request<void>(`/repos/${id}`, { method: 'DELETE' }),
  syncStars: () => request<SyncResult>('/repos/sync-stars', { method: 'POST' }),

  listNotifications: (limit = 50) => request<Notification[]>(`/notifications?limit=${limit}`),

  testNotification: (title: string, message: string) =>
    request<{ status: string }>('/test-notification', {
      method: 'POST',
      body: JSON.stringify({ title, message }),
    }),

  runMonitor: () => request<{ status: string }>('/monitor/run', { method: 'POST' }),
}
