// API 客户端：封装 /api 下的所有接口。与后端 REST 契约严格一致。

async function request(path, options = {}) {
  const res = await fetch('/api' + path, {
    headers: { 'Content-Type': 'application/json' },
    ...options,
  });
  if (!res.ok) {
    let msg = `请求失败 (${res.status})`;
    try {
      const data = await res.json();
      if (data && data.error) msg = data.error;
    } catch (_) {
      /* 忽略解析错误 */
    }
    throw new Error(msg);
  }
  if (res.status === 204) return undefined;
  return res.json();
}

export const api = {
  health: () => request('/health'),

  getSettings: () => request('/settings'),
  saveSettings: (s) =>
    request('/settings', { method: 'PUT', body: JSON.stringify(s) }),

  // monitored 传 true/false，转换为后端接受的 "1"/"0"
  listRepos: (params = {}) => {
    const q = new URLSearchParams();
    if (params.query) q.set('query', params.query);
    if (params.language) q.set('language', params.language);
    if (typeof params.monitored === 'boolean')
      q.set('monitored', params.monitored ? '1' : '0');
    const suffix = q.toString() ? '?' + q.toString() : '';
    return request('/repos' + suffix);
  },
  addRepo: (fullName) =>
    request('/repos', {
      method: 'POST',
      body: JSON.stringify({ full_name: fullName }),
    }),
  setMonitored: (id, monitored, ignorePattern, trackTags) => {
    const body = { monitored };
    if (ignorePattern !== undefined) body.ignore_pattern = ignorePattern;
    if (trackTags !== undefined) body.track_tags = trackTags;
    return request('/repos/' + id, {
      method: 'PATCH',
      body: JSON.stringify(body),
    });
  },
  batchMonitor: (ids, monitored) =>
    request('/repos/batch-monitor', {
      method: 'POST',
      body: JSON.stringify({ ids, monitored }),
    }),
  deleteRepo: (id) => request('/repos/' + id, { method: 'DELETE' }),

  syncStars: () => request('/repos/sync-stars', { method: 'POST' }),
  syncStarsStatus: () => request('/repos/sync-stars/status'),

  listNotifications: (params = {}) => {
    const q = new URLSearchParams();
    if (params.limit) q.set('limit', String(params.limit));
    if (params.query) q.set('query', params.query);
    if (params.status) q.set('status', params.status);
    const suffix = q.toString() ? '?' + q.toString() : '';
    return request('/notifications' + suffix);
  },
  retryNotification: (id) =>
    request('/notifications/' + id + '/retry', { method: 'POST' }),

  translate: (text, targetLang, engine) =>
    request('/translate', {
      method: 'POST',
      body: JSON.stringify({
        text,
        target_lang: targetLang,
        engine,
      }),
    }),

  testNotification: (title, message) =>
    request('/test-notification', {
      method: 'POST',
      body: JSON.stringify({ title, message }),
    }),

  backup: () => request('/backup'),
  restore: (b) =>
    request('/restore', { method: 'POST', body: JSON.stringify(b) }),

  runMonitor: () => request('/monitor/run', { method: 'POST' }),
};
