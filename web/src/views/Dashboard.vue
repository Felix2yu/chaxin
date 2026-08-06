<script setup lang="ts">
import { onMounted, ref, reactive, computed } from 'vue'
import { api } from '../api'
import type { Notification, Repo } from '../types'
import { useToast } from '../components/toast'

const toast = useToast()
const loading = ref(true)
const repos = ref<Repo[]>([])
const notifications = ref<Notification[]>([])
const stats = reactive({
  total: 0,
  monitored: 0,
  sent: 0,
  failed: 0,
})

const monitoredRepos = computed(() => repos.value.filter((r) => r.monitored))

async function load() {
  loading.value = true
  try {
    const [repoList, notifList] = await Promise.all([
      api.listRepos(),
      api.listNotifications({ limit: 50 }),
    ])
    repos.value = repoList
    notifications.value = notifList
    stats.total = repoList.length
    stats.monitored = repoList.filter((r) => r.monitored).length
    stats.sent = notifList.filter((n) => n.status === 'sent').length
    stats.failed = notifList.filter((n) => n.status === 'failed').length
  } catch (e: any) {
    toast.error(e.message || '加载失败')
  } finally {
    loading.value = false
  }
}

function formatTime(t: string) {
  if (!t) return '-'
  return new Date(t).toLocaleString('zh-CN', {
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
  })
}

function truncate(s: string, n: number) {
  if (!s) return ''
  return s.length > n ? s.slice(0, n) + '...' : s
}

const statCards = computed(() => [
  { key: 'total', label: '仓库总数', value: stats.total, icon: 'folder', color: 'from-indigo-500 to-blue-500', bg: 'bg-indigo-50 dark:bg-indigo-500/10', text: 'text-indigo-600 dark:text-indigo-400' },
  { key: 'monitored', label: '监控中', value: stats.monitored, icon: 'eye', color: 'from-emerald-500 to-teal-500', bg: 'bg-emerald-50 dark:bg-emerald-500/10', text: 'text-emerald-600 dark:text-emerald-400' },
  { key: 'sent', label: '已发送通知', value: stats.sent, icon: 'check', color: 'from-sky-500 to-cyan-500', bg: 'bg-sky-50 dark:bg-sky-500/10', text: 'text-sky-600 dark:text-sky-400' },
  { key: 'failed', label: '发送失败', value: stats.failed, icon: 'alert', color: 'from-rose-500 to-pink-500', bg: 'bg-rose-50 dark:bg-rose-500/10', text: 'text-rose-600 dark:text-rose-400' },
])

const statusBadgeClass = (status: string) => {
  if (status === 'sent') return 'bg-emerald-50 text-emerald-700 dark:bg-emerald-500/15 dark:text-emerald-400'
  return 'bg-rose-50 text-rose-700 dark:bg-rose-500/15 dark:text-rose-400'
}
const statusLabel = (status: string) => status === 'sent' ? '已发送' : '失败'

onMounted(load)
</script>

