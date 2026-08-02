export interface Settings {
  github_token: string
  shoutrrr_url: string
  poll_interval: string
  notify_on_first_run: boolean
  github_api_base_url: string
}

export interface Repo {
  id: number
  full_name: string
  owner: string
  name: string
  description: string
  language: string
  stargazers_count: number
  html_url: string
  monitored: boolean
  last_known_tag: string
  last_checked_at: string
  created_at: string
}

export interface Notification {
  id: number
  repo_id: number
  full_name: string
  tag: string
  release_url: string
  release_body: string
  released_at: string
  sent_at: string
  status: 'sent' | 'failed' | string
  error: string
}

export interface SettingsSaveResult {
  settings: Settings
  verify: {
    token_valid: boolean
    token_error?: string
    username?: string
  }
}

export interface SyncResult {
  started: boolean
}

export interface SyncStatus {
  running: boolean
  page: number
  total: number
  progress: number
  repos: number
  added: number
  updated: number
  skipped: number
  removed: number
  error: string
}

export interface BatchMonitorResult {
  updated: number
  monitored: boolean
}
