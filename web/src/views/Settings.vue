<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { api } from '../api'
import { useToast } from '../components/toast'
import ToggleSwitch from '../components/ToggleSwitch.vue'
import { applyTheme, getTheme, setTheme } from '../theme'
import type { Settings, SettingsSaveResult } from '../types'
import type { ThemeMode } from '../theme'

const toast = useToast()
const loading = ref(true)
const saving = ref(false)
const testing = ref(false)
const backing = ref(false)
const restoring = ref(false)
const fileInput = ref<HTMLInputElement | null>(null)

const feedUrl = `${location.origin}/feed`

async function copyFeed() {
  try {
    await navigator.clipboard.writeText(feedUrl)
    toast.success('订阅地址已复制')
  } catch {
    toast.error('复制失败，请手动复制')
  }
}

const form = ref<Settings>({
  github_token: '',
  shoutrrr_url: '',
  poll_interval: '30m',
  notify_on_first_run: false,
  github_api_base_url: '',
  max_notifications: 0,
  translate_engine: 'off',
  translate_target_lang: 'zh-Hans',
  translate_url: '',
  translate_api_key: '',
  translate_model: '',
})

const showToken = ref(false)
const saveResult = ref<SettingsSaveResult['verify'] | null>(null)

const themeMode = ref<ThemeMode>(getTheme())

const themeOptions: { value: ThemeMode; label: string; desc: string }[] = [
  { value: 'system', label: '跟随系统', desc: '随操作系统自动切换' },
  { value: 'light', label: '亮色', desc: '始终使用浅色主题' },
  { value: 'dark', label: '暗色', desc: '始终使用深色主题' },
]

function setThemeMode(mode: ThemeMode) {
  themeMode.value = mode
  setTheme(mode)
}

const intervalOptions = [
  { value: '5m', label: '5 分钟' },
  { value: '10m', label: '10 分钟' },
  { value: '15m', label: '15 分钟' },
  { value: '30m', label: '30 分钟' },
  { value: '1h', label: '1 小时' },
  { value: '3h', label: '3 小时' },
  { value: '6h', label: '6 小时' },
  { value: '12h', label: '12 小时' },
  { value: '24h', label: '24 小时' },
]

const engineOptions = [
  { value: 'off', label: '关闭' },
  { value: 'dlx', label: 'DLX（自托管 DeepL 兼容）' },
  { value: 'google', label: 'Google 网页翻译（免费接口）' },
  { value: 'youdao', label: '有道（免费接口）' },
  { value: 'bing', label: '必应翻译（需 Azure 密钥）' },
  { value: 'openai', label: 'OpenAI 兼容 AI' },
]

const langOptions = [
  { value: 'zh-Hans', label: '简体中文' },
  { value: 'zh-Hant', label: '繁体中文' },
  { value: 'en', label: 'English' },
  { value: 'ja', label: '日本語' },
  { value: 'ko', label: '한국어' },
]

async function load() {
  loading.value = true
  try {
    const s = await api.getSettings()
    form.value = { ...s }
  } catch (e) {
    toast.error((e as Error).message)
  } finally {
    loading.value = false
  }
}

async function save() {
  saving.value = true
  saveResult.value = null
  try {
    const res = await api.saveSettings(form.value)
    saveResult.value = res.verify
    toast.success('设置已保存')
  } catch (e) {
    toast.error((e as Error).message)
  } finally {
    saving.value = false
  }
}

async function testNotify() {
  testing.value = true
  try {
    await api.testNotification('察新 测试通知', '如果你收到这条消息，说明通知配置已生效。')
    toast.success('测试通知已发送')
  } catch (e) {
    toast.error((e as Error).message)
  } finally {
    testing.value = false
  }
}

