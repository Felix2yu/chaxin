export type ThemeMode = 'system' | 'light' | 'dark'

const STORAGE_KEY = 'chaxin-theme'

const media = window.matchMedia('(prefers-color-scheme: dark)')

function resolve(mode: ThemeMode): boolean {
  if (mode === 'dark') return true
  if (mode === 'light') return false
  return media.matches
}

export function applyTheme(mode: ThemeMode) {
  document.documentElement.classList.toggle('dark', resolve(mode))
}

export function getTheme(): ThemeMode {
  const v = localStorage.getItem(STORAGE_KEY)
  if (v === 'light' || v === 'dark' || v === 'system') return v
  return 'system'
}

export function setTheme(mode: ThemeMode) {
  localStorage.setItem(STORAGE_KEY, mode)
  applyTheme(mode)
}

export function initTheme() {
  const mode = getTheme()
  applyTheme(mode)
  media.addEventListener('change', () => applyTheme(mode))
}
