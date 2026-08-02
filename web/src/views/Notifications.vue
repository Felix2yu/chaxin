<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { api } from '../api'
import { useToast } from '../components/toast'
import type { Notification } from '../types'

const toast = useToast()
const loading = ref(true)
const limit = ref(100)
const query = ref('')
const status = ref('')
const items = ref<Notification[]>([])
const expanded = ref<number | null>(null)
const retrying = ref<number | null>(null)

function toggleLog(id: number) {
  expanded.value = expanded.value === id ? null : id
}

async function load() {
  loading.value = true
  try {
    items.value = await api.listNotifications({ limit: limit.value, query: query.value, status: status.value })
  } catch (e) {
    toast.error((e as Error).message)
  } finally {
    loading.value = false
  }
}

async function retry(n: Notification) {
  retrying.value = n.id
  try {
    await api.retryNotification(n.id)
    toast.success(`已重新发送 ${n.full_name} ${n.tag}`)
    await load()
  } catch (e) {
    toast.error((e as Error).message)
  } finally {
    retrying.value = null
  }
}

function fmtTime(s?: string) {
  if (!s) return '-'
  return new Date(s).toLocaleString('zh-CN', { hour12: false })
}

onMounted(load)
</script>

<template>
  <div class="mx-auto max-w-6xl p-4 md:p-8">
    <div class="mb-6 flex flex-wrap items-center justify-between gap-3">
      <div>
        <h1 class="text-2xl font-bold text-slate-900 dark:text-slate-100">通知记录</h1>
        <p class="mt-1 text-sm text-slate-500 dark:text-slate-400">通过 shoutrrr 发送的历史通知</p>
      </div>
      <div class="flex flex-wrap items-center gap-2">
        <div class="relative">
          <svg class="pointer-events-none absolute left-3 top-2.5 h-4 w-4 text-slate-400" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path stroke-linecap="round" stroke-linejoin="round" d="M21 21l-5.197-5.197m0 0A7.5 7.5 0 1 0 5.196 5.196a7.5 7.5 0 0 0 10.607 10.607z" />
          </svg>
          <input
            v-model="query"
            type="text"
            placeholder="搜索仓库或版本..."
            class="w-52 rounded-xl border border-slate-300 bg-white py-2 pl-9 pr-3 text-sm outline-none transition focus:border-sky-500 focus:ring-2 focus:ring-sky-100 dark:border-slate-700 dark:bg-slate-900 dark:text-slate-100"
            @keyup.enter="load"
          />
        </div>
        <select
          v-model="status"
          class="rounded-xl border border-slate-300 bg-white px-3 py-2 text-sm outline-none transition focus:border-sky-500 dark:border-slate-700 dark:bg-slate-900 dark:text-slate-100"
          @change="load"
        >
          <option value="">全部状态</option>
          <option value="sent">已发送</option>
          <option value="failed">失败</option>
        </select>
        <select
          v-model.number="limit"
          class="rounded-xl border border-slate-300 bg-white px-3 py-2 text-sm outline-none transition focus:border-sky-500 dark:border-slate-700 dark:bg-slate-900 dark:text-slate-100"
          @change="load"
        >
          <option :value="50">最近 50 条</option>
          <option :value="100">最近 100 条</option>
          <option :value="200">最近 200 条</option>
        </select>
        <button
          class="inline-flex items-center gap-2 rounded-xl bg-slate-900 px-4 py-2 text-sm font-medium text-white shadow-sm transition hover:bg-slate-700 dark:bg-slate-100 dark:text-slate-900 dark:hover:bg-white"
          @click="load"
        >
          <svg class="h-4 w-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path stroke-linecap="round" stroke-linejoin="round" d="M16.023 9.348h4.992v-.001M2.985 19.644v-4.992m0 0h4.992m-4.993 0l3.181 3.183a8.25 8.25 0 0 0 13.803-3.7M4.031 9.865a8.25 8.25 0 0 1 13.803-3.7l3.181 3.182m0-4.991v4.99" />
          </svg>
          刷新
        </button>
      </div>
    </div>

    <div class="overflow-hidden rounded-2xl border border-slate-200 bg-white shadow-sm dark:border-slate-800 dark:bg-slate-900">
      <div class="overflow-x-auto">
        <table class="w-full text-left text-sm">
          <thead>
            <tr class="border-b border-slate-100 text-xs text-slate-400 dark:border-slate-800 dark:text-slate-500">
              <th class="px-5 py-3 font-medium">仓库</th>
              <th class="px-4 py-3 font-medium">版本</th>
              <th class="hidden px-4 py-3 font-medium md:table-cell">发布时间</th>
              <th class="hidden px-4 py-3 font-medium sm:table-cell">通知时间</th>
              <th class="px-4 py-3 font-medium">状态</th>
              <th class="px-4 py-3 text-right font-medium">日志</th>
            </tr>
          </thead>
          <tbody v-if="!loading && items.length">
            <template v-for="n in items" :key="n.id">
              <tr class="border-b border-slate-50 transition hover:bg-slate-50/70 dark:border-slate-800/60 dark:hover:bg-slate-800/40">
                <td class="px-5 py-3.5 font-medium text-slate-900 dark:text-slate-100">{{ n.full_name }}</td>
                <td class="px-4 py-3.5">
                  <a
                    v-if="n.release_url"
                    :href="n.release_url"
                    target="_blank"
                    rel="noopener"
                    class="rounded-lg bg-slate-100 px-2.5 py-1 font-mono text-xs font-semibold text-sky-700 hover:bg-sky-50 dark:bg-slate-800 dark:text-sky-400 dark:hover:bg-slate-700"
                  >
                    {{ n.tag }}
                  </a>
                  <span v-else class="rounded-lg bg-slate-100 px-2.5 py-1 font-mono text-xs font-medium text-slate-700 dark:bg-slate-800 dark:text-slate-300">
                    {{ n.tag }}
                  </span>
                </td>
                <td class="hidden px-4 py-3.5 text-slate-500 dark:text-slate-400 md:table-cell">{{ fmtTime(n.released_at) }}</td>
                <td class="hidden px-4 py-3.5 text-slate-500 dark:text-slate-400 sm:table-cell">{{ fmtTime(n.sent_at) }}</td>
                <td class="px-4 py-3.5">
                  <div class="flex items-center gap-2">
                    <span
                      class="inline-flex items-center gap-1.5 rounded-full px-2.5 py-1 text-xs font-medium"
                      :class="n.status === 'sent' ? 'bg-emerald-50 text-emerald-700 dark:bg-emerald-900/40 dark:text-emerald-400' : 'bg-rose-50 text-rose-700 dark:bg-rose-900/40 dark:text-rose-400'"
                    >
                      <span class="h-1.5 w-1.5 rounded-full" :class="n.status === 'sent' ? 'bg-emerald-500' : 'bg-rose-500'" />
                      {{ n.status === 'sent' ? '已发送' : '失败' }}
                    </span>
                    <button
                      v-if="n.status !== 'sent'"
                      :disabled="retrying === n.id"
                      class="rounded-lg bg-sky-50 px-2 py-1 text-xs font-medium text-sky-700 transition hover:bg-sky-100 disabled:opacity-50 dark:bg-sky-950/50 dark:text-sky-400 dark:hover:bg-sky-900/50"
                      @click="retry(n)"
                    >
                      {{ retrying === n.id ? '重发中...' : '重发' }}
                    </button>
                  </div>
                  <div v-if="n.status !== 'sent' && n.error" class="mt-1 max-w-xs truncate text-xs text-rose-500 dark:text-rose-400">
                    {{ n.error }}
                  </div>
                </td>
                <td class="px-4 py-3.5 text-right">
                  <button
                    v-if="n.release_body"
                    class="inline-flex items-center gap-1 rounded-lg px-2 py-1 text-xs font-medium text-slate-500 transition hover:bg-slate-100 hover:text-slate-800 dark:text-slate-400 dark:hover:bg-slate-800 dark:hover:text-slate-100"
                    @click="toggleLog(n.id)"
                  >
                    {{ expanded === n.id ? '收起' : '查看' }}
                    <svg
                      class="h-3.5 w-3.5 transition-transform"
                      :class="expanded === n.id ? 'rotate-180' : ''"
                      viewBox="0 0 24 24"
                      fill="none"
                      stroke="currentColor"
                      stroke-width="2"
                    >
                      <path stroke-linecap="round" stroke-linejoin="round" d="M19.5 8.25l-7.5 7.5-7.5-7.5" />
                    </svg>
                  </button>
                </td>
              </tr>
              <tr v-if="expanded === n.id" class="border-b border-slate-50 bg-slate-50/60 dark:border-slate-800/60 dark:bg-slate-800/40">
                <td colspan="6" class="px-5 py-4">
                  <div class="mb-2 text-xs font-medium text-slate-400 dark:text-slate-500">更新日志 · {{ n.full_name }} {{ n.tag }}</div>
                  <pre class="max-h-80 overflow-auto whitespace-pre-wrap break-words rounded-xl bg-white p-4 text-sm leading-relaxed text-slate-700 shadow-inner dark:bg-slate-950 dark:text-slate-300">{{ n.release_body }}</pre>
                </td>
              </tr>
            </template>
          </tbody>
        </table>
      </div>
      <div v-if="loading" class="space-y-3 p-5">
        <div v-for="i in 6" :key="i" class="h-12 animate-pulse rounded-xl bg-slate-100 dark:bg-slate-800" />
      </div>
      <div v-else-if="items.length === 0" class="p-12 text-center text-sm text-slate-400 dark:text-slate-500">
        暂无通知记录。
      </div>
    </div>
  </div>
</template>