async function exportBackup() {
  backing.value = true
  try {
    const b = await api.backup()
    const blob = new Blob([JSON.stringify(b, null, 2)], { type: 'application/json' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `chaxin-backup-${new Date().toISOString().slice(0, 10)}.json`
    a.click()
    URL.revokeObjectURL(url)
    toast.success('配置已导出')
  } catch (e) {
    toast.error((e as Error).message)
  } finally {
    backing.value = false
  }
}

function pickBackup() {
  fileInput.value?.click()
}

async function importBackup(e: Event) {
  const input = e.target as HTMLInputElement
  const file = input.files?.[0]
  if (!file) return
  if (!window.confirm('导入将覆盖当前全部设置与仓库列表，确定继续吗？')) {
    input.value = ''
    return
  }
  restoring.value = true
  try {
    const text = await file.text()
    const data = JSON.parse(text)
    await api.restore(data)
    toast.success('配置已导入')
    await load()
  } catch (err) {
    toast.error(`导入失败：${(err as Error).message}`)
  } finally {
    restoring.value = false
    input.value = ''
  }
}

onMounted(load)
</script>

<template>
  <div class="mx-auto max-w-3xl p-4 md:p-8">
    <div class="mb-6">
      <h1 class="text-2xl font-bold text-slate-900 dark:text-slate-100">设置</h1>
      <p class="mt-1 text-sm text-slate-500 dark:text-slate-400">配置 GitHub 认证、通知目标与轮询策略</p>
    </div>

    <div v-if="loading" class="space-y-3">
      <div v-for="i in 4" :key="i" class="h-20 animate-pulse rounded-2xl bg-slate-100 dark:bg-slate-800" />
    </div>

    <form v-else class="space-y-5" @submit.prevent="save">
      <section class="rounded-2xl border border-slate-200 bg-white p-5 shadow-sm dark:border-slate-800 dark:bg-slate-900">
        <h2 class="text-sm font-semibold text-slate-900 dark:text-slate-100">GitHub 认证</h2>
        <p class="mt-1 text-xs text-slate-400 dark:text-slate-500">
          需要 <span class="font-medium">repo</span> 与 <span class="font-medium">read:user</span> 权限的 Personal Access Token，
          用于同步 Star 与查询 Release。
        </p>
        <div class="relative mt-4">
          <input
            v-model="form.github_token"
            :type="showToken ? 'text' : 'password'"
            placeholder="ghp_..."
            autocomplete="off"
            class="w-full rounded-xl border border-slate-300 py-2.5 pl-3.5 pr-11 text-sm outline-none transition focus:border-sky-500 focus:ring-2 focus:ring-sky-100 dark:border-slate-700 dark:bg-slate-950 dark:text-slate-100 dark:focus:ring-sky-900"
          />
          <button
            type="button"
            class="absolute right-2.5 top-2 text-slate-400 transition hover:text-slate-600 dark:hover:text-slate-300"
            @click="showToken = !showToken"
          >
            <svg v-if="!showToken" class="h-5 w-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8">
              <path stroke-linecap="round" stroke-linejoin="round" d="M2.036 12.322a1.012 1.012 0 0 1 0-.639C3.423 7.51 7.36 4.5 12 4.5c4.638 0 8.573 3.007 9.963 7.178.07.207.07.431 0 .639C20.577 16.49 16.64 19.5 12 19.5c-4.638 0-8.573-3.007-9.963-7.178z" />
              <path stroke-linecap="round" stroke-linejoin="round" d="M15 12a3 3 0 1 1-6 0 3 3 0 0 1 6 0z" />
            </svg>
            <svg v-else class="h-5 w-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8">
              <path stroke-linecap="round" stroke-linejoin="round" d="M3.98 8.223A10.477 10.477 0 0 0 1.934 12C3.226 16.338 7.244 19.5 12 19.5c.993 0 1.953-.138 2.863-.395M6.228 6.228A10.451 10.451 0 0 1 12 4.5c4.756 0 8.773 3.162 10.065 7.498a10.522 10.522 0 0 1-4.293 5.774M6.228 6.228L3 3m3.228 3.228l3.65 3.65m7.894 7.894L21 21m-3.228-3.228l-3.65-3.65m0 0a3 3 0 1 0-4.243-4.243m4.242 4.242L9.88 9.88" />
            </svg>
          </button>
        </div>
        <div v-if="saveResult" class="mt-3 flex items-center gap-2 text-xs">
          <span
            class="rounded-full px-2.5 py-1 font-medium"
            :class="saveResult.token_valid ? 'bg-emerald-50 text-emerald-700 dark:bg-emerald-900/40 dark:text-emerald-400' : 'bg-rose-50 text-rose-700 dark:bg-rose-900/40 dark:text-rose-400'"
          >
            {{ saveResult.token_valid ? 'Token 有效' : 'Token 无效' }}
          </span>
          <span v-if="saveResult.username" class="text-slate-500 dark:text-slate-400">已认证用户：{{ saveResult.username }}</span>
          <span v-else-if="saveResult.token_error" class="max-w-sm truncate text-slate-400 dark:text-slate-500">{{ saveResult.token_error }}</span>
        </div>
      </section>

      <section class="rounded-2xl border border-slate-200 bg-white p-5 shadow-sm dark:border-slate-800 dark:bg-slate-900">
        <h2 class="text-sm font-semibold text-slate-900 dark:text-slate-100">通知</h2>
        <p class="mt-1 text-xs text-slate-400 dark:text-slate-500">
          Shoutrrr 服务 URL，例如 Telegram、Discord、Slack 等（<a
            class="text-sky-600 hover:underline dark:text-sky-400"
            href="https://containrrr.dev/shoutrrr/"
            target="_blank"
            rel="noopener"
          >格式参考</a>）。
        </p>
        <input
          v-model="form.shoutrrr_url"
          type="text"
          placeholder="telegram://bot_token@telegram?channels=channel_id"
          class="mt-4 w-full rounded-xl border border-slate-300 px-3.5 py-2.5 font-mono text-sm outline-none transition focus:border-sky-500 focus:ring-2 focus:ring-sky-100 dark:border-slate-700 dark:bg-slate-950 dark:text-slate-100 dark:focus:ring-sky-900"
        />
      </section>

      <section class="rounded-2xl border border-slate-200 bg-white p-5 shadow-sm dark:border-slate-800 dark:bg-slate-900">
        <h2 class="text-sm font-semibold text-slate-900 dark:text-slate-100">外观</h2>
        <p class="mt-1 text-xs text-slate-400 dark:text-slate-500">选择界面主题，立即生效，无需保存</p>
        <div class="mt-4 grid gap-3 sm:grid-cols-3">
          <button
            v-for="o in themeOptions"
            :key="o.value"
            type="button"
            class="rounded-xl border p-3 text-left transition"
            :class="
              themeMode === o.value
                ? 'border-sky-500 bg-sky-50 ring-2 ring-sky-100 dark:bg-sky-950/40 dark:ring-sky-900'
                : 'border-slate-200 hover:border-slate-300 dark:border-slate-700 dark:hover:border-slate-600'
            "
            @click="setThemeMode(o.value)"
          >
            <div class="flex items-center gap-2">
              <span
                class="flex h-5 w-5 items-center justify-center rounded-full border"
                :class="themeMode === o.value ? 'border-sky-600' : 'border-slate-300 dark:border-slate-600'"
              >
                <span v-if="themeMode === o.value" class="h-2.5 w-2.5 rounded-full bg-sky-600" />
              </span>
              <span class="text-sm font-medium text-slate-800 dark:text-slate-100">{{ o.label }}</span>
            </div>
            <div class="mt-1.5 pl-7 text-xs text-slate-400 dark:text-slate-500">{{ o.desc }}</div>
          </button>
        </div>
      </section>

      <section class="rounded-2xl border border-slate-200 bg-white p-5 shadow-sm dark:border-slate-800 dark:bg-slate-900">
        <h2 class="text-sm font-semibold text-slate-900 dark:text-slate-100">RSS 订阅</h2>
        <p class="mt-1 text-xs text-slate-400 dark:text-slate-500">
          在任意 RSS 阅读器中订阅以下地址，即可聚合查看所有被监控仓库的最新版本发布。
        </p>
        <div class="mt-4 flex items-center gap-2">
          <input
            :value="feedUrl"
            readonly
            class="w-full rounded-xl border border-slate-300 bg-slate-50 px-3.5 py-2.5 font-mono text-sm text-slate-700 outline-none dark:border-slate-700 dark:bg-slate-950 dark:text-slate-200"
            @focus="(e: FocusEvent) => (e.target as HTMLInputElement).select()"
          />
          <button
            type="button"
            class="inline-flex shrink-0 items-center gap-1.5 rounded-xl bg-slate-900 px-4 py-2.5 text-sm font-medium text-white shadow-sm transition hover:bg-slate-700 dark:bg-slate-100 dark:text-slate-900 dark:hover:bg-white"
            @click="copyFeed"
          >
            <svg class="h-4 w-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8">
              <path stroke-linecap="round" stroke-linejoin="round" d="M8.25 7.5V6.108c0-1.135.845-2.098 1.976-2.192.373-.03.748-.057 1.123-.08M15.75 18H18a2.25 2.25 0 0 0 2.25-2.25V6.108c0-1.135-.845-2.098-1.976-2.192a48.424 48.424 0 0 0-1.123-.08M15.75 18.75v-1.875a3.375 3.375 0 0 0-3.375-3.375h-1.5a1.125 1.125 0 0 1-1.125-1.125v-1.5A3.375 3.375 0 0 0 6.375 7.5H5.25m11.9-3.664A2.251 2.251 0 0 0 15 2.25h-1.5a2.251 2.251 0 0 0-2.25 2.25v3.75m6 12a3 3 0 1 1-6 0 3 3 0 0 1 6 0zM2.25 19.5h.008v.008H2.25v-.008z" />
            </svg>
            复制
          </button>
        </div>
      </section>

      <section class="rounded-2xl border border-slate-200 bg-white p-5 shadow-sm dark:border-slate-800 dark:bg-slate-900">
        <h2 class="text-sm font-semibold text-slate-900 dark:text-slate-100">轮询策略</h2>
        <div class="mt-4 grid gap-5 sm:grid-cols-2">
          <div>
            <label class="text-xs font-medium text-slate-500 dark:text-slate-400">检查间隔</label>
            <select
              v-model="form.poll_interval"
              class="mt-1.5 w-full rounded-xl border border-slate-300 bg-white px-3 py-2.5 text-sm outline-none transition focus:border-sky-500 dark:border-slate-700 dark:bg-slate-950 dark:text-slate-100"
            >
              <option v-for="o in intervalOptions" :key="o.value" :value="o.value">{{ o.label }}</option>
            </select>
          </div>
          <div>
            <label class="text-xs font-medium text-slate-500 dark:text-slate-400">GitHub API Base URL（可选）</label>
            <input
              v-model="form.github_api_base_url"
              type="text"
              placeholder="https://api.github.com/（GitHub Enterprise 请填写）"
              class="mt-1.5 w-full rounded-xl border border-slate-300 px-3.5 py-2.5 text-sm outline-none transition focus:border-sky-500 dark:border-slate-700 dark:bg-slate-950 dark:text-slate-100"
            />
          </div>
          <div>
            <label class="text-xs font-medium text-slate-500 dark:text-slate-400">通知记录保留条数（0 为不限制）</label>
            <input
              v-model.number="form.max_notifications"
              type="number"
              min="0"
              placeholder="0"
              class="mt-1.5 w-full rounded-xl border border-slate-300 px-3.5 py-2.5 text-sm outline-none transition focus:border-sky-500 dark:border-slate-700 dark:bg-slate-950 dark:text-slate-100"
            />
          </div>
        </div>
        <div class="mt-5 flex items-center justify-between rounded-xl bg-slate-50 px-4 py-3 dark:bg-slate-800/60">
          <div>
            <div class="text-sm font-medium text-slate-700 dark:text-slate-200">首次监控时通知历史最新版本</div>
            <div class="text-xs text-slate-400 dark:text-slate-500">默认首次监控仅建立基线不通知，开启后会把已有最新版也发送一次</div>
          </div>
          <ToggleSwitch v-model="form.notify_on_first_run" />
        </div>
      </section>

      <section class="rounded-2xl border border-slate-200 bg-white p-5 shadow-sm dark:border-slate-800 dark:bg-slate-900">
        <h2 class="text-sm font-semibold text-slate-900 dark:text-slate-100">更新日志翻译</h2>
        <p class="mt-1 text-xs text-slate-400 dark:text-slate-500">
          检测更新日志语言：已是目标语言则直接使用/提取，否则自动翻译。通知与记录页都会使用翻译结果。
        </p>
        <div class="mt-4 grid gap-5 sm:grid-cols-2">
          <div>
            <label class="text-xs font-medium text-slate-500 dark:text-slate-400">翻译引擎</label>
            <select
              v-model="form.translate_engine"
              class="mt-1.5 w-full rounded-xl border border-slate-300 bg-white px-3 py-2.5 text-sm outline-none transition focus:border-sky-500 dark:border-slate-700 dark:bg-slate-950 dark:text-slate-100"
            >
              <option v-for="o in engineOptions" :key="o.value" :value="o.value">{{ o.label }}</option>
            </select>
          </div>
          <div>
            <label class="text-xs font-medium text-slate-500 dark:text-slate-400">目标语言</label>
            <select
              v-model="form.translate_target_lang"
              :disabled="form.translate_engine === 'off'"
              class="mt-1.5 w-full rounded-xl border border-slate-300 bg-white px-3 py-2.5 text-sm outline-none transition focus:border-sky-500 disabled:opacity-50 dark:border-slate-700 dark:bg-slate-950 dark:text-slate-100"
            >
              <option v-for="o in langOptions" :key="o.value" :value="o.value">{{ o.label }}</option>
            </select>
          </div>
          <div v-if="form.translate_engine === 'dlx'">
            <label class="text-xs font-medium text-slate-500 dark:text-slate-400">DLX 服务地址</label>
            <input
              v-model="form.translate_url"
              type="text"
              placeholder="http://localhost:1188"
              class="mt-1.5 w-full rounded-xl border border-slate-300 px-3.5 py-2.5 font-mono text-sm outline-none transition focus:border-sky-500 dark:border-slate-700 dark:bg-slate-950 dark:text-slate-100"
            />
          </div>
          <template v-if="form.translate_engine === 'openai'">
            <div>
              <label class="text-xs font-medium text-slate-500 dark:text-slate-400">API Base URL</label>
              <input
                v-model="form.translate_url"
                type="text"
                placeholder="https://api.openai.com/v1"
                class="mt-1.5 w-full rounded-xl border border-slate-300 px-3.5 py-2.5 font-mono text-sm outline-none transition focus:border-sky-500 dark:border-slate-700 dark:bg-slate-950 dark:text-slate-100"
              />
            </div>
            <div>
              <label class="text-xs font-medium text-slate-500 dark:text-slate-400">API Key（本地网关可留空）</label>
              <input
                v-model="form.translate_api_key"
                type="password"
                autocomplete="off"
                placeholder="sk-..."
                class="mt-1.5 w-full rounded-xl border border-slate-300 px-3.5 py-2.5 font-mono text-sm outline-none transition focus:border-sky-500 dark:border-slate-700 dark:bg-slate-950 dark:text-slate-100"
              />
            </div>
            <div>
              <label class="text-xs font-medium text-slate-500 dark:text-slate-400">模型</label>
              <input
                v-model="form.translate_model"
                type="text"
                placeholder="gpt-4o-mini"
                class="mt-1.5 w-full rounded-xl border border-slate-300 px-3.5 py-2.5 font-mono text-sm outline-none transition focus:border-sky-500 dark:border-slate-700 dark:bg-slate-950 dark:text-slate-100"
              />
            </div>
          </template>
          <div v-if="form.translate_engine === 'google'">
            <label class="text-xs font-medium text-slate-500 dark:text-slate-400">Google 接口地址</label>
            <input
              v-model="form.translate_url"
              type="text"
              placeholder="https://translate.googleapis.com"
              class="mt-1.5 w-full rounded-xl border border-slate-300 px-3.5 py-2.5 font-mono text-sm outline-none transition focus:border-sky-500 dark:border-slate-700 dark:bg-slate-950 dark:text-slate-100"
            />
          </div>
          <div v-if="form.translate_engine === 'bing'" class="sm:col-span-2">
            <p class="text-xs text-slate-400 dark:text-slate-500">
              必应翻译接口（api-edge.cognitive.microsofttranslator.com）已不再免密钥开放，需要注册 Azure Translator 并携带订阅密钥方可调用。
            </p>
          </div>
        </div>
      </section>

      <div class="flex flex-wrap items-center gap-3 pb-10">
        <button
          type="submit"
          :disabled="saving"
          class="inline-flex items-center gap-2 rounded-xl bg-sky-600 px-6 py-2.5 text-sm font-medium text-white shadow-sm transition hover:bg-sky-500 disabled:opacity-50"
        >
          <svg v-if="saving" class="h-4 w-4 animate-spin" viewBox="0 0 24 24" fill="none">
            <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" />
            <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 0 1 8-8V0C5.373 0 0 5.373 0 12h4z" />
          </svg>
          保存并验证
        </button>
        <button
          type="button"
          :disabled="testing"
          class="inline-flex items-center gap-2 rounded-xl border border-slate-300 bg-white px-6 py-2.5 text-sm font-medium text-slate-700 shadow-sm transition hover:bg-slate-50 disabled:opacity-50 dark:border-slate-700 dark:bg-slate-900 dark:text-slate-200 dark:hover:bg-slate-800"
          @click="testNotify"
        >
          {{ testing ? '发送中...' : '发送测试通知' }}
        </button>
        <button
          type="button"
          :disabled="backing"
          class="inline-flex items-center gap-2 rounded-xl border border-slate-300 bg-white px-6 py-2.5 text-sm font-medium text-slate-700 shadow-sm transition hover:bg-slate-50 disabled:opacity-50 dark:border-slate-700 dark:bg-slate-900 dark:text-slate-200 dark:hover:bg-slate-800"
          @click="exportBackup"
        >
          {{ backing ? '导出中...' : '导出配置' }}
        </button>
        <button
          type="button"
          :disabled="restoring"
          class="inline-flex items-center gap-2 rounded-xl border border-slate-300 bg-white px-6 py-2.5 text-sm font-medium text-slate-700 shadow-sm transition hover:bg-slate-50 disabled:opacity-50 dark:border-slate-700 dark:bg-slate-900 dark:text-slate-200 dark:hover:bg-slate-800"
          @click="pickBackup"
        >
          {{ restoring ? '导入中...' : '导入配置' }}
        </button>
        <input ref="fileInput" type="file" accept="application/json,.json" class="hidden" @change="importBackup" />
      </div>
    </form>
  </div>
</template>
