<script setup lang="ts">
import { onMounted, ref, reactive, computed } from 'vue'
import { api } from '../api'
import type { Repo, SyncStatus } from '../types'
import { useToast } from '../components/toast'

const toast = useToast()
const repos = ref<Repo[]>([])
const loading = ref(true)
const syncing = ref(false)
const syncStatus = reactive<SyncStatus>({
  running: false, page: 0, total: 0, progress: 0,
  repos: 0, added: 0, updated: 0, skipped: 0, removed: 0, error: '',
})

const search = ref('')
const langFilter = ref('')
const monitorFilter = ref('')
const selected = ref<Set<number>>(new Set())
const selectAll = computed({
  get: () => repos.value.length > 0 && repos.value.every(r => selected.value.has(r.id)),
  set: (v: boolean) => {
    if (v) repos.value.forEach(r => selected.value.add(r.id))
    else selected.value.clear()
  },
})

// Add repo
const showAdd = ref(false)
const newRepoName = ref('')
const adding = ref(false)

// Edit ignore
const showEdit = ref(false)
const editingRepo = ref<Repo | null>(null)
const editIgnorePattern = ref('')
const editing = ref(false)

let pollTimer: ReturnType<typeof setInterval> | null = null

async function load() {
  loading.value = true
  try {
    const params: any = {}
    if (search.value) params.query = search.value
    if (langFilter.value) params.language = langFilter.value
    if (monitorFilter.value !== '') params.monitored = monitorFilter.value
    repos.value = await api.listRepos(params)
  } catch (e: any) {
    toast.error(e.message || '加载失败')
  } finally {
    loading.value = false
  }
}

async function doSync() {
  syncing.value = true
  syncStatus.running = true
  syncStatus.progress = 0
  syncStatus.error = ''
  try {
    await api.syncStars()
    pollTimer = setInterval(pollSync, 1500)
  } catch (e: any) {
    toast.error(e.message || '同步失败')
    syncing.value = false
    syncStatus.running = false
  }
}

async function pollSync() {
  try {
    const s = await api.syncStarsStatus()
    Object.assign(syncStatus, s)
    syncStatus.progress = s.total > 0 ? Math.min(100, Math.round((s.page / s.total) * 100)) : 0
    if (!s.running) {
      stopPoll()
      await load()
      if (s.error) toast.error(s.error)
      else toast.success('同步完成')
    }
  } catch {
    stopPoll()
    toast.error('同步状态查询失败')
  }
}

function stopPoll() {
  if (pollTimer) { clearInterval(pollTimer); pollTimer = null }
  syncing.value = false
}

async function toggleMonitor(repo: Repo) {
  try {
    await api.setMonitored(repo.id, !repo.monitored)
    repo.monitored = !repo.monitored
    toast.success(repo.monitored ? '已开启监控' : '已关闭监控')
  } catch (e: any) {
    toast.error(e.message)
  }
}

async function batchMonitor(monitored: boolean) {
  const ids = Array.from(selected.value)
  if (ids.length === 0) return
  try {
    await api.batchMonitor(ids, monitored)
    await load()
    selected.value.clear()
    toast.success(`已${monitored ? '开启' : '关闭'} ${ids.length} 个仓库`)
  } catch (e: any) {
    toast.error(e.message)
  }
}

async function deleteRepo(id: number) {
  if (!confirm('确定删除该仓库吗？')) return
  try {
    await api.deleteRepo(id)
    await load()
    toast.success('已删除')
  } catch (e: any) {
    toast.error(e.message)
  }
}

function openEdit(repo: Repo) {
  editingRepo.value = repo
  editIgnorePattern.value = repo.ignore_pattern || ''
  showEdit.value = true
}

async function saveEdit() {
  if (!editingRepo.value) return
  editing.value = true
  try {
    await api.setMonitored(editingRepo.value.id, editingRepo.value.monitored, editIgnorePattern.value)
    editingRepo.value.ignore_pattern = editIgnorePattern.value
    showEdit.value = false
    toast.success('已更新')
  } catch (e: any) {
    toast.error(e.message)
  } finally {
    editing.value = false
  }
}

async function addRepo() {
  const name = newRepoName.value.trim()
  if (!name) return
  adding.value = true
  try {
    await api.addRepo(name)
    newRepoName.value = ''
    showAdd.value = false
    await load()
    toast.success('已添加')
  } catch (e: any) {
    toast.error(e.message)
  } finally {
    adding.value = false
  }
}

function formatTime(t: string) {
  if (!t) return '-'
  return new Date(t).toLocaleString('zh-CN', {
    month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit',
  })
}

const allLangs = computed(() => {
  const set = new Set(repos.value.map(r => r.language).filter(Boolean))
  return Array.from(set).sort()
})

onMounted(load)
</script>

