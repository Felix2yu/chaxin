// 仓库管理：搜索 / 语言 / 监控状态筛选、批量监控、同步 Stars、添加 / 编辑 / 删除。

import { api } from '../api.js';
import { icon, escapeHtml, fmtShort, toast, debounce } from '../ui.js';

const state = {
  repos: [],
  loading: false,
  search: '',
  lang: '',
  monitored: '',
  selected: new Set(),
  syncing: false,
  sync: null, // SyncStatus
  poll: null,
  showAdd: false,
  showEdit: false,
  editRepo: null,
  newName: '',
  adding: false,
  editing: false,
};

function monitoredPill(on) {
  return `<span class="monitored-pill ${on ? 'is-on' : 'is-off'}">
    <span class="badge__dot"></span>${on ? '监控中' : '未监控'}</span>`;
}

function skeletonRows() {
  return `<div style="padding:16px;display:flex;flex-direction:column;gap:12px">
    ${Array(8).fill('<div class="skeleton" style="height:40px;border-radius:10px"></div>').join('')}
  </div>`;
}

function emptyState() {
  return `<div class="empty">
    <div class="empty__icon">${icon('repos')}</div>
    <div class="empty__title">暂无仓库</div>
    <div class="empty__hint">点击「添加仓库」或「同步 Stars」开始</div>
  </div>`;
}

function rowHtml(r) {
  const selected = state.selected.has(r.id);
  const lang = r.language
    ? `<span><span class="lang-dot"></span>${escapeHtml(r.language)}</span>`
    : '<span style="color:var(--text-faint)">-</span>';
  return `
    <tr>
      <td style="width:44px">
        <input type="checkbox" data-action="select" data-id="${r.id}" ${
    selected ? 'checked' : ''
  } style="width:16px;height:16px;accent-color:var(--primary);cursor:pointer" />
      </td>
      <td>
        <a class="repo-name" href="${escapeHtml(r.html_url)}" target="_blank" rel="noopener">${escapeHtml(
    r.full_name
  )}</a>
        ${r.description ? `<div class="repo-desc">${escapeHtml(r.description)}</div>` : ''}
      </td>
      <td style="display:none" data-col="lang">${lang}</td>
      <td data-col="stars">
        <span style="display:inline-flex;align-items:center;gap:4px;color:var(--text-muted);font-size:12px">
          ${icon('star')}<span style="width:14px;height:14px;display:inline-flex">${escapeHtml(
    r.stargazers_count
  )}</span>
        </span>
      </td>
      <td data-col="checked"><span style="font-size:12px;color:var(--text-muted)">${fmtShort(
        r.last_checked_at
      )}</span></td>
      <td>
        <div style="display:flex;align-items:center;gap:6px">
          <button class="monitored-pill-wrap" data-action="toggle-monitor" data-id="${r.id}" style="border:none;background:none;padding:0;cursor:pointer">
            ${monitoredPill(r.monitored)}
          </button>
          ${r.track_tags ? '<span class="badge badge--muted" title="无 Release 时回退用 tag 监控">tag</span>' : ''}
        </div>
      </td>
      <td style="width:80px">
        <div class="row-actions">
          <button class="icon-btn" data-action="edit-open" data-id="${r.id}" title="编辑忽略规则">${icon(
    'edit'
  )}</button>
          <button class="icon-btn icon-btn--danger" data-action="delete" data-id="${r.id}" title="取消星标并删除">${icon(
    'trash'
  )}</button>
        </div>
      </td>
    </tr>`;
}

function tableHtml() {
  if (state.loading) return skeletonRows();
  if (state.repos.length === 0) return emptyState();
  const allSelected =
    state.repos.length > 0 && state.repos.every((r) => state.selected.has(r.id));
  return `
    <table class="tbl">
      <thead>
        <tr>
          <th style="width:44px"><input type="checkbox" data-action="select-all" ${
            allSelected ? 'checked' : ''
          } style="width:16px;height:16px;accent-color:var(--primary);cursor:pointer" /></th>
          <th>仓库</th>
          <th style="display:none" data-col="lang">语言</th>
          <th data-col="stars">Stars</th>
          <th data-col="checked">最近检查</th>
          <th>监控</th>
          <th style="width:80px">操作</th>
        </tr>
      </thead>
      <tbody>${state.repos.map(rowHtml).join('')}</tbody>
    </table>`;
}

function batchBarHtml() {
  if (state.selected.size === 0) return '';
  return `
    <div class="badge badge--muted" style="border-radius:999px">已选 ${state.selected.size}</div>
    <button class="btn btn--sm" data-action="batch-on">批量监控</button>
    <button class="btn btn--sm" data-action="batch-off">取消监控</button>`;
}

