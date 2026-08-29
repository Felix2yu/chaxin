// 通知中心：搜索 / 状态筛选、展开查看 Release Notes、按需翻译、失败重试。

import { api } from '../api.js';
import { icon, escapeHtml, fmtFull, toast, debounce } from '../ui.js';

const state = {
  notifications: [],
  loading: true,
  search: '',
  statusFilter: '',
  expanded: new Set(),
  translating: new Set(),
  cache: new Map(), // id -> {html, text}
};

function statusBadge(status) {
  if (status === 'sent')
    return '<span class="badge badge--success"><span class="badge__dot"></span>已发送</span>';
  if (status === 'failed')
    return '<span class="badge badge--danger"><span class="badge__dot"></span>失败</span>';
  return `<span class="badge badge--muted">${escapeHtml(status)}</span>`;
}

function translatedHtml(n) {
  const c = state.cache.get(n.id);
  if (c && c.html) return c.html;
  return n.release_body_translated_html || '';
}

function itemHtml(n) {
  const open = state.expanded.has(n.id);
  const cached = state.cache.has(n.id);
  const translating = state.translating.has(n.id);
  const tHtml = translatedHtml(n);
  const hasTrans = !!tHtml;

  const errorBlock = n.error
    ? `<div class="verify verify--err" style="margin-bottom:12px"><strong>错误信息</strong><div style="margin-top:4px">${escapeHtml(
        n.error
      )}</div></div>`
    : '';

  const transBlock = hasTrans
    ? `<div class="notif__translated">
         <div class="notif__note-title">中文翻译</div>
         <div class="md">${tHtml}</div>
       </div>`
    : '';

  const originalBlock = n.release_body_html
    ? `<div class="md">${n.release_body_html}</div>`
    : '<p style="color:var(--text-muted);font-style:italic">无 Release Notes</p>';

  const translateBtn = n.release_body
    ? `<button class="panel__link" data-action="translate" data-id="${n.id}" ${
        translating || cached ? 'disabled' : ''
      } style="border:none;background:none;cursor:pointer">${
        translating ? '翻译中…' : cached ? '已翻译' : '翻译为中文'
      }</button>`
    : '';

  const actions = `
    ${
      n.release_url
        ? `<a class="panel__link" href="${escapeHtml(n.release_url)}" target="_blank" rel="noopener" style="display:inline-flex;align-items:center;gap:4px">${icon(
            'external'
          )}查看 Release</a>`
        : ''
    }
    ${
      n.status === 'failed'
        ? `<button class="panel__link" data-action="retry" data-id="${n.id}" style="border:none;background:none;cursor:pointer;display:inline-flex;align-items:center;gap:4px;color:var(--warning)">${icon(
            'sync'
          )}重试发送</button>`
        : ''
    }`;

  return `
    <div class="notif">
      <div class="notif__row" data-action="toggle" data-id="${n.id}">
        <span class="badge__dot" style="background:${
          n.status === 'sent' ? 'var(--success)' : n.status === 'failed' ? 'var(--danger)' : 'var(--warning)'
        }"></span>
        <div style="min-width:0;flex:1">
          <div style="display:flex;align-items:center;gap:8px">
            <span class="repo-name" style="font-size:13px">${escapeHtml(n.full_name)}</span>
            ${n.tag ? `<span class="badge badge--muted">${escapeHtml(n.tag)}</span>` : ''}
          </div>
          <div style="font-size:12px;color:var(--text-muted);margin-top:2px">
            发布: ${fmtFull(n.released_at)}${n.sent_at ? ` · 通知: ${fmtFull(n.sent_at)}` : ''}
          </div>
        </div>
        ${statusBadge(n.status)}
        <span style="color:var(--text-faint);transition:transform .2s;${
          open ? 'transform:rotate(180deg)' : ''
        }">${icon('chevron')}</span>
      </div>
      ${
        open
          ? `<div class="notif__body">
              <div class="notif__note">
                ${errorBlock}
                <div style="display:flex;align-items:center;justify-content:space-between;margin-bottom:8px">
                  <div class="notif__note-title" style="margin:0">Release Notes</div>
                  ${translateBtn}
                </div>
                ${transBlock}
                ${originalBlock}
                <div style="display:flex;gap:16px;margin-top:14px;padding-top:12px;border-top:1px solid var(--border)">
                  ${actions}
                </div>
              </div>
            </div>`
          : ''
      }
    </div>`;
}