<template>
  <div class="p-6 lg:p-8 max-w-7xl mx-auto">
    <!-- Header -->
    <div class="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4 mb-6">
      <div>
        <h1 class="text-2xl font-bold tracking-tight">仓库管理</h1>
        <p class="mt-1 text-sm text-muted">管理监控的 GitHub 仓库</p>
      </div>
      <div class="flex items-center gap-2">
        <button
          @click="doSync" :disabled="syncing"
          class="inline-flex items-center gap-2 px-4 py-2 text-sm font-medium rounded-xl border border-border/60 hover:bg-surface-alt transition-all duration-200 disabled:opacity-50"
        >
          <svg :class="['w-4 h-4', syncing && 'animate-spin']" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"/>
          </svg>
          {{ syncing ? `同步中 ${syncStatus.progress}%` : '同步 Stars' }}
        </button>
        <button
          @click="showAdd = true"
          class="inline-flex items-center gap-2 px-4 py-2 text-sm font-medium rounded-xl bg-gradient-to-r from-indigo-500 to-violet-500 text-white hover:from-indigo-600 hover:to-violet-600 transition-all duration-200 shadow-md shadow-indigo-500/20"
        >
          <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4"/>
          </svg>
          添加仓库
        </button>
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
          placeholder="搜索仓库名称..."
          class="input-focus w-full pl-9 pr-4 py-2 text-sm rounded-xl border border-border/60 bg-surface placeholder:text-muted/60"
        />
      </div>
      <select
        v-model="langFilter" @change="load"
        class="input-focus px-3 py-2 text-sm rounded-xl border border-border/60 bg-surface text-muted"
      >
        <option value="">所有语言</option>
        <option v-for="l in allLangs" :key="l" :value="l">{{ l }}</option>
      </select>
      <select
        v-model="monitorFilter" @change="load"
        class="input-focus px-3 py-2 text-sm rounded-xl border border-border/60 bg-surface text-muted"
      >
        <option value="">所有状态</option>
        <option value="true">监控中</option>
        <option value="false">未监控</option>
      </select>

      <!-- Batch actions -->
      <div v-if="selected.size > 0" class="flex items-center gap-2 ml-auto">
        <span class="text-xs text-muted">已选 {{ selected.size }}</span>
        <button @click="batchMonitor(true)" class="px-3 py-1.5 text-xs font-medium rounded-lg bg-emerald-50 text-emerald-700 dark:bg-emerald-500/15 dark:text-emerald-400 hover:bg-emerald-100 dark:hover:bg-emerald-500/25 transition-colors">
          批量监控
        </button>
        <button @click="batchMonitor(false)" class="px-3 py-1.5 text-xs font-medium rounded-lg bg-surface-alt text-muted hover:text-foreground transition-colors">
          取消监控
        </button>
      </div>
    </div>

    <!-- Table -->
    <div class="rounded-2xl border border-border/60 overflow-hidden">
      <!-- Loading skeleton -->
      <div v-if="loading" class="p-6 space-y-3">
        <div v-for="i in 8" :key="i" class="h-10 rounded-xl skeleton" />
      </div>

      <!-- Empty -->
      <div v-else-if="repos.length === 0" class="px-6 py-20 text-center">
        <div class="w-16 h-16 mx-auto rounded-2xl bg-gradient-to-br from-indigo-50 to-violet-50 dark:from-indigo-500/10 dark:to-violet-500/10 flex items-center justify-center mb-4">
          <svg class="w-8 h-8 text-indigo-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M7 21a4 4 0 01-4-4V5a2 2 0 012-2h4a2 2 0 012 2v12a4 4 0 01-4 4zm0 0h12a2 2 0 002-2v-4a2 2 0 00-2-2h-2.343M11 7.343l1.657-1.657a2 2 0 012.828 0l2.829 2.829a2 2 0 010 2.828l-8.486 8.485M7 17h.01"/>
          </svg>
        </div>
        <h3 class="text-sm font-medium mb-1">暂无仓库</h3>
        <p class="text-xs text-muted mb-4">点击 "添加仓库" 或 "同步 Stars" 开始</p>
        <button @click="showAdd = true" class="inline-flex items-center gap-2 px-4 py-2 text-sm font-medium rounded-xl bg-gradient-to-r from-indigo-500 to-violet-500 text-white">
          添加仓库
        </button>
      </div>

      <!-- Repo Table -->
      <table v-else class="w-full">
        <thead>
          <tr class="border-b border-border/60 bg-surface-alt/50">
            <th class="w-10 px-5 py-3">
              <input
                type="checkbox"
                :checked="selectAll"
                @change="selectAll = ($event.target as HTMLInputElement).checked"
                class="rounded-md border-border/60 text-indigo-500 focus:ring-indigo-500"
              />
            </th>
            <th class="px-3 py-3 text-left text-xs font-semibold text-muted uppercase tracking-wider">仓库</th>
            <th class="px-3 py-3 text-left text-xs font-semibold text-muted uppercase tracking-wider hidden md:table-cell">语言</th>
            <th class="px-3 py-3 text-left text-xs font-semibold text-muted uppercase tracking-wider hidden sm:table-cell">Stars</th>
            <th class="px-3 py-3 text-left text-xs font-semibold text-muted uppercase tracking-wider hidden lg:table-cell">最近检查</th>
            <th class="px-3 py-3 text-left text-xs font-semibold text-muted uppercase tracking-wider">监控</th>
            <th class="px-3 py-3 text-left text-xs font-semibold text-muted uppercase tracking-wider w-20">操作</th>
          </tr>
        </thead>
        <tbody class="divide-y divide-border/40">
          <tr
            v-for="repo in repos" :key="repo.id"
            class="hover:bg-surface-alt/50 transition-colors"
          >
            <td class="px-5 py-3.5">
              <input
                type="checkbox"
                :checked="selected.has(repo.id)"
                @change="selected.has(repo.id) ? selected.delete(repo.id) : selected.add(repo.id)"
                class="rounded-md border-border/60 text-indigo-500 focus:ring-indigo-500"
              />
            </td>
            <td class="px-3 py-3.5">
              <div class="flex flex-col">
                <a :href="repo.html_url" target="_blank" class="text-sm font-medium hover:text-indigo-500 transition-colors truncate max-w-[240px]">
                  {{ repo.full_name }}
                </a>
                <p v-if="repo.description" class="text-xs text-muted mt-0.5 truncate max-w-[240px]">
                  {{ repo.description }}
                </p>
              </div>
            </td>
            <td class="px-3 py-3.5 hidden md:table-cell">
              <span v-if="repo.language" class="inline-flex items-center gap-1.5 text-xs text-muted">
                <span class="w-2 h-2 rounded-full bg-indigo-400" />
                {{ repo.language }}
              </span>
              <span v-else class="text-xs text-muted/50">-</span>
            </td>
            <td class="px-3 py-3.5 hidden sm:table-cell">
              <span class="inline-flex items-center gap-1 text-xs text-muted">
                <svg class="w-3.5 h-3.5 text-amber-400" fill="currentColor" viewBox="0 0 20 20">
                  <path d="M9.049 2.927c.3-.921 1.603-.921 1.902 0l1.07 3.292a1 1 0 00.95.69h3.462c.969 0 1.371 1.24.588 1.81l-2.8 2.034a1 1 0 00-.364 1.118l1.07 3.292c.3.921-.755 1.688-1.54 1.118l-2.8-2.034a1 1 0 00-1.175 0l-2.8 2.034c-.784.57-1.838-.197-1.539-1.118l1.07-3.292a1 1 0 00-.364-1.118L2.98 8.72c-.783-.57-.38-1.81.588-1.81h3.461a1 1 0 00.951-.69l1.07-3.292z"/>
                </svg>
                {{ repo.stargazers_count }}
              </span>
            </td>
            <td class="px-3 py-3.5 hidden lg:table-cell">
              <span class="text-xs text-muted">{{ formatTime(repo.last_checked_at) }}</span>
            </td>
            <td class="px-3 py-3.5">
              <button
                @click="toggleMonitor(repo)"
                :class="[
                  'inline-flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-medium transition-all duration-200',
                  repo.monitored
                    ? 'bg-emerald-50 text-emerald-700 dark:bg-emerald-500/15 dark:text-emerald-400 hover:bg-emerald-100 dark:hover:bg-emerald-500/25'
                    : 'bg-surface-alt text-muted hover:text-foreground'
                ]"
              >
                <span :class="['w-1.5 h-1.5 rounded-full', repo.monitored ? 'bg-emerald-500' : 'bg-muted/40']" />
                {{ repo.monitored ? '监控中' : '未监控' }}
              </button>
            </td>
            <td class="px-3 py-3.5">
              <div class="flex items-center gap-1">
                <button
                  @click="openEdit(repo)"
                  class="p-1.5 rounded-lg text-muted hover:text-foreground hover:bg-surface-alt transition-all"
                  title="编辑忽略规则"
                >
                  <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z"/>
                  </svg>
                </button>
                <button
                  @click="deleteRepo(repo.id)"
                  class="p-1.5 rounded-lg text-muted hover:text-amber-500 hover:bg-amber-50 dark:hover:bg-amber-500/10 transition-all"
                  title="取消星标"
                >
                  <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M11.049 2.927c.3-.921 1.603-.921 1.902 0l1.519 4.674a1 1 0 00.95.69h4.915c.969 0 1.371 1.24.588 1.81l-3.976 2.888a1 1 0 00-.363 1.118l1.518 4.674c.3.922-.755 1.688-1.538 1.118l-3.976-2.888a1 1 0 00-1.176 0l-3.976 2.888c-.783.57-1.838-.197-1.538-1.118l1.518-4.674a1 1 0 00-.363-1.118l-3.976-2.888c-.784-.57-.38-1.81.588-1.81h4.914a1 1 0 00.951-.69l1.519-4.674z"/>
                    <line x1="3" y1="3" x2="21" y2="21" stroke="currentColor" stroke-width="2" stroke-linecap="round"/>
                  </svg>
                </button>
              </div>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- Add Repo Modal -->
    <Teleport to="body">
      <div v-if="showAdd" class="fixed inset-0 z-50 flex items-center justify-center">
        <div class="absolute inset-0 bg-black/40 backdrop-blur-sm" @click="showAdd = false" />
        <div class="relative w-full max-w-md mx-4 bg-surface rounded-2xl shadow-elevated border border-border/60 animate-scale-in">
          <div class="flex items-center justify-between px-5 py-4 border-b border-border/50">
            <h3 class="font-semibold">添加仓库</h3>
            <button @click="showAdd = false" class="p-1 rounded-lg text-muted hover:text-foreground hover:bg-surface-alt transition-colors">
              <svg class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"/>
              </svg>
            </button>
          </div>
          <div class="p-5">
            <label class="block text-sm font-medium mb-2">仓库全名</label>
            <input
              v-model="newRepoName"
              @keyup.enter="addRepo"
              placeholder="owner/repo"
              class="input-focus w-full px-4 py-2.5 text-sm rounded-xl border border-border/60 bg-surface placeholder:text-muted/50"
            />
          </div>
          <div class="flex justify-end gap-3 px-5 pb-5">
            <button @click="showAdd = false" class="px-4 py-2 text-sm rounded-xl border border-border/60 text-muted hover:text-foreground hover:bg-surface-alt transition-all">
              取消
            </button>
            <button @click="addRepo" :disabled="adding || !newRepoName.trim()" class="px-4 py-2 text-sm font-medium rounded-xl bg-gradient-to-r from-indigo-500 to-violet-500 text-white hover:from-indigo-600 hover:to-violet-600 disabled:opacity-50 transition-all">
              {{ adding ? '添加中...' : '确定添加' }}
            </button>
          </div>
        </div>
      </div>

      <!-- Edit Modal -->
      <div v-if="showEdit" class="fixed inset-0 z-50 flex items-center justify-center">
        <div class="absolute inset-0 bg-black/40 backdrop-blur-sm" @click="showEdit = false" />
        <div class="relative w-full max-w-md mx-4 bg-surface rounded-2xl shadow-elevated border border-border/60 animate-scale-in">
          <div class="flex items-center justify-between px-5 py-4 border-b border-border/50">
            <h3 class="font-semibold">编辑: {{ editingRepo?.full_name }}</h3>
            <button @click="showEdit = false" class="p-1 rounded-lg text-muted hover:text-foreground hover:bg-surface-alt transition-colors">
              <svg class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"/>
              </svg>
            </button>
          </div>
          <div class="p-5">
            <label class="block text-sm font-medium mb-2">忽略规则 (支持正则)</label>
            <input
              v-model="editIgnorePattern"
              placeholder="例如: v0\..*, -alpha$"
              class="input-focus w-full px-4 py-2.5 text-sm rounded-xl border border-border/60 bg-surface placeholder:text-muted/50"
            />
            <p class="mt-1.5 text-xs text-muted">匹配此规则的 tag 不会发送通知</p>
          </div>
          <div class="flex justify-end gap-3 px-5 pb-5">
            <button @click="showEdit = false" class="px-4 py-2 text-sm rounded-xl border border-border/60 text-muted hover:text-foreground hover:bg-surface-alt transition-all">
              取消
            </button>
            <button @click="saveEdit" :disabled="editing" class="px-4 py-2 text-sm font-medium rounded-xl bg-gradient-to-r from-indigo-500 to-violet-500 text-white hover:from-indigo-600 hover:to-violet-600 disabled:opacity-50 transition-all">
              {{ editing ? '保存中...' : '保存' }}
            </button>
          </div>
        </div>
      </div>
    </Teleport>
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

.border-border\/40 {
  border-color: color-mix(in srgb, var(--color-border) 40%, transparent);
}

.shadow-elevated {
  box-shadow: 0 10px 40px rgba(0, 0, 0, 0.12), 0 4px 12px rgba(0, 0, 0, 0.06);
}

:root.dark .shadow-elevated {
  box-shadow: 0 10px 40px rgba(0, 0, 0, 0.3), 0 4px 12px rgba(0, 0, 0, 0.2);
}
</style>
