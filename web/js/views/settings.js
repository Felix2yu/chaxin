// 系统设置：GitHub / 通知 / 翻译 / 主题 / 备份。表单状态集中在 form 对象，
// 输入即时写入 form（不重渲染以保持焦点），离散操作才重渲染。

import { api } from '../api.js';
import { icon, escapeHtml, toast } from '../ui.js';
import { getTheme, setTheme } from '../theme.js';

const DEFAULT_FORM = {
  github_token: '',
  notify_url: '',
  poll_interval: '5m',
  notify_on_first_run: false,
  monitor_new_stars: true,
  github_api_base_url: 'https://api.github.com',
  max_notifications: 50,
  translate_engine: '',
  translate_target_lang: 'zh',
  translate_url: '',
  translate_api_key: '',
  translate_model: '',
};

const ENGINES = [
  { value: '', label: '关闭翻译' },
  { value: 'dlx', label: 'DLX（自托管 DeepL 兼容）' },
  { value: 'google', label: 'Google 网页翻译（免费接口）' },
  { value: 'youdao', label: '有道翻译（免费接口）' },
  { value: 'bing', label: '必应翻译（需 Azure 密钥）' },
  { value: 'openai', label: 'OpenAI 兼容 AI' },
];

const state = {
  loading: true,
  saving: false,
  testing: false,
  exporting: false,
  showToken: false,
  verify: null,
  themeMode: 'system',
};

let form = { ...DEFAULT_FORM };
let root = null;

// ---- 模板 ----
function sectionHead(iconName, cls, title, sub) {
  return `
    <div class="section__head">
      <div class="section__icon ${cls}">${icon(iconName)}</div>
      <div>
        <div class="section__title">${title}</div>
        <div class="section__sub">${sub}</div>
      </div>
    </div>`;
}

function translateFields() {
  if (!form.translate_engine) return '';
  const common = `
    <div class="field">
      <label class="field__label">目标语言</label>
      <input class="input" data-field="translate_target_lang" value="${escapeHtml(
        form.translate_target_lang
      )}" placeholder="zh" />
    </div>`;
  let extra = '';
  switch (form.translate_engine) {
    case 'dlx':
      extra = `
        <div class="field">
          <label class="field__label">DLX 服务地址</label>
          <input class="input" data-field="translate_url" value="${escapeHtml(
            form.translate_url
          )}" placeholder="http://localhost:1188" />
        </div>
        <div class="field">
          <label class="field__label">DLX API Key（可选）</label>
          <input class="input" type="password" data-field="translate_api_key" value="${escapeHtml(
            form.translate_api_key
          )}" placeholder="可留空" />
        </div>`;
      break;
    case 'google':
      extra = `
        <div class="field">
          <label class="field__label">TLD 域名后缀</label>
          <input class="input" data-field="translate_url" value="${escapeHtml(
            form.translate_url
          )}" placeholder=".cn（默认 .com）" />
        </div>`;
      break;
    case 'bing':
      extra = `
        <div class="field">
          <label class="field__label">Azure API Key</label>
          <input class="input" type="password" data-field="translate_api_key" value="${escapeHtml(
            form.translate_api_key
          )}" placeholder="输入 Azure 密钥" />
        </div>
        <div class="field">
          <label class="field__label">Azure 区域</label>
          <input class="input" data-field="translate_url" value="${escapeHtml(
            form.translate_url
          )}" placeholder="global（默认）" />
        </div>`;
      break;
    case 'openai':
      extra = `
        <div class="field">
          <label class="field__label">API 地址</label>
          <input class="input" data-field="translate_url" value="${escapeHtml(
            form.translate_url
          )}" placeholder="https://api.openai.com/v1/chat/completions" />
        </div>
        <div class="field">
          <label class="field__label">API Key</label>
          <input class="input" type="password" data-field="translate_api_key" value="${escapeHtml(
            form.translate_api_key
          )}" placeholder="sk-..." />
        </div>
        <div class="field">
          <label class="field__label">模型名称</label>
          <input class="input" data-field="translate_model" value="${escapeHtml(
            form.translate_model
          )}" placeholder="gpt-3.5-turbo" />
        </div>`;
      break;
  }
  return common + extra;
}