function syncBoxHtml() {
  if (!state.syncing || !state.sync) return '';
  const s = state.sync;
  const pct = Math.round((s.progress || 0) * 100);
  return `
    <div class="sync-box">
      <div class="sync-box__row">
        <span>正在同步 Stars…</span>
        <span>${pct}%</span>
      </div>
      <div class="progress"><div class="progress__bar" style="width:${pct}%"></div></div>
      <div class="sync-box__row" style="margin:6px 0 0">
        <span>已处理 ${s.repos} 个仓库</span>
        <span>新增 ${s.added} · 更新 ${s.updated} · 移除 ${s.removed}</span>
      </div>
      ${s.error ? `<div class="verify verify--err" style="margin-top:8px">${escapeHtml(s.error)}</div>` : ''}
    </div>`;
}

function langOptionsHtml() {
  const langs = Array.from(
    new Set(state.repos.map((r) => r.language).filter(Boolean))
  ).sort();
  const opts = ['<option value="">所有语言</option>'].concat(
    langs.map((l) => `<option value="${escapeHtml(l)}">${escapeHtml(l)}</option>`)
  );
  return opts.join('');
}

function shell() {
  return `
    <div class="content__inner">
      <div class="page-head" style="display:flex;justify-content:space-between;align-items:flex-start;gap:16px;flex-wrap:wrap">
        <div>
          <h1>仓库管理</h1>
          <p>管理监控的 GitHub 仓库</p>
        </div>
        <div style="display:flex;gap:10px">
          <button class="btn" data-action="sync" id="sync-btn">
            ${icon('sync')}<span id="sync-label">同步 Stars</span>
          </button>
          <button class="btn btn--primary" data-action="add-open">${icon('plus')}添加仓库</button>
        </div>
      </div>

      <div id="sync-slot">${syncBoxHtml()}</div>

      <div class="toolbar">
        <div class="input-group">
          ${icon('search')}
          <input class="input" id="repo-search" placeholder="搜索仓库名称…" value="${escapeHtml(
            state.search
          )}" />
        </div>
        <select class="input" id="repo-lang" style="width:auto">${langOptionsHtml()}</select>
        <select class="input" id="repo-mon" style="width:auto">
          <option value="">所有状态</option>
          <option value="1">监控中</option>
          <option value="0">未监控</option>
        </select>
        <div id="batch-bar" style="display:flex;gap:8px;align-items:center;margin-left:auto">${batchBarHtml()}</div>
      </div>

      <div class="card table-wrap" id="repo-table">${skeletonRows()}</div>

      ${addModalHtml()}
      ${editModalHtml()}
    </div>`;
}

function addModalHtml() {
  return `
    <div class="modal-backdrop" id="modal-add" style="display:none">
      <div class="modal">
        <div class="modal__head">
          <div class="modal__title">添加仓库</div>
          <button class="icon-btn" data-action="modal-close" data-target="modal-add">${icon('close')}</button>
        </div>
        <div class="modal__body">
          <label class="field__label">仓库全名</label>
          <input class="input" id="add-name" placeholder="owner/repo" value="${escapeHtml(
            state.newName
          )}" />
        </div>
        <div class="modal__foot">
          <button class="btn" data-action="modal-close" data-target="modal-add">取消</button>
          <button class="btn btn--primary" id="add-confirm" data-action="add-confirm">确定添加</button>
        </div>
      </div>
    </div>`;
}

function editModalHtml() {
  const r = state.editRepo;
  return `
    <div class="modal-backdrop" id="modal-edit" style="display:none">
      <div class="modal">
        <div class="modal__head">
          <div class="modal__title" id="edit-title">编辑忽略规则</div>
          <button class="icon-btn" data-action="modal-close" data-target="modal-edit">${icon('close')}</button>
        </div>
        <div class="modal__body">
          <label class="field__label">忽略规则（支持正则）</label>
          <input class="input" id="edit-pattern" placeholder="例如: v0\\..*, -alpha$" />
          <div class="field__hint">匹配此规则的 tag 不会发送通知</div>

          <label style="display:inline-flex;align-items:center;gap:8px;margin-top:16px;cursor:pointer">
            <input type="checkbox" id="edit-track-tags" style="width:16px;height:16px;accent-color:var(--primary);cursor:pointer" />
            <span>监控 tag（仅无 Release 时回退）</span>
          </label>
          <div class="field__hint">开启后，若仓库未发布任何 GitHub Release，将自动以 tag 作为版本来源发送通知（tag 无更新日志）。</div>
        </div>
        <div class="modal__foot">
          <button class="btn" data-action="modal-close" data-target="modal-edit">取消</button>
          <button class="btn btn--primary" id="edit-confirm" data-action="edit-confirm">保存</button>
        </div>
      </div>
    </div>`;
}