function listHtml() {
  if (state.loading)
    return `<div style="padding:16px;display:flex;flex-direction:column;gap:12px">${Array(8)
      .fill('<div class="skeleton" style="height:56px;border-radius:12px"></div>')
      .join('')}</div>`;
  if (state.notifications.length === 0)
    return `<div class="empty">
      <div class="empty__icon">${icon('notifications')}</div>
      <div class="empty__title">暂无通知</div>
      <div class="empty__hint">当监控仓库有新版本发布时将出现在这里</div>
    </div>`;
  return state.notifications.map(itemHtml).join('');
}

function shell() {
  return `
    <div class="content__inner">
      <div class="page-head">
        <h1>通知中心</h1>
        <p>查看版本发布通知记录</p>
      </div>
      <div class="toolbar">
        <div class="input-group">
          ${icon('search')}
          <input class="input" id="notif-search" placeholder="搜索仓库名…" value="${escapeHtml(
            state.search
          )}" />
        </div>
        <select class="input" id="notif-status" style="width:auto">
          <option value="">所有状态</option>
          <option value="sent">已发送</option>
          <option value="failed">失败</option>
        </select>
        <span style="font-size:12px;color:var(--text-muted);margin-left:auto" id="notif-count"></span>
      </div>
      <div class="card" id="notif-list">${listHtml()}</div>
    </div>`;
}

const doSearch = debounce(() => updateList(), 350);

async function updateList() {
  state.loading = true;
  renderList();
  try {
    const params = { limit: 200 };
    if (state.search) params.query = state.search;
    if (state.statusFilter) params.status = state.statusFilter;
    state.notifications = await api.listNotifications(params);
  } catch (e) {
    toast(e.message || '加载失败', 'error');
    state.notifications = [];
  } finally {
    state.loading = false;
    renderList();
  }
}

function renderList() {
  const el = document.getElementById('notif-list');
  if (el) el.innerHTML = listHtml();
  const count = document.getElementById('notif-count');
  if (count) count.textContent = `共 ${state.notifications.length} 条记录`;
}

function toggle(id) {
  if (state.expanded.has(id)) state.expanded.delete(id);
  else state.expanded.add(id);
  renderList();
}

async function doTranslate(id) {
  const n = state.notifications.find((x) => x.id === id);
  if (!n || !n.release_body || state.translating.has(id)) return;
  state.translating.add(id);
  renderList();
  try {
    const res = await api.translate(n.release_body, 'zh');
    state.cache.set(id, { html: res.html, text: res.text });
  } catch (e) {
    toast(e.message || '翻译失败', 'error');
  } finally {
    state.translating.delete(id);
    renderList();
  }
}

async function doRetry(id) {
  try {
    await api.retryNotification(id);
    toast('已重试', 'success');
    await updateList();
  } catch (e) {
    toast(e.message || '重试失败', 'error');
  }
}

function onClick(e) {
  const t = e.target.closest('[data-action]');
  if (!t) return;
  const action = t.getAttribute('data-action');
  const id = Number(t.getAttribute('data-id'));
  if (action === 'toggle') toggle(id);
  else if (action === 'translate') doTranslate(id);
  else if (action === 'retry') doRetry(id);
}

function onInput(e) {
  if (e.target.id === 'notif-search') {
    state.search = e.target.value;
    doSearch();
  } else if (e.target.id === 'notif-status') {
    state.statusFilter = e.target.value;
    updateList();
  }
}

let root = null;

export function render(container) {
  root = container;
  container.innerHTML = shell();
  const statusSel = document.getElementById('notif-status');
  if (statusSel) statusSel.value = state.statusFilter;
  container.addEventListener('click', onClick);
  container.addEventListener('input', onInput);
  updateList();
}

export function cleanup() {
  if (root) {
    root.removeEventListener('click', onClick);
    root.removeEventListener('input', onInput);
    root = null;
  }
}