function template() {
  const verifyHtml = state.verify
    ? state.verify.token_valid
      ? `<div class="verify verify--ok">令牌验证成功${
          state.verify.username ? '（' + escapeHtml(state.verify.username) + '）' : ''
        }</div>`
      : `<div class="verify verify--err">令牌验证失败: ${escapeHtml(
          state.verify.token_error || '未知错误'
        )}</div>`
    : '';

  const themeOpts = [
    { key: 'system', label: '跟随系统', ic: 'monitor' },
    { key: 'light', label: '浅色', ic: 'sun' },
    { key: 'dark', label: '深色', ic: 'moon' },
  ]
    .map(
      (t) => `
      <button class="theme-opt ${state.themeMode === t.key ? 'is-active' : ''}" data-action="theme" data-mode="${
        t.key
      }">
        ${icon(t.ic)}<span>${t.label}</span>
      </button>`
    )
    .join('');

  if (state.loading) {
    return `<div class="content__inner"><div class="page-head"><h1>系统设置</h1><p>配置 GitHub 连接、通知渠道和翻译参数</p></div>
      ${Array(4).fill('<div class="skeleton" style="height:180px;border-radius:14px;margin-bottom:16px"></div>').join('')}</div>`;
  }

  return `
    <div class="content__inner">
      <div class="page-head"><h1>系统设置</h1><p>配置 GitHub 连接、通知渠道和翻译参数</p></div>

      <div class="section">
        ${sectionHead('github', 'section__icon--github', 'GitHub 连接', '配置 GitHub API 访问令牌')}
        <div class="section__body">
          <div class="field">
            <label class="field__label">GitHub Token</label>
            <div style="position:relative">
              <input class="input" data-field="github_token" type="${
                state.showToken ? 'text' : 'password'
              }" value="${escapeHtml(form.github_token)}" placeholder="ghp_xxxxxxxxxxxxxxxxxxxx" style="padding-right:42px;font-family:ui-monospace,monospace" />
              <button class="icon-btn" data-action="toggle-token" style="position:absolute;right:8px;top:50%;transform:translateY(-50%)">${
                state.showToken ? icon('eye') : icon('eye_off')
              }</button>
            </div>
          </div>
          <div class="field">
            <label class="field__label">GitHub API 地址</label>
            <input class="input" data-field="github_api_base_url" value="${escapeHtml(
              form.github_api_base_url
            )}" placeholder="https://api.github.com" />
          </div>
          ${verifyHtml}
        </div>
      </div>

      <div class="section">
        ${sectionHead('notifications', 'section__icon--notify', '通知配置', '配置通知服务和监控参数')}
        <div class="section__body">
          <div class="field">
            <label class="field__label">通知 URL</label>
            <input class="input" data-field="notify_url" value="${escapeHtml(
              form.notify_url
            )}" placeholder="discord://token@channel..." style="font-family:ui-monospace,monospace" />
            <div class="field__hint">支持 Discord, Telegram, Slack, Webhook 等，详情见 <a href="https://appriseit.com/services/" target="_blank" rel="noopener">Apprise 支持的服务</a></div>
          </div>
          <div class="grid-2">
            <div class="field" style="margin-bottom:0">
              <label class="field__label">轮询间隔</label>
              <select class="input" data-field="poll_interval">
                ${['1m', '5m', '10m', '30m', '1h']
                  .map(
                    (v) =>
                      `<option value="${v}" ${
                        form.poll_interval === v ? 'selected' : ''
                      }>${v}</option>`
                  )
                  .join('')}
              </select>
            </div>
            <div class="field" style="margin-bottom:0">
              <label class="field__label">最大通知数</label>
              <input class="input" type="number" min="1" max="200" data-field="max_notifications" value="${escapeHtml(
                form.max_notifications
              )}" />
            </div>
          </div>
          <div style="display:flex;flex-direction:column;gap:10px;margin-top:16px">
            <label class="check-row">
              <input type="checkbox" data-field="notify_on_first_run" ${
                form.notify_on_first_run ? 'checked' : ''
              } style="width:16px;height:16px;accent-color:var(--primary);margin-top:2px" />
              <div><div class="check-row__text">首次启动发送通知</div><div class="check-row__hint">首次运行时发送所有已有 tag 的通知</div></div>
            </label>
            <label class="check-row">
              <input type="checkbox" data-field="monitor_new_stars" ${
                form.monitor_new_stars ? 'checked' : ''
              } style="width:16px;height:16px;accent-color:var(--primary);margin-top:2px" />
              <div><div class="check-row__text">自动监控新的 Stars</div><div class="check-row__hint">同步 Stars 时自动启用新仓库的监控</div></div>
            </label>
          </div>
          <div style="margin-top:16px">
            <button class="btn" data-action="test" ${state.testing ? 'disabled' : ''}>${
    state.testing
      ? `<span class="spin" style="display:inline-flex">${icon('sync')}</span> 发送中…`
      : '发送测试通知'
  }</button>
          </div>
        </div>
      </div>

      <div class="section">
        ${sectionHead('translate', 'section__icon--translate', '翻译设置', '配置 Release Notes 自动翻译')}
        <div class="section__body">
          <div class="field" style="margin-bottom:0">
            <label class="field__label">翻译引擎</label>
            <select class="input" data-field="translate_engine">
              ${ENGINES.map(
                (e) =>
                  `<option value="${e.value}" ${
                    form.translate_engine === e.value ? 'selected' : ''
                  }>${e.label}</option>`
              ).join('')}
            </select>
          </div>
          <div id="translate-extra" style="margin-top:16px">${translateFields()}</div>
        </div>
      </div>

      <div class="section">
        ${sectionHead('monitor', 'section__icon--theme', '主题设置', '选择界面主题模式')}
        <div class="section__body"><div class="theme-options">${themeOpts}</div></div>
      </div>

      <div class="section">
        ${sectionHead('backup', 'section__icon--backup', '数据备份', '备份和恢复应用数据')}
        <div class="section__body" style="display:flex;gap:10px">
          <button class="btn btn--primary" data-action="backup" ${
            state.exporting ? 'disabled' : ''
          }>${icon('backup')}${state.exporting ? '导出中…' : '导出备份'}</button>
          <button class="btn" data-action="restore">恢复备份</button>
        </div>
      </div>

      <div class="footer-actions">
        <button class="btn btn--primary" data-action="save" ${
          state.saving ? 'disabled' : ''
        }>${state.saving ? '保存中…' : '保存设置'}</button>
      </div>
    </div>`;
}