// ---- 数据加载 ----
const doSearch = debounce(() => updateList(), 350);

async function updateList() {
  state.loading = true;
  const tbl = document.getElementById('repo-table');
  if (tbl) tbl.innerHTML = skeletonRows();
  try {
    const params = {
      query: state.search,
      language: state.lang,
    };
    if (state.monitored === '1') params.monitored = true;
    else if (state.monitored === '0') params.monitored = false;
    const repos = await api.listRepos(params);
    state.repos = repos;
  } catch (e) {
    toast(e.message || '加载失败', 'error');
    state.repos = [];
  } finally {
    state.loading = false;
    renderTableAndFilters();
  }
}

function renderTableAndFilters() {
  const tbl = document.getElementById('repo-table');
  if (tbl) tbl.innerHTML = tableHtml();
  const langSel = document.getElementById('repo-lang');
  if (langSel) {
    const prev = state.lang;
    langSel.innerHTML = langOptionsHtml();
    langSel.value = prev;
  }
  const bar = document.getElementById('batch-bar');
  if (bar) bar.innerHTML = batchBarHtml();
}

// ---- 交互 ----
function openModal(id) {
  document.getElementById(id).style.display = 'flex';
}
function closeModal(id) {
  document.getElementById(id).style.display = 'none';
}

async function startSync() {
  if (state.syncing) return;
  state.syncing = true;
  state.sync = { running: true, progress: 0, repos: 0, added: 0, updated: 0, removed: 0, error: '' };
  renderSyncSlot();
  setSyncBtn(true);
  try {
    await api.syncStars();
    state.poll = setInterval(pollSync, 1500);
  } catch (e) {
    toast(e.message || '同步失败', 'error');
    state.syncing = false;
    renderSyncSlot();
    setSyncBtn(false);
  }
}

async function pollSync() {
  try {
    const s = await api.syncStarsStatus();
    state.sync = s;
    renderSyncSlot();
    if (!s.running) {
      stopPoll();
      await updateList();
      if (s.error) toast(s.error, 'error');
      else toast('同步完成', 'success');
      setSyncBtn(false);
    }
  } catch {
    stopPoll();
    toast('同步状态查询失败', 'error');
    setSyncBtn(false);
  }
}

function stopPoll() {
  if (state.poll) clearInterval(state.poll);
  state.poll = null;
  state.syncing = false;
}

function renderSyncSlot() {
  const slot = document.getElementById('sync-slot');
  if (slot) slot.innerHTML = syncBoxHtml();
}

function setSyncBtn(on) {
  const btn = document.getElementById('sync-btn');
  const label = document.getElementById('sync-label');
  if (!btn) return;
  btn.disabled = on;
  if (label) {
    label.textContent = on
      ? `同步中 ${Math.round((state.sync ? state.sync.progress : 0) * 100)}%`
      : '同步 Stars';
  }
  const ic = btn.querySelector('svg');
  if (ic) ic.classList.toggle('spin', on);
}

async function toggleMonitor(id) {
  const r = state.repos.find((x) => x.id === id);
  if (!r) return;
  try {
    await api.setMonitored(id, !r.monitored);
    r.monitored = !r.monitored;
    renderTableAndFilters();
    toast(r.monitored ? '已开启监控' : '已关闭监控', 'success');
  } catch (e) {
    toast(e.message || '操作失败', 'error');
  }
}

async function batch(monitored) {
  const ids = Array.from(state.selected);
  if (ids.length === 0) return;
  try {
    await api.batchMonitor(ids, monitored);
    await updateList();
    state.selected.clear();
    renderTableAndFilters();
    toast(`已${monitored ? '开启' : '取消'} ${ids.length} 个仓库`, 'success');
  } catch (e) {
    toast(e.message || '操作失败', 'error');
  }
}

async function removeRepo(id) {
  if (!confirm('确定删除该仓库吗？（若已 Star 会同时在 GitHub 取消星标）')) return;
  try {
    await api.deleteRepo(id);
    state.selected.delete(id);
    await updateList();
    toast('已删除', 'success');
  } catch (e) {
    toast(e.message || '删除失败', 'error');
  }
}

