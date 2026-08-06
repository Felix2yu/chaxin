<script setup lang="ts">
import { onMounted, ref, computed } from 'vue'
import { api } from '../api'
import type { Notification } from '../types'
import { useToast } from '../components/toast'

const toast = useToast()
const notifications = ref<Notification[]>([])
const loading = ref(true)
const search = ref('')
const statusFilter = ref('')
const expanded = ref<Set<number>>(new Set())
const translating = ref<Set<number>>(new Set())
const translateCache = ref<Map<number, string>>(new Map())

async function load() {
  loading.value = true
  try {
    const params: any = { limit: 200 }
    if (search.value) params.query = search.value
    if (statusFilter.value) params.status = statusFilter.value
    notifications.value = await api.listNotifications(params)
  } catch (e: any) {
    toast.error(e.message || '加载失败')
  } finally {
    loading.value = false
  }
}

function toggleExpand(id: number) {
  if (expanded.value.has(id)) expanded.value.delete(id)
  else expanded.value.add(id)
}

async function doTranslate(n: Notification) {
  if (!n.release_body || translating.value.has(n.id)) return
  translating.value.add(n.id)
  try {
    const result = await api.translate(n.release_body, 'zh')
    translateCache.value.set(n.id, result.text)
  } catch (e: any) {
    toast.error(e.message || '翻译失败')
  } finally {
    translating.value.delete(n.id)
  }
}

async function doRetry(id: number) {
  try {
    await api.retryNotification(id)
    await load()
    toast.success('已重试')
  } catch (e: any) {
    toast.error(e.message)
  }
}

function formatTime(t: string) {
  if (!t) return '-'
  return new Date(t).toLocaleString('zh-CN')
}