function rerender() {
  if (root) root.innerHTML = template();
}

// ---- 数据 ----
async function loadSettings() {
  state.loading = true;
  rerender();
  try {
    const s = await api.getSettings();
    form = { ...DEFAULT_FORM, ...s };
  } catch (e) {
    toast(e.message || '加载设置失败', 'error');
  } finally {
    state.loading = false;
    rerender();
  }
}

async function saveSettings() {
  state.saving = true;
  state.verify = null;
  rerender();
  try {
    const result = await api.saveSettings({ ...form });
    if (result && result.verify) state.verify = result.verify;
    toast('设置已保存', 'success');
  } catch (e) {
    toast(e.message || '保存失败', 'error');
  } finally {
    state.saving = false;
    rerender();
  }
}

async function testNotify() {
  state.testing = true;
  rerender();
  try {
    await api.testNotification('察新 测试通知', '如果你收到这条消息，说明通知配置正确。');
    toast('测试通知已发送', 'success');
  } catch (e) {
    toast(e.message || '测试通知发送失败', 'error');
  } finally {
    state.testing = false;
    rerender();
  }
}

async function doBackup() {
  state.exporting = true;
  rerender();
  try {
    const backup = await api.backup();
    const blob = new Blob([JSON.stringify(backup, null, 2)], { type: 'application/json' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `chaxin-backup-${new Date().toISOString().slice(0, 10)}.json`;
    a.click();
    URL.revokeObjectURL(url);
    toast('备份已下载', 'success');
  } catch (e) {
    toast(e.message || '导出失败', 'error');
  } finally {
    state.exporting = false;
    rerender();
  }
}

function handleRestore() {
  const input = document.createElement('input');
  input.type = 'file';
  input.accept = '.json';
  input.onchange = async () => {
    const file = input.files && input.files[0];
    if (!file) return;
    try {
      const text = await file.text();
      const backup = JSON.parse(text);
      await api.restore(backup);
      await loadSettings();
      toast('数据已恢复', 'success');
    } catch (e) {
      toast(e.message || '恢复失败', 'error');
    }
  };
  input.click();
}

function updateTheme(mode) {
  state.themeMode = mode;
  setTheme(mode);
  rerender();
}

// ---- 事件 ----
function onField(e) {
  const t = e.target;
  const f = t.getAttribute && t.getAttribute('data-field');
  if (!f) return;
  if (t.type === 'checkbox') form[f] = t.checked;
  else if (f === 'max_notifications') form[f] = parseInt(t.value, 10) || 0;
  else form[f] = t.value;
  // 引擎切换需重渲染以显示/隐藏条件字段
  if (f === 'translate_engine') {
    const extra = document.getElementById('translate-extra');
    if (extra) extra.innerHTML = translateFields();
  }
}

function onClick(e) {
  const t = e.target.closest('[data-action]');
  if (!t) return;
  const action = t.getAttribute('data-action');
  switch (action) {
    case 'toggle-token':
      state.showToken = !state.showToken;
      rerender();
      break;
    case 'theme':
      updateTheme(t.getAttribute('data-mode'));
      break;
    case 'test':
      testNotify();
      break;
    case 'backup':
      doBackup();
      break;
    case 'restore':
      handleRestore();
      break;
    case 'save':
      saveSettings();
      break;
  }
}

export function render(container) {
  root = container;
  state.themeMode = getTheme();
  container.innerHTML = template();
  container.addEventListener('click', onClick);
  container.addEventListener('input', onField);
  container.addEventListener('change', onField);
  loadSettings();
}

export function cleanup() {
  if (root) {
    root.removeEventListener('click', onClick);
    root.removeEventListener('input', onField);
    root.removeEventListener('change', onField);
    root = null;
  }
}
