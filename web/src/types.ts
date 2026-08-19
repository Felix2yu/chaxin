export interface Settings {
  github_token: string
  shoutrrr_url: string
  poll_interval: string
  notify_on_first_run: boolean
  monitor_new_stars: boolean
  github_api_base_url: string
  max_notifications: number
  translate_engine: string
  translate_target_lang: string
  translate_url: string
  translate_api_key: string
  translate_model: string
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
  source: string
  ignore_pattern: string
}

export interface Notification {
  id: number
  repo_id: number
  full_name: string
  tag: string
  release_url: string
  release_body: string
  release_body_translated: string
  release_body_html: string
  release_body_translated_html: string
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

export interface TranslateResult {
  translated: boolean
  extracted: boolean
  text: string
  html: string
}

export interface Backup {
  version: number
  settings: Settings
  repos: Repo[]
}
