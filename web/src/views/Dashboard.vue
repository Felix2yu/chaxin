<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { api } from '../api'
import { useToast } from '../components/toast'
import type { Notification, Repo } from '../types'

const toast = useToast()
const loading = ref(true)
const running = ref(false)
const repos = ref<Repo[]>([])
const notifications = ref<Notification[]>([])

const stats = computed(() => {
  const monitored = repos.value.filter((r) => r.monitored).length
  let lastCheck = ''
  for (const r of repos.value) {
    if (r.last_checked_at && (!lastCheck || r.last_checked_at > lastCheck)) lastCheck = r.last_checked_at
  }
  return {
    total: repos.value.length,
    monitored,
    lastCheck,
  }
})

const failedCount = computed(() => notifications.value.filter((n) => n.status === 'failed').length)

async function load() {
  loading.value = true
  try {
    const [r, n] = await Promise.all([api.listRepos(), api.listNotifications({ limit: 10 })])
    repos.value = r
    notifications.value = n
  } catch (e) {
    toast.error((e as Error).message)
  } finally {
    loading.value = false
  }
}

async function runNow() {
  running.value = true
  try {
    await api.runMonitor()
    toast.success('本轮检查完成')
    await load()
  } catch (e) {
    toast.error((e as Error).message)
  } finally {
    running.value = false
  }
}

function fmtTime(s?: string) {
  if (!s) return '-'
  const d = new Date(s)
  return d.toLocaleString('zh-CN', { hour12: false })
}

onMounted(load)
</script>

<template>
  <div class="mx-auto max-w-6xl p-4 md:p-8">
    <div class="mb-6 flex flex-wrap items-center justify-between gap-3">
      <div>
        <h1 class="text-2xl font-bold text-slate-900 dark:text-slate-100">概览</h1>
        <p class="mt-1 text-sm text-slate-500 dark:text-slate-400">监控中仓库的最新版本与最近通知</p>
      </div>
      <button
        class="inline-flex items-center gap-2 rounded-xl bg-slate-900 px-4 py-2.5 text-sm font-medium text-white shadow-sm transition hover:bg-slate-700 dark:bg-slate-100 dark:text-slate-900 dark:hover:bg-white disabled:opacity-50"
        :disabled="running"
        @click="runNow"
      >
        <svg v-if="running" class="h-4 w-4 animate-spin" viewBox="0 0 24 24" fill="none">
          <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" />
          <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 0 1 8-8V0C5.373 0 0 5.373 0 12h4z" />
        </svg>
        <svg v-else class="h-4 w-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path stroke-linecap="round" stroke-linejoin="round" d="M16.023 9.348h4.992v-.001M2.985 19.644v-4.992m0 0h4.992m-4.993 0l3.181 3.183a8.25 8.25 0 0 0 13.803-3.7M4.031 9.865a8.25 8.25 0 0 1 13.803-3.7l3.181 3.182m0-4.991v4.99" />
        </svg>
        {{ running ? '检查中...' : '立即检查' }}
      </button>
    </div>

    <div class="grid grid-cols-2 gap-4 lg:grid-cols-4">
      <div class="rounded-2xl border border-slate-200 bg-white p-5 shadow-sm dark:border-slate-800 dark:bg-slate-900">
        <div class="text-sm text-slate-500 dark:text-slate-400">监控仓库</div>
        <div class="mt-2 text-3xl font-bold text-slate-900 dark:text-slate-100">{{ stats.monitored }}</div>
      </div>
      <div class="rounded-2xl border border-slate-200 bg-white p-5 shadow-sm dark:border-slate-800 dark:bg-slate-900">
        <div class="text-sm text-slate-500 dark:text-slate-400">仓库总数</div>
        <div class="mt-2 text-3xl font-bold text-slate-900 dark:text-slate-100">{{ stats.total }}</div>
      </div>
      <div class="rounded-2xl border border-slate-200 bg-white p-5 shadow-sm dark:border-slate-800 dark:bg-slate-900">
        <div class="text-sm text-slate-500 dark:text-slate-400">通知失败</div>
        <div class="mt-2 text-3xl font-bold" :class="failedCount > 0 ? 'text-rose-600 dark:text-rose-400' : 'text-slate-900 dark:text-slate-100'">
          {{ failedCount }}
        </div>
      </div>
      <div class="rounded-2xl border border-slate-200 bg-white p-5 shadow-sm dark:border-slate-800 dark:bg-slate-900">
        <div class="text-sm text-slate-500 dark:text-slate-400">上次检查</div>
        <div class="mt-2 text-lg font-bold leading-8 text-slate-900 dark:text-slate-100">{{ fmtTime(stats.lastCheck) }}</div>
      </div>
    </div>

    <div class="mt-8 rounded-2xl border border-slate-200 bg-white shadow-sm dark:border-slate-800 dark:bg-slate-900">
      <div class="flex items-center justify-between border-b border-slate-100 px-5 py-4 dark:border-slate-800">
        <h2 class="font-semibold text-slate-900 dark:text-slate-100">最近通知</h2>
        <RouterLink to="/notifications" class="text-sm font-medium text-sky-600 hover:text-sky-700 dark:text-sky-400 dark:hover:text-sky-300">
          查看全部
        </RouterLink>
      </div>
      <div v-if="loading" class="space-y-3 p-5">
        <div v-for="i in 4" :key="i" class="h-14 animate-pulse rounded-xl bg-slate-100 dark:bg-slate-800" />
      </div>
      <div v-else-if="notifications.length === 0" class="p-10 text-center text-sm text-slate-400 dark:text-slate-500">
        暂无通知记录。配置完成后点击「立即检查」触发首次扫描。
      </div>
      <ul v-else class="divide-y divide-slate-100 dark:divide-slate-800">
        <li v-for="n in notifications" :key="n.id" class="flex items-center gap-3 px-5 py-3.5">
          <span
            class="h-2.5 w-2.5 shrink-0 rounded-full"
            :class="n.status === 'sent' ? 'bg-emerald-500' : 'bg-rose-500'"
          />
          <div class="min-w-0 flex-1">
            <div class="truncate text-sm font-medium text-slate-900 dark:text-slate-100">
              <span class="text-slate-400 dark:text-slate-500">{{ n.full_name }}</span>
              <span class="mx-1.5 text-slate-300 dark:text-slate-600">→</span>
              <span class="text-sky-600 dark:text-sky-400">{{ n.tag }}</span>
            </div>
            <div v-if="n.status === 'failed' && n.error" class="mt-0.5 truncate text-xs text-rose-500 dark:text-rose-400">
              {{ n.error }}
            </div>
          </div>
          <div class="shrink-0 text-xs text-slate-400 dark:text-slate-500">{{ fmtTime(n.sent_at) }}</div>
        </li>
      </ul>
    </div>
  </div>
</template>