function openEdit(id) {
  const r = state.repos.find((x) => x.id === id);
  if (!r) return;
  state.editRepo = r;
  openModal('modal-edit');
  const title = document.getElementById('edit-title');
  const input = document.getElementById('edit-pattern');
  const tt = document.getElementById('edit-track-tags');
  if (title) title.textContent = '编辑: ' + r.full_name;
  if (input) input.value = r.ignore_pattern || '';
  if (tt) tt.checked = !!r.track_tags;
}

async function saveEdit() {
  if (!state.editRepo) return;
  const input = document.getElementById('edit-pattern');
  const pattern = input ? input.value : '';
  const ttEl = document.getElementById('edit-track-tags');
  const trackTags = ttEl ? ttEl.checked : false;
  state.editing = true;
  try {
    await api.setMonitored(state.editRepo.id, state.editRepo.monitored, pattern, trackTags);
    state.editRepo.ignore_pattern = pattern;
    state.editRepo.track_tags = trackTags;
    closeModal('modal-edit');
    toast('已更新', 'success');
  } catch (e) {
    toast(e.message || '保存失败', 'error');
  } finally {
    state.editing = false;
  }
}

async function addRepo() {
  const input = document.getElementById('add-name');
  const name = input ? input.value.trim() : '';
  if (!name) return;
  state.adding = true;
  const btn = document.getElementById('add-confirm');
  if (btn) btn.disabled = true;
  try {
    await api.addRepo(name);
    state.newName = '';
    closeModal('modal-add');
    await updateList();
    toast('已添加', 'success');
  } catch (e) {
    toast(e.message || '添加失败', 'error');
  } finally {
    state.adding = false;
    if (btn) btn.disabled = false;
  }
}

function onClick(e) {
  const t = e.target.closest('[data-action]');
  if (!t) {
    // 点击模态框背景关闭
    if (e.target.classList && e.target.classList.contains('modal-backdrop')) {
      e.target.style.display = 'none';
    }
    return;
  }
  const action = t.getAttribute('data-action');
  const id = Number(t.getAttribute('data-id'));
  switch (action) {
    case 'sync':
      startSync();
      break;
    case 'add-open':
      openModal('modal-add');
      break;
    case 'modal-close':
      closeModal(t.getAttribute('data-target'));
      break;
    case 'add-confirm':
      addRepo();
      break;
    case 'edit-open':
      openEdit(id);
      break;
    case 'edit-confirm':
      saveEdit();
      break;
    case 'toggle-monitor':
      toggleMonitor(id);
      break;
    case 'delete':
      removeRepo(id);
      break;
    case 'batch-on':
      batch(true);
      break;
    case 'batch-off':
      batch(false);
      break;
    case 'select-all': {
      const checked = t.checked;
      state.repos.forEach((r) =>
        checked ? state.selected.add(r.id) : state.selected.delete(r.id)
      );
      renderTableAndFilters();
      break;
    }
  }
}

function onInput(e) {
  const t = e.target;
  if (t.id === 'repo-search') {
    state.search = t.value;
    doSearch();
  } else if (t.id === 'repo-lang') {
    state.lang = t.value;
    updateList();
  } else if (t.id === 'repo-mon') {
    state.monitored = t.value;
    updateList();
  } else if (t.id === 'add-name') {
    state.newName = t.value;
  }
}

function onChange(e) {
  const t = e.target;
  if (t.getAttribute && t.getAttribute('data-action') === 'select') {
    const id = Number(t.getAttribute('data-id'));
    if (t.checked) state.selected.add(id);
    else state.selected.delete(id);
    const bar = document.getElementById('batch-bar');
    if (bar) bar.innerHTML = batchBarHtml();
  }
}

function onKeydown(e) {
  if (e.key === 'Escape') {
    closeModal('modal-add');
    closeModal('modal-edit');
  } else if (e.key === 'Enter' && e.target.id === 'add-name') {
    addRepo();
  }
}

let root = null;

export function render(container) {
  root = container;
  container.innerHTML = shell();
  // 还原筛选选择
  const langSel = document.getElementById('repo-lang');
  if (langSel) langSel.value = state.lang;
  const monSel = document.getElementById('repo-mon');
  if (monSel) monSel.value = state.monitored;

  container.addEventListener('click', onClick);
  container.addEventListener('input', onInput);
  container.addEventListener('change', onChange);
  container.addEventListener('keydown', onKeydown);

  updateList();
}

export function cleanup() {
  if (state.poll) clearInterval(state.poll);
  state.poll = null;
  if (root) {
    root.removeEventListener('click', onClick);
    root.removeEventListener('input', onInput);
    root.removeEventListener('change', onChange);
    root.removeEventListener('keydown', onKeydown);
    root = null;
  }
}