<template>
  <div class="p-6 lg:p-8 max-w-7xl mx-auto">
    <!-- Page Header -->
    <div class="mb-8">
      <h1 class="text-2xl font-bold tracking-tight">总览</h1>
      <p class="mt-1 text-sm text-muted">监控仓库和通知的概览信息</p>
    </div>

    <!-- Loading skeleton -->
    <template v-if="loading">
      <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-5 mb-8">
        <div v-for="i in 4" :key="i" class="h-28 rounded-2xl skeleton" />
      </div>
      <div class="h-80 rounded-2xl skeleton" />
    </template>

    <template v-else>
      <!-- Stat Cards -->
      <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-5 mb-8">
        <div
          v-for="card in statCards"
          :key="card.key"
          class="stat-card group relative overflow-hidden rounded-2xl border border-border/60 p-5 transition-all duration-300 hover:-translate-y-0.5 hover:shadow-card-hover"
        >
          <div class="flex items-start justify-between">
            <div>
              <p class="text-sm font-medium text-muted">{{ card.label }}</p>
              <p class="mt-2 text-3xl font-bold tracking-tight">{{ card.value }}</p>
            </div>
            <div :class="[card.bg, 'w-10 h-10 rounded-xl flex items-center justify-center']">
              <svg v-if="card.icon === 'folder'" :class="card.text" class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2z"/>
              </svg>
              <svg v-else-if="card.icon === 'eye'" :class="card.text" class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z"/><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z"/>
              </svg>
              <svg v-else-if="card.icon === 'check'" :class="card.text" class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z"/>
              </svg>
              <svg v-else :class="card.text" class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"/>
              </svg>
            </div>
          </div>
          <!-- Gradient bar -->
          <div :class="['absolute bottom-0 left-0 right-0 h-1 bg-gradient-to-r rounded-b-2xl opacity-50', card.color]" />
        </div>
      </div>

      <!-- Two-column layout for lists -->
      <div class="grid grid-cols-1 lg:grid-cols-2 gap-5">
        <!-- Monitored Repos -->
        <div class="rounded-2xl border border-border/60 overflow-hidden">
          <div class="flex items-center justify-between px-5 py-4 border-b border-border/50">
            <h3 class="font-semibold text-sm">监控中的仓库</h3>
            <router-link
              to="/repos"
              class="text-xs font-medium text-indigo-500 hover:text-indigo-600 dark:text-indigo-400 dark:hover:text-indigo-300 transition-colors"
            >
              查看全部 &rarr;
            </router-link>
          </div>
          <div class="divide-y divide-border/40">
            <div v-if="monitoredRepos.length === 0" class="px-5 py-12 text-center">
              <div class="w-12 h-12 mx-auto rounded-full bg-surface-alt flex items-center justify-center mb-3">
                <svg class="w-6 h-6 text-muted" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M20 13V6a2 2 0 00-2-2H6a2 2 0 00-2 2v7m16 0v5a2 2 0 01-2 2H6a2 2 0 01-2-2v-5m16 0h-2.586a1 1 0 00-.707.293l-2.414 2.414a1 1 0 01-.707.293h-3.172a1 1 0 01-.707-.293l-2.414-2.414A1 1 0 006.586 13H4"/>
                </svg>
              </div>
              <p class="text-sm text-muted">暂无监控中的仓库</p>
            </div>
            <a
              v-for="repo in monitoredRepos.slice(0, 5)"
              :key="repo.id"
              :href="repo.html_url"
              target="_blank"
              class="flex items-center justify-between px-5 py-3.5 hover:bg-surface-alt/60 transition-colors group"
            >
              <div class="min-w-0 flex-1">
                <p class="text-sm font-medium truncate group-hover:text-indigo-500 transition-colors">
                  {{ repo.full_name }}
                </p>
                <div class="flex items-center gap-2 mt-0.5">
                  <span class="text-xs text-muted">
                    <span class="inline-flex items-center gap-1">
                      <svg class="w-3 h-3" fill="currentColor" viewBox="0 0 20 20">
                        <path d="M9.049 2.927c.3-.921 1.603-.921 1.902 0l1.07 3.292a1 1 0 00.95.69h3.462c.969 0 1.371 1.24.588 1.81l-2.8 2.034a1 1 0 00-.364 1.118l1.07 3.292c.3.921-.755 1.688-1.54 1.118l-2.8-2.034a1 1 0 00-1.175 0l-2.8 2.034c-.784.57-1.838-.197-1.539-1.118l1.07-3.292a1 1 0 00-.364-1.118L2.98 8.72c-.783-.57-.38-1.81.588-1.81h3.461a1 1 0 00.951-.69l1.07-3.292z"/>
                      </svg>
                      {{ repo.stargazers_count }}
                    </span>
                  </span>
                  <span v-if="repo.language" class="text-xs text-muted">&#8226; {{ repo.language }}</span>
                </div>
              </div>
              <svg class="w-4 h-4 text-muted/40 group-hover:text-indigo-400 transition-colors ml-3 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M14 5l7 7m0 0l-7 7m7-7H3"/>
              </svg>
            </a>
          </div>
        </div>

        <!-- Recent Notifications -->
        <div class="rounded-2xl border border-border/60 overflow-hidden">
          <div class="flex items-center justify-between px-5 py-4 border-b border-border/50">
            <h3 class="font-semibold text-sm">最近通知</h3>
            <router-link
              to="/notifications"
              class="text-xs font-medium text-indigo-500 hover:text-indigo-600 dark:text-indigo-400 dark:hover:text-indigo-300 transition-colors"
            >
              查看全部 &rarr;
            </router-link>
          </div>
          <div class="divide-y divide-border/40">
            <div v-if="notifications.length === 0" class="px-5 py-12 text-center">
              <div class="w-12 h-12 mx-auto rounded-full bg-surface-alt flex items-center justify-center mb-3">
                <svg class="w-6 h-6 text-muted" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 17h5l-1.405-1.405A2.032 2.032 0 0118 14.158V11a6 6 0 10-12 0v3.159c0 .538-.214 1.055-.595 1.436L4 17h5m6 0v1a3 3 0 11-6 0v-1m6 0H9"/>
                </svg>
              </div>
              <p class="text-sm text-muted">暂无通知记录</p>
            </div>
            <div
              v-for="n in notifications.slice(0, 6)"
              :key="n.id"
              class="flex items-center justify-between px-5 py-3.5 hover:bg-surface-alt/60 transition-colors"
            >
              <div class="min-w-0 flex-1">
                <div class="flex items-center gap-2">
                  <span class="text-sm font-medium truncate">{{ n.full_name }}</span>
                  <span v-if="n.tag" class="text-xs text-muted shrink-0">{{ n.tag }}</span>
                </div>
                <p class="text-xs text-muted mt-0.5">{{ formatTime(n.released_at) }}</p>
              </div>
              <span :class="['text-xs px-2.5 py-0.5 rounded-full font-medium shrink-0 ml-3', statusBadgeClass(n.status)]">
                {{ statusLabel(n.status) }}
              </span>
            </div>
          </div>
        </div>
      </div>
    </template>
  </div>
</template>

<style scoped>
.stat-card {
  background: var(--color-surface);
}

.text-muted {
  color: var(--color-text-muted);
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
</style>
