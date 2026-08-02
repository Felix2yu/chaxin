<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { api } from '../api'
import { useToast } from '../components/toast'
import ToggleSwitch from '../components/ToggleSwitch.vue'
import type { Settings, SettingsSaveResult } from '../types'

const toast = useToast()
const loading = ref(true)
const saving = ref(false)
const testing = ref(false)

const form = ref<Settings>({
  github_token: '',
  shoutrrr_url: '',
  poll_interval: '30m',
  notify_on_first_run: false,
  github_api_base_url: '',
})

const showToken = ref(false)
const saveResult = ref<SettingsSaveResult['verify'] | null>(null)

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

onMounted(load)
</script>

<template>
  <div class="mx-auto max-w-3xl p-4 md:p-8">
    <div class="mb-6">
      <h1 class="text-2xl font-bold text-slate-900">设置</h1>
      <p class="mt-1 text-sm text-slate-500">配置 GitHub 认证、通知目标与轮询策略</p>
    </div>

    <div v-if="loading" class="space-y-3">
      <div v-for="i in 4" :key="i" class="h-20 animate-pulse rounded-2xl bg-slate-100" />
    </div>

    <form v-else class="space-y-5" @submit.prevent="save">
      <section class="rounded-2xl border border-slate-200 bg-white p-5 shadow-sm">
        <h2 class="text-sm font-semibold text-slate-900">GitHub 认证</h2>
        <p class="mt-1 text-xs text-slate-400">
          需要 <span class="font-medium">repo</span> 与 <span class="font-medium">read:user</span> 权限的 Personal Access Token，
          用于同步 Star 与查询 Release。
        </p>
        <div class="relative mt-4">
          <input
            v-model="form.github_token"
            :type="showToken ? 'text' : 'password'"
            placeholder="ghp_..."
            autocomplete="off"
            class="w-full rounded-xl border border-slate-300 py-2.5 pl-3.5 pr-11 text-sm outline-none transition focus:border-sky-500 focus:ring-2 focus:ring-sky-100"
          />
          <button
            type="button"
            class="absolute right-2.5 top-2 text-slate-400 transition hover:text-slate-600"
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
            :class="saveResult.token_valid ? 'bg-emerald-50 text-emerald-700' : 'bg-rose-50 text-rose-700'"
          >
            {{ saveResult.token_valid ? 'Token 有效' : 'Token 无效' }}
          </span>
          <span v-if="saveResult.username" class="text-slate-500">已认证用户：{{ saveResult.username }}</span>
          <span v-else-if="saveResult.token_error" class="max-w-sm truncate text-slate-400">{{ saveResult.token_error }}</span>
        </div>
      </section>

      <section class="rounded-2xl border border-slate-200 bg-white p-5 shadow-sm">
        <h2 class="text-sm font-semibold text-slate-900">通知</h2>
        <p class="mt-1 text-xs text-slate-400">
          Shoutrrr 服务 URL，例如 Telegram、Discord、Slack 等（<a
            class="text-sky-600 hover:underline"
            href="https://containrrr.dev/shoutrrr/"
            target="_blank"
            rel="noopener"
          >格式参考</a>）。
        </p>
        <input
          v-model="form.shoutrrr_url"
          type="text"
          placeholder="telegram://bot_token@telegram?channels=channel_id"
          class="mt-4 w-full rounded-xl border border-slate-300 px-3.5 py-2.5 font-mono text-sm outline-none transition focus:border-sky-500 focus:ring-2 focus:ring-sky-100"
        />
      </section>

      <section class="rounded-2xl border border-slate-200 bg-white p-5 shadow-sm">
        <h2 class="text-sm font-semibold text-slate-900">轮询策略</h2>
        <div class="mt-4 grid gap-5 sm:grid-cols-2">
          <div>
            <label class="text-xs font-medium text-slate-500">检查间隔</label>
            <select
              v-model="form.poll_interval"
              class="mt-1.5 w-full rounded-xl border border-slate-300 bg-white px-3 py-2.5 text-sm outline-none transition focus:border-sky-500"
            >
              <option v-for="o in intervalOptions" :key="o.value" :value="o.value">{{ o.label }}</option>
            </select>
          </div>
          <div>
            <label class="text-xs font-medium text-slate-500">GitHub API Base URL（可选）</label>
            <input
              v-model="form.github_api_base_url"
              type="text"
              placeholder="https://api.github.com/（GitHub Enterprise 请填写）"
              class="mt-1.5 w-full rounded-xl border border-slate-300 px-3.5 py-2.5 text-sm outline-none transition focus:border-sky-500"
            />
          </div>
        </div>
        <div class="mt-5 flex items-center justify-between rounded-xl bg-slate-50 px-4 py-3">
          <div>
            <div class="text-sm font-medium text-slate-700">首次监控时通知历史最新版本</div>
            <div class="text-xs text-slate-400">默认首次监控仅建立基线不通知，开启后会把已有最新版也发送一次</div>
          </div>
          <ToggleSwitch v-model="form.notify_on_first_run" />
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
          class="inline-flex items-center gap-2 rounded-xl border border-slate-300 bg-white px-6 py-2.5 text-sm font-medium text-slate-700 shadow-sm transition hover:bg-slate-50 disabled:opacity-50"
          @click="testNotify"
        >
          {{ testing ? '发送中...' : '发送测试通知' }}
        </button>
      </div>
    </form>
  </div>
</template>
