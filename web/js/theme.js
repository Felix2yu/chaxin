// 主题：跟随系统 / 浅色 / 深色，状态持久化到 localStorage。

const KEY = 'chaxin-theme';
const mq = window.matchMedia('(prefers-color-scheme: dark)');

export function getTheme() {
  const v = localStorage.getItem(KEY);
  return v === 'light' || v === 'dark' || v === 'system' ? v : 'system';
}

function resolve(mode) {
  if (mode === 'dark') return true;
  if (mode === 'light') return false;
  return mq.matches;
}

export function applyTheme(mode) {
  document.documentElement.classList.toggle('dark', resolve(mode));
}

export function setTheme(mode) {
  localStorage.setItem(KEY, mode);
  applyTheme(mode);
}

export function initTheme() {
  const mode = getTheme();
  applyTheme(mode);
  const handler = () => applyTheme(getTheme());
  if (mq.addEventListener) mq.addEventListener('change', handler);
  else mq.addListener(handler);
}
