<script setup lang="ts">
import { onMounted, ref, reactive } from 'vue'
import { api } from '../api'
import type { Settings, SettingsSaveResult, Backup } from '../types'
import { useToast } from '../components/toast'
import { setTheme, getTheme } from '../theme'
import type { ThemeMode } from '../theme'

const toast = useToast()
const loading = ref(true)
const saving = ref(false)
const testing = ref(false)
const exporting = ref(false)
const showToken = ref(false)
const verifyResult = ref<SettingsSaveResult['verify'] | null>(null)

const themeMode = ref<ThemeMode>(getTheme())

const form = reactive<Settings>({
  github_token: '',
  shoutrrr_url: '',
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
})

const translateEngines = [
  { value: '', label: '关闭翻译' },
  { value: 'dlx', label: 'DLX（自托管 DeepL 兼容）' },
  { value: 'google', label: 'Google 网页翻译（免费接口）' },
  { value: 'youdao', label: '有道翻译（免费接口）' },
  { value: 'bing', label: '必应翻译（需 Azure 密钥）' },
  { value: 'openai', label: 'OpenAI 兼容 AI' },
]

async function loadSettings() {
  loading.value = true
  try {
    const s = await api.getSettings()
    Object.assign(form, s)
  } catch (e: any) {
    toast.error(e.message || '加载设置失败')
  } finally {
    loading.value = false
  }
}

async function saveSettings() {
  saving.value = true
  verifyResult.value = null
  try {
    const result = await api.saveSettings({ ...form })
    if (result.verify) verifyResult.value = result.verify
    toast.success('设置已保存')
  } catch (e: any) {
    toast.error(e.message)
  } finally {
    saving.value = false
  }
}

async function testNotify() {
  testing.value = true
  try {
    await api.testNotification('Chaxin 测试通知', '如果你收到这条消息，说明通知配置正确。')
    toast.success('测试通知已发送')
  } catch (e: any) {
    toast.error(e.message || '测试通知发送失败')
  } finally {
    testing.value = false
  }
}