function formatBody(text: string) {
  if (!text) return ''
  return text.replace(/^#+\s+(.*)/gm, '<strong>$1</strong>')
}

const getStatusBadge = (status: string) => {
  if (status === 'sent') return 'bg-emerald-50 text-emerald-700 dark:bg-emerald-500/15 dark:text-emerald-400'
  if (status === 'failed') return 'bg-rose-50 text-rose-700 dark:bg-rose-500/15 dark:text-rose-400'
  return 'bg-amber-50 text-amber-700 dark:bg-amber-500/15 dark:text-amber-400'
}
const getStatusDot = (status: string) => {
  if (status === 'sent') return 'bg-emerald-500'
  if (status === 'failed') return 'bg-rose-500'
  return 'bg-amber-500'
}
const statusLabel = (status: string) => {
  if (status === 'sent') return '已发送'
  if (status === 'failed') return '失败'
  return status
}

onMounted(load)
</script>

<template>
  <div class="p-6 lg:p-8 max-w-7xl mx-auto">
    <!-- Header -->
    <div class="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4 mb-6">
      <div>
        <h1 class="text-2xl font-bold tracking-tight">通知中心</h1>
        <p class="mt-1 text-sm text-muted">查看版本发布通知记录</p>
      </div>
    </div>

    <!-- Filters -->
    <div class="flex flex-wrap items-center gap-3 mb-5">
      <div class="relative flex-1 min-w-[200px] max-w-xs">
        <svg class="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-muted" fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"/>
        </svg>
        <input
          v-model="search" @input="load"
          placeholder="搜索仓库名..."
          class="input-focus w-full pl-9 pr-4 py-2 text-sm rounded-xl border border-border/60 bg-surface placeholder:text-muted/60"
        />
      </div>
      <select
        v-model="statusFilter" @change="load"
        class="input-focus px-3 py-2 text-sm rounded-xl border border-border/60 bg-surface text-muted"
      >
        <option value="">所有状态</option>
        <option value="sent">已发送</option>
        <option value="failed">失败</option>
      </select>
      <span class="text-xs text-muted ml-auto">
        共 {{ notifications.length }} 条记录
      </span>
    </div>

    <!-- Content -->
    <div class="rounded-2xl border border-border/60 overflow-hidden">
      <!-- Loading -->
      <div v-if="loading" class="p-6 space-y-3">
        <div v-for="i in 8" :key="i" class="h-14 rounded-xl skeleton" />
      </div>

      <!-- Empty -->
      <div v-else-if="notifications.length === 0" class="px-6 py-20 text-center">
        <div class="w-16 h-16 mx-auto rounded-2xl bg-gradient-to-br from-indigo-50 to-violet-50 dark:from-indigo-500/10 dark:to-violet-500/10 flex items-center justify-center mb-4">
          <svg class="w-8 h-8 text-indigo-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 17h5l-1.405-1.405A2.032 2.032 0 0118 14.158V11a6 6 0 10-12 0v3.159c0 .538-.214 1.055-.595 1.436L4 17h5m6 0v1a3 3 0 11-6 0v-1m6 0H9"/>
          </svg>
        </div>
        <h3 class="text-sm font-medium mb-1">暂无通知</h3>
        <p class="text-xs text-muted">当监控仓库有新版本发布时将出现在这里</p>
      </div>

      <!-- List -->
      <div v-else class="divide-y divide-border/40">
        <div v-for="n in notifications" :key="n.id" class="notification-item">
          <!-- Row -->
          <div
            @click="toggleExpand(n.id)"
            class="flex items-center gap-3 px-5 py-4 cursor-pointer hover:bg-surface-alt/50 transition-colors"
          >
            <span :class="['w-2 h-2 rounded-full shrink-0', getStatusDot(n.status)]" />
            <div class="min-w-0 flex-1">
              <div class="flex items-center gap-2">
                <span class="text-sm font-medium truncate">{{ n.full_name }}</span>
                <span v-if="n.tag" class="text-xs bg-indigo-50 text-indigo-600 dark:bg-indigo-500/15 dark:text-indigo-400 px-2 py-0.5 rounded-full font-medium">
                  {{ n.tag }}
                </span>
              </div>
              <p class="text-xs text-muted mt-0.5 flex items-center gap-3">
                <span>发布: {{ formatTime(n.released_at) }}</span>
                <span v-if="n.sent_at" class="hidden sm:inline">通知: {{ formatTime(n.sent_at) }}</span>
              </p>
            </div>
            <span :class="['text-xs px-2.5 py-0.5 rounded-full font-medium shrink-0 hidden sm:inline-block', getStatusBadge(n.status)]">
              {{ statusLabel(n.status) }}
            </span>
            <svg
              :class="['w-4 h-4 text-muted shrink-0 transition-transform duration-200', expanded.has(n.id) && 'rotate-180']"
              fill="none" viewBox="0 0 24 24" stroke="currentColor"
            >
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7"/>
            </svg>
          </div>

          <!-- Expanded -->
          <div
            v-if="expanded.has(n.id)"
            class="px-5 pb-5 animate-slide-down"
          >
            <div class="rounded-xl bg-surface-alt/60 border border-border/40 p-4">
              <!-- Error -->
              <div v-if="n.error" class="mb-4 p-3 rounded-lg bg-rose-50 dark:bg-rose-500/10 border border-rose-200/50 dark:border-rose-500/20">
                <p class="text-xs font-medium text-rose-600 dark:text-rose-400 mb-1">错误信息</p>
                <p class="text-sm text-rose-700 dark:text-rose-300">{{ n.error }}</p>
              </div>

              <!-- Body -->
              <div class="flex items-center justify-between mb-2">
                <p class="text-xs font-semibold text-muted uppercase tracking-wider">Release Notes</p>
                <button
                  v-if="n.release_body"
                  @click="doTranslate(n)"
                  :disabled="translating.has(n.id) || !!translateCache.get(n.id)"
                  class="text-xs font-medium text-indigo-500 hover:text-indigo-600 dark:text-indigo-400 dark:hover:text-indigo-300 disabled:opacity-50 transition-colors"
                >
                  {{ translating.has(n.id) ? '翻译中...' : translateCache.get(n.id) ? '已翻译' : '翻译为中文' }}
                </button>
              </div>

              <!-- Translated (shown first) -->
              <div v-if="translateCache.get(n.id) || n.release_body_translated" class="mb-3 p-3 rounded-lg bg-indigo-50/50 dark:bg-indigo-500/5 border border-indigo-200/50 dark:border-indigo-500/15">
                <p class="text-xs font-semibold text-indigo-500 dark:text-indigo-400 mb-1.5">中文翻译</p>
                <div class="text-sm whitespace-pre-wrap break-words">
                  {{ translateCache.get(n.id) || n.release_body_translated }}
                </div>
              </div>

              <!-- Original (shown below translation) -->
              <div v-if="n.release_body" class="text-sm prose prose-sm max-w-none dark:prose-invert whitespace-pre-wrap break-words text-muted/90" :class="{ 'mt-0': translateCache.get(n.id) || n.release_body_translated }">
                {{ formatBody(n.release_body).replace(/<[^>]+>/g, '') }}
              </div>

              <!-- No body -->
              <p v-if="!n.release_body" class="text-sm text-muted italic">无 Release Notes</p>

              <!-- Actions -->
              <div class="flex items-center gap-3 mt-4 pt-3 border-t border-border/30">
                <a
                  v-if="n.release_url"
                  :href="n.release_url"
                  target="_blank"
                  class="inline-flex items-center gap-1.5 text-xs font-medium text-indigo-500 hover:text-indigo-600 dark:text-indigo-400 dark:hover:text-indigo-300 transition-colors"
                >
                  <svg class="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14"/>
                  </svg>
                  查看 Release
                </a>
                <button
                  v-if="n.status === 'failed'"
                  @click="doRetry(n.id)"
                  class="inline-flex items-center gap-1.5 text-xs font-medium text-amber-600 dark:text-amber-400 hover:text-amber-700 dark:hover:text-amber-300 transition-colors"
                >
                  <svg class="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"/>
                  </svg>
                  重试发送
                </button>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
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

.bg-surface-alt\/60 {
  background: color-mix(in srgb, var(--color-surface-alt) 60%, transparent);
}

.border-border\/60 {
  border-color: color-mix(in srgb, var(--color-border) 60%, transparent);
}

.border-border\/50 {
  border-color: color-mix(in srgb, var(--color-border) 50%, transparent);
}

.border-border\/40 {
  border-color: color-mix(in srgb, var(--color-border) 40%, transparent);
}

.border-border\/30 {
  border-color: color-mix(in srgb, var(--color-border) 30%, transparent);
}

.notification-item:last-child {
  border-bottom: none;
}
</style>
