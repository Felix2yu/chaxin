// 总览页：统计卡片 + 监控中仓库 + 最近通知。

import { api } from '../api.js';
import { icon, escapeHtml, fmtShort, toast } from '../ui.js';

const STAT_DEFS = [
  { key: 'total', label: '仓库总数', icon: 'repos', color: 'var(--primary)', bg: 'var(--primary-soft)' },
  { key: 'monitored', label: '监控中', icon: 'eye', color: 'var(--success)', bg: 'var(--success-soft)' },
  { key: 'sent', label: '已发送通知', icon: 'check', color: 'var(--primary)', bg: 'var(--primary-soft)' },
  { key: 'failed', label: '发送失败', icon: 'alert', color: 'var(--danger)', bg: 'var(--danger-soft)' },
];

function statCard(def, value) {
  return `
    <div class="stat">
      <div class="stat__label">${def.label}</div>
      <div class="stat__value">${value}</div>
      <div class="stat__icon" style="background:${def.bg};color:${def.color}">${icon(def.icon)}</div>
      <div class="stat__bar" style="background:linear-gradient(90deg, ${def.color}, transparent)"></div>
    </div>`;
}

function statusBadge(status) {
  if (status === 'sent') return '<span class="badge badge--success"><span class="badge__dot"></span>已发送</span>';
  if (status === 'failed') return '<span class="badge badge--danger"><span class="badge__dot"></span>失败</span>';
  return `<span class="badge badge--muted">${escapeHtml(status)}</span>`;
}

function body(data) {
  const monitored = data.repos.filter((r) => r.monitored);
  const recent = data.notifications.slice(0, 6);

  const repoList =
    monitored.length === 0
      ? `<div class="empty"><div class="empty__icon">${icon('repos')}</div><div class="empty__title">暂无监控中的仓库</div></div>`
      : monitored
          .slice(0, 5)
          .map(
            (r) => `
        <a class="list__item is-link" href="${escapeHtml(r.html_url)}" target="_blank" rel="noopener">
          <div style="min-width:0;flex:1">
            <div class="repo-name" style="font-size:13px">${escapeHtml(r.full_name)}</div>
            <div style="font-size:12px;color:var(--text-muted);margin-top:2px;display:flex;align-items:center;gap:6px">
              ${icon('star')}<span style="width:14px;height:14px;display:inline-flex">${escapeHtml(r.stargazers_count)}</span>
              ${r.language ? `<span>&bull; ${escapeHtml(r.language)}</span>` : ''}
            </div>
          </div>
          <span style="color:var(--text-faint)">${icon('external')}</span>
        </a>`
          )
          .join('');

  const notifList =
    recent.length === 0
      ? `<div class="empty"><div class="empty__icon">${icon('notifications')}</div><div class="empty__title">暂无通知记录</div></div>`
      : recent
          .map(
            (n) => `
        <div class="list__item">
          <div style="min-width:0;flex:1">
            <div style="display:flex;align-items:center;gap:8px">
              <span class="repo-name" style="font-size:13px">${escapeHtml(n.full_name)}</span>
              ${n.tag ? `<span class="badge badge--muted">${escapeHtml(n.tag)}</span>` : ''}
            </div>
            <div style="font-size:12px;color:var(--text-muted);margin-top:2px">${fmtShort(n.released_at)}</div>
          </div>
          ${statusBadge(n.status)}
        </div>`
          )
          .join('');

  return `
    <div class="stat-grid">
      ${STAT_DEFS.map((d) => statCard(d, data.stats[d.key])).join('')}
    </div>
    <div class="two-col">
      <div class="panel">
        <div class="panel__head">
          <div class="panel__title">监控中的仓库</div>
          <a class="panel__link" href="#/repos">查看全部 &rarr;</a>
        </div>
        <ul class="list">${repoList}</ul>
      </div>
      <div class="panel">
        <div class="panel__head">
          <div class="panel__title">最近通知</div>
          <a class="panel__link" href="#/notifications">查看全部 &rarr;</a>
        </div>
        <ul class="list">${notifList}</ul>
      </div>
    </div>`;
}

function skeleton() {
  return `
    <div class="stat-grid">
      ${Array(4).fill('<div class="skeleton" style="height:108px;border-radius:14px"></div>').join('')}
    </div>
    <div class="skeleton" style="height:320px;border-radius:14px"></div>`;
}

export function render(container) {
  container.innerHTML = `
    <div class="content__inner">
      <div class="page-head">
        <h1>总览</h1>
        <p>监控仓库和通知的概览信息</p>
      </div>
      <div id="dash-body">${skeleton()}</div>
    </div>`;

  load(container);
}

async function load(container) {
  const bodyEl = container.querySelector('#dash-body');
  try {
    const [repos, notifications] = await Promise.all([
      api.listRepos(),
      api.listNotifications({ limit: 50 }),
    ]);
    const stats = {
      total: repos.length,
      monitored: repos.filter((r) => r.monitored).length,
      sent: notifications.filter((n) => n.status === 'sent').length,
      failed: notifications.filter((n) => n.status === 'failed').length,
    };
    bodyEl.innerHTML = body({ repos, notifications, stats });
  } catch (e) {
    toast(e.message || '加载失败', 'error');
    bodyEl.innerHTML = '';
  }
}