async function doBackup() {
  exporting.value = true
  try {
    const backup = await api.backup()
    const blob = new Blob([JSON.stringify(backup, null, 2)], { type: 'application/json' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `chaxin-backup-${new Date().toISOString().slice(0, 10)}.json`
    a.click()
    URL.revokeObjectURL(url)
    toast.success('备份已下载')
  } catch (e: any) {
    toast.error(e.message)
  } finally {
    exporting.value = false
  }
}

function handleRestore() {
  const input = document.createElement('input')
  input.type = 'file'
  input.accept = '.json'
  input.onchange = async () => {
    const file = input.files?.[0]
    if (!file) return
    try {
      const text = await file.text()
      const backup: Backup = JSON.parse(text)
      await api.restore(backup)
      await loadSettings()
      toast.success('数据已恢复')
    } catch (e: any) {
      toast.error(e.message || '恢复失败')
    }
  }
  input.click()
}

function updateTheme(mode: ThemeMode) {
  themeMode.value = mode
  setTheme(mode)
}

onMounted(loadSettings)
</script>

<template>
  <div class="p-6 lg:p-8 max-w-3xl mx-auto">
    <!-- Header -->
    <div class="mb-8">
      <h1 class="text-2xl font-bold tracking-tight">系统设置</h1>
      <p class="mt-1 text-sm text-muted">配置 GitHub 连接、通知渠道和翻译参数</p>
    </div>

    <!-- Loading -->
    <div v-if="loading" class="space-y-6">
      <div v-for="i in 4" :key="i" class="h-48 rounded-2xl skeleton" />
    </div>

    <template v-else>
      <div class="space-y-5">

        <!-- Section: GitHub -->
        <section class="rounded-2xl border border-border/60 overflow-hidden">
          <div class="px-5 py-4 border-b border-border/50 bg-surface-alt/50">
            <div class="flex items-center gap-3">
              <div class="w-8 h-8 rounded-lg bg-indigo-50 dark:bg-indigo-500/10 flex items-center justify-center">
                <svg class="w-4 h-4 text-indigo-500" fill="currentColor" viewBox="0 0 24 24">
                  <path d="M12 0c-6.626 0-12 5.373-12 12 0 5.302 3.438 9.8 8.207 11.387.599.111.793-.261.793-.577v-2.234c-3.338.726-4.033-1.416-4.033-1.416-.546-1.387-1.333-1.756-1.333-1.756-1.089-.745.083-.729.083-.729 1.205.084 1.839 1.237 1.839 1.237 1.07 1.834 2.807 1.304 3.492.997.107-.775.418-1.305.762-1.604-2.665-.305-5.467-1.334-5.467-5.931 0-1.311.469-2.381 1.236-3.221-.124-.303-.535-1.524.117-3.176 0 0 1.008-.322 3.301 1.23.957-.266 1.983-.399 3.003-.404 1.02.005 2.047.138 3.006.404 2.291-1.552 3.297-1.23 3.297-1.23.653 1.653.242 2.874.118 3.176.77.84 1.235 1.911 1.235 3.221 0 4.609-2.807 5.624-5.479 5.921.43.372.823 1.102.823 2.222v3.293c0 .319.192.694.801.576 4.765-1.589 8.199-6.086 8.199-11.386 0-6.627-5.373-12-12-12z"/>
                </svg>
              </div>
              <div>
                <h2 class="text-sm font-semibold">GitHub 连接</h2>
                <p class="text-xs text-muted">配置 GitHub API 访问令牌</p>
              </div>
            </div>
          </div>
          <div class="p-5 space-y-4">
            <div>
              <label class="block text-sm font-medium mb-2">GitHub Token</label>
              <div class="relative">
                <input
                  v-model="form.github_token"
                  :type="showToken ? 'text' : 'password'"
                  placeholder="ghp_xxxxxxxxxxxxxxxxxxxx"
                  class="input-focus w-full px-4 py-2.5 pr-10 text-sm rounded-xl border border-border/60 bg-surface font-mono placeholder:text-muted/50"
                />
                <button
                  @click="showToken = !showToken"
                  class="absolute right-2 top-1/2 -translate-y-1/2 p-1.5 rounded-lg text-muted hover:text-foreground hover:bg-surface-alt transition-colors"
                >
                  <svg v-if="showToken" class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13.875 18.825A10.05 10.05 0 0112 19c-4.478 0-8.268-2.943-9.543-7a9.97 9.97 0 011.563-3.029m5.858.908a3 3 0 114.243 4.243M9.878 9.878l4.242 4.242M9.88 9.88l-3.29-3.29m7.532 7.532l3.29 3.29M3 3l3.59 3.59m0 0A9.953 9.953 0 0112 5c4.478 0 8.268 2.943 9.543 7a10.025 10.025 0 01-4.132 5.411m0 0L21 21"/>
                  </svg>
                  <svg v-else class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z"/><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z"/>
                  </svg>
                </button>
              </div>
            </div>
            <div>
              <label class="block text-sm font-medium mb-2">GitHub API 地址</label>
              <input
                v-model="form.github_api_base_url"
                placeholder="https://api.github.com"
                class="input-focus w-full px-4 py-2.5 text-sm rounded-xl border border-border/60 bg-surface placeholder:text-muted/50"
              />
            </div>
            <!-- Verify result -->
            <div v-if="verifyResult" :class="['p-3 rounded-xl text-sm border', verifyResult.token_valid ? 'bg-emerald-50 border-emerald-200/50 dark:bg-emerald-500/10 dark:border-emerald-500/20' : 'bg-rose-50 border-rose-200/50 dark:bg-rose-500/10 dark:border-rose-500/20']">
              <template v-if="verifyResult.token_valid">
                <p class="text-emerald-700 dark:text-emerald-400">
                  令牌验证成功
                  <span v-if="verifyResult.username" class="font-medium ml-1">({{ verifyResult.username }})</span>
                </p>
              </template>
              <template v-else>
                <p class="text-rose-700 dark:text-rose-400">
                  令牌验证失败: {{ verifyResult.token_error || '未知错误' }}
                </p>
              </template>
            </div>
          </div>
        </section>

        <!-- Section: Notifications -->
        <section class="rounded-2xl border border-border/60 overflow-hidden">
          <div class="px-5 py-4 border-b border-border/50 bg-surface-alt/50">
            <div class="flex items-center gap-3">
              <div class="w-8 h-8 rounded-lg bg-sky-50 dark:bg-sky-500/10 flex items-center justify-center">
                <svg class="w-4 h-4 text-sky-500" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 17h5l-1.405-1.405A2.032 2.032 0 0118 14.158V11a6 6 0 10-12 0v3.159c0 .538-.214 1.055-.595 1.436L4 17h5m6 0v1a3 3 0 11-6 0v-1m6 0H9"/>
                </svg>
              </div>
              <div>
                <h2 class="text-sm font-semibold">通知配置</h2>
                <p class="text-xs text-muted">配置 Shoutrrr 通知服务和监控参数</p>
              </div>
            </div>
          </div>
          <div class="p-5 space-y-4">
            <div>
              <label class="block text-sm font-medium mb-2">Shoutrrr URL</label>
              <input
                v-model="form.shoutrrr_url"
                placeholder="discord://token@channel..."
                class="input-focus w-full px-4 py-2.5 text-sm rounded-xl border border-border/60 bg-surface placeholder:text-muted/50 font-mono"
              />
              <p class="mt-1.5 text-xs text-muted">
                支持 Discord, Telegram, Slack, Webhook 等，详情见
                <a href="https://containrrr.dev/shoutrrr/" target="_blank" class="text-indigo-500 hover:text-indigo-600 dark:text-indigo-400">Shoutrrr 文档</a>
              </p>
            </div>

            <div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
              <div>
                <label class="block text-sm font-medium mb-2">轮询间隔</label>
                <select
                  v-model="form.poll_interval"
                  class="input-focus w-full px-4 py-2.5 text-sm rounded-xl border border-border/60 bg-surface"
                >
                  <option value="1m">1 分钟</option>
                  <option value="5m">5 分钟</option>
                  <option value="10m">10 分钟</option>
                  <option value="30m">30 分钟</option>
                  <option value="1h">1 小时</option>
                </select>
              </div>
              <div>
                <label class="block text-sm font-medium mb-2">最大通知数</label>
                <input
                  v-model.number="form.max_notifications"
                  type="number"
                  min="1"
                  max="200"
                  class="input-focus w-full px-4 py-2.5 text-sm rounded-xl border border-border/60 bg-surface"
                />
              </div>
            </div>

            <div class="flex flex-col gap-3">
              <label class="flex items-center gap-3 p-3 rounded-xl border border-border/60 hover:bg-surface-alt/50 transition-colors cursor-pointer">
                <input
                  v-model="form.notify_on_first_run"
                  type="checkbox"
                  class="rounded-md border-border/60 text-indigo-500 focus:ring-indigo-500"
                />
                <div>
                  <span class="text-sm font-medium">首次启动发送通知</span>
                  <p class="text-xs text-muted mt-0.5">首次运行时发送所有已有 tag 的通知</p>
                </div>
              </label>
              <label class="flex items-center gap-3 p-3 rounded-xl border border-border/60 hover:bg-surface-alt/50 transition-colors cursor-pointer">
                <input
                  v-model="form.monitor_new_stars"
                  type="checkbox"
                  class="rounded-md border-border/60 text-indigo-500 focus:ring-indigo-500"
                />
                <div>
                  <span class="text-sm font-medium">自动监控新的 Stars</span>
                  <p class="text-xs text-muted mt-0.5">同步 Stars 时自动启用新仓库的监控</p>
                </div>
              </label>
            </div>

            <div class="flex items-center gap-3">
              <button
                @click="testNotify"
                :disabled="testing"
                class="px-4 py-2 text-sm font-medium rounded-xl border border-border/60 text-muted hover:text-foreground hover:bg-surface-alt transition-all disabled:opacity-50"
              >
                <span v-if="testing" class="flex items-center gap-2">
                  <svg class="w-4 h-4 animate-spin" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"/>
                  </svg>
                  发送中...
                </span>
                <span v-else>发送测试通知</span>
              </button>
            </div>
          </div>
        </section>

        <!-- Section: Translation -->
        <section class="rounded-2xl border border-border/60 overflow-hidden">
          <div class="px-5 py-4 border-b border-border/50 bg-surface-alt/50">
            <div class="flex items-center gap-3">
              <div class="w-8 h-8 rounded-lg bg-violet-50 dark:bg-violet-500/10 flex items-center justify-center">
                <svg class="w-4 h-4 text-violet-500" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 5h12M9 3v2m1.048 9.5A18.022 18.022 0 016.412 9m6.088 9h7M11 21l5-10 5 10M12.751 5C11.783 10.77 8.07 15.61 3 18.129"/>
                </svg>
              </div>
              <div>
                <h2 class="text-sm font-semibold">翻译设置</h2>
                <p class="text-xs text-muted">配置 Release Notes 自动翻译</p>
              </div>
            </div>
          </div>
          <div class="p-5 space-y-4">
            <div>
              <label class="block text-sm font-medium mb-2">翻译引擎</label>
              <select
                v-model="form.translate_engine"
                class="input-focus w-full px-4 py-2.5 text-sm rounded-xl border border-border/60 bg-surface"
              >
                <option v-for="eng in translateEngines" :key="eng.value" :value="eng.value">
                  {{ eng.label }}
                </option>
              </select>
            </div>
            <template v-if="form.translate_engine">
              <div>
                <label class="block text-sm font-medium mb-2">目标语言</label>
                <input
                  v-model="form.translate_target_lang"
                  placeholder="zh"
                  class="input-focus w-full px-4 py-2.5 text-sm rounded-xl border border-border/60 bg-surface"
                />
              </div>

              <!-- DLX -->
              <template v-if="form.translate_engine === 'dlx'">
                <div>
                  <label class="block text-sm font-medium mb-2">DLX 服务地址</label>
                  <input
                    v-model="form.translate_url"
                    placeholder="http://localhost:1188"
                    class="input-focus w-full px-4 py-2.5 text-sm rounded-xl border border-border/60 bg-surface placeholder:text-muted/50"
                  />
                </div>
                <div>
                  <label class="block text-sm font-medium mb-2">DLX API Key（可选）</label>
                  <input
                    v-model="form.translate_api_key"
                    type="password"
                    placeholder="可留空"
                    class="input-focus w-full px-4 py-2.5 text-sm rounded-xl border border-border/60 bg-surface placeholder:text-muted/50 font-mono"
                  />
                </div>
              </template>

              <!-- Google -->
              <template v-if="form.translate_engine === 'google'">
                <div>
                  <label class="block text-sm font-medium mb-2">TLD 域名后缀</label>
                  <input
                    v-model="form.translate_url"
                    placeholder=".cn（默认 .com）"
                    class="input-focus w-full px-4 py-2.5 text-sm rounded-xl border border-border/60 bg-surface placeholder:text-muted/50"
                  />
                </div>
              </template>

              <!-- Bing -->
              <template v-if="form.translate_engine === 'bing'">
                <div>
                  <label class="block text-sm font-medium mb-2">Azure API Key</label>
                  <input
                    v-model="form.translate_api_key"
                    type="password"
                    placeholder="输入 Azure 密钥"
                    class="input-focus w-full px-4 py-2.5 text-sm rounded-xl border border-border/60 bg-surface placeholder:text-muted/50 font-mono"
                  />
                </div>
                <div>
                  <label class="block text-sm font-medium mb-2">Azure 区域</label>
                  <input
                    v-model="form.translate_url"
                    placeholder="global（默认）"
                    class="input-focus w-full px-4 py-2.5 text-sm rounded-xl border border-border/60 bg-surface placeholder:text-muted/50"
                  />
                </div>
              </template>

              <!-- OpenAI -->
              <template v-if="form.translate_engine === 'openai'">
                <div>
                  <label class="block text-sm font-medium mb-2">API 地址</label>
                  <input
                    v-model="form.translate_url"
                    placeholder="https://api.openai.com/v1/chat/completions"
                    class="input-focus w-full px-4 py-2.5 text-sm rounded-xl border border-border/60 bg-surface placeholder:text-muted/50"
                  />
                </div>
                <div>
                  <label class="block text-sm font-medium mb-2">API Key</label>
                  <input
                    v-model="form.translate_api_key"
                    type="password"
                    placeholder="sk-..."
                    class="input-focus w-full px-4 py-2.5 text-sm rounded-xl border border-border/60 bg-surface placeholder:text-muted/50 font-mono"
                  />
                </div>
                <div>
                  <label class="block text-sm font-medium mb-2">模型名称</label>
                  <input
                    v-model="form.translate_model"
                    placeholder="gpt-3.5-turbo"
                    class="input-focus w-full px-4 py-2.5 text-sm rounded-xl border border-border/60 bg-surface placeholder:text-muted/50"
                  />
                </div>
              </template>
            </template>
          </div>
        </section>

        <!-- Section: Theme -->
        <section class="rounded-2xl border border-border/60 overflow-hidden">
          <div class="px-5 py-4 border-b border-border/50 bg-surface-alt/50">
            <div class="flex items-center gap-3">
              <div class="w-8 h-8 rounded-lg bg-amber-50 dark:bg-amber-500/10 flex items-center justify-center">
                <svg class="w-4 h-4 text-amber-500" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M7 21a4 4 0 01-4-4V5a2 2 0 012-2h4a2 2 0 012 2v12a4 4 0 01-4 4zm0 0h12a2 2 0 002-2v-4a2 2 0 00-2-2h-2.343M11 7.343l1.657-1.657a2 2 0 012.828 0l2.829 2.829a2 2 0 010 2.828l-8.486 8.485M7 17h.01"/>
                </svg>
              </div>
              <div>
                <h2 class="text-sm font-semibold">主题设置</h2>
                <p class="text-xs text-muted">选择界面主题模式</p>
              </div>
            </div>
          </div>
          <div class="p-5">
            <div class="flex gap-3">
              <button
                v-for="opt in [
                  { key: 'system', label: '跟随系统', icon: 'M9.75 17L9 20l-1 1h8l-1-1-.75-3M3 13h18M5 17h14a2 2 0 002-2V5a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z' },
                  { key: 'light', label: '浅色', icon: 'M12 3v1m0 16v1m9-9h-1M4 12H3m15.364 6.364l-.707-.707M6.343 6.343l-.707-.707m12.728 0l-.707.707M6.343 17.657l-.707.707M16 12a4 4 0 11-8 0 4 4 0 018 0z' },
                  { key: 'dark', label: '深色', icon: 'M20.354 15.354A9 9 0 018.646 3.646 9.003 9.003 0 0012 21a9.003 9.003 0 008.354-5.646z' },
                ]"
                :key="opt.key"
                @click="updateTheme(opt.key as ThemeMode)"
                :class="[
                  'flex-1 flex flex-col items-center gap-2 p-4 rounded-xl border-2 transition-all duration-200',
                  themeMode === opt.key
                    ? 'border-indigo-500 bg-indigo-50/50 dark:bg-indigo-500/5'
                    : 'border-border/60 hover:border-border text-muted hover:text-foreground hover:bg-surface-alt/50'
                ]"
              >
                <svg class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" :d="opt.icon"/>
                </svg>
                <span class="text-xs font-medium">{{ opt.label }}</span>
              </button>
            </div>
          </div>
        </section>

        <!-- Section: Backup -->
        <section class="rounded-2xl border border-border/60 overflow-hidden">
          <div class="px-5 py-4 border-b border-border/50 bg-surface-alt/50">
            <div class="flex items-center gap-3">
              <div class="w-8 h-8 rounded-lg bg-rose-50 dark:bg-rose-500/10 flex items-center justify-center">
                <svg class="w-4 h-4 text-rose-500" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-8l-4-4m0 0L8 8m4-4v12"/>
                </svg>
              </div>
              <div>
                <h2 class="text-sm font-semibold">数据备份</h2>
                <p class="text-xs text-muted">备份和恢复应用数据</p>
              </div>
            </div>
          </div>
          <div class="p-5 flex items-center gap-3">
            <button
              @click="doBackup"
              :disabled="exporting"
              class="px-4 py-2 text-sm font-medium rounded-xl bg-gradient-to-r from-indigo-500 to-violet-500 text-white hover:from-indigo-600 hover:to-violet-600 disabled:opacity-50 transition-all"
            >
              {{ exporting ? '导出中...' : '导出备份' }}
            </button>
            <button
              @click="handleRestore"
              class="px-4 py-2 text-sm font-medium rounded-xl border border-border/60 text-muted hover:text-foreground hover:bg-surface-alt transition-all"
            >
              恢复备份
            </button>
          </div>
        </section>

        <!-- Save Button -->
        <div class="sticky bottom-6 flex justify-end">
          <button
            @click="saveSettings"
            :disabled="saving"
            class="px-6 py-3 text-sm font-medium rounded-xl bg-gradient-to-r from-indigo-500 to-violet-500 text-white hover:from-indigo-600 hover:to-violet-600 disabled:opacity-50 transition-all shadow-lg shadow-indigo-500/20"
          >
            <span v-if="saving" class="flex items-center gap-2">
              <svg class="w-4 h-4 animate-spin" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"/>
              </svg>
              保存中...
            </span>
            <span v-else>保存设置</span>
          </button>
        </div>

      </div>
    </template>
  </div>
</template>

<style scoped>
.text-muted {
  color: var(--color-text-muted);
}

.text-foreground {
  color: var(--color-text);
}

.bg-surface {
  background: var(--color-surface);
}

.bg-surface-alt {
  background: var(--color-surface-alt);
}

.bg-surface-alt\/50 {
  background: color-mix(in srgb, var(--color-surface-alt) 50%, transparent);
}

.border-border\/60 {
  border-color: color-mix(in srgb, var(--color-border) 60%, transparent);
}

.border-border\/50 {
  border-color: color-mix(in srgb, var(--color-border) 50%, transparent);
}
</style>
