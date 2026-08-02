<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { api } from '../api'
import { useToast } from '../components/toast'
import ToggleSwitch from '../components/ToggleSwitch.vue'
import type { Repo, SyncStatus } from '../types'

const toast = useToast()
const loading = ref(true)
const syncing = ref(false)
const syncStatus = ref<SyncStatus | null>(null)
const repos = ref<Repo[]>([])
const search = ref('')
const langFilter = ref('')
const monFilter = ref('')

const showAdd = ref(false)
const addName = ref('')
const adding = ref(false)

const selected = ref<Set<number>>(new Set())
let pollTimer: number | undefined

const languages = computed(() => {
  const set = new Set<string>()
  for (const r of repos.value) if (r.language) set.add(r.language)
  return [...set].sort()
})

const filtered = computed(() => {
  return repos.value.filter((r) => {
    if (search.value && !r.full_name.toLowerCase().includes(search.value.toLowerCase())) return false
    if (langFilter.value && r.language !== langFilter.value) return false
    if (monFilter.value === 'monitored' && !r.monitored) return false
    if (monFilter.value === 'unmonitored' && r.monitored) return false
    return true
  })
})

const progressPct = computed(() => Math.round((syncStatus.value?.progress ?? 0) * 100))

const selectedCount = computed(() => selected.value.size)

const allChecked = computed(
  () => filtered.value.length > 0 && filtered.value.every((r) => selected.value.has(r.id)),
)

async function load() {
  loading.value = true
  try {
    repos.value = await api.listRepos()
  } catch (e) {
    toast.error((e as Error).message)
  } finally {
    loading.value = false
  }
}

function pollSync() {
  window.clearTimeout(pollTimer)
  const tick = async () => {
    try {
      const st = await api.syncStarsStatus()
      syncStatus.value = st
      syncing.value = st.running
      if (st.running) {
        pollTimer = window.setTimeout(tick, 600)
      } else {
        if (st.error) {
          toast.error('同步失败：' + st.error)
        } else {
          toast.success(`同步完成：已处理 ${st.repos} 个仓库，新增 ${st.added} 个`)
        }
        await load()
      }
    } catch (e) {
      toast.error((e as Error).message)
      syncing.value = false
    }
  }
  pollTimer = window.setTimeout(tick, 300)
}

async function syncStars() {
  try {
    await api.syncStars()
    syncStatus.value = { running: true, page: 0, total: 0, progress: 0, repos: 0, added: 0, error: '' }
    syncing.value = true
    pollSync()
  } catch (e) {
    toast.error((e as Error).message)
  }
}

async function toggle(r: Repo, v: boolean) {
  const prev = r.monitored
  r.monitored = v
  try {
    await api.setMonitored(r.id, v)
    toast.success(v ? `已加入监控：${r.full_name}` : `已取消监控：${r.full_name}`)
  } catch (e) {
    r.monitored = prev
    toast.error((e as Error).message)
  }
}

function toggleSelect(id: number) {
  const s = new Set(selected.value)
  if (s.has(id)) s.delete(id)
  else s.add(id)
  selected.value = s
}

function toggleAll(e: Event) {
  const checked = (e.target as HTMLInputElement).checked
  const s = new Set<number>()
  if (checked) filtered.value.forEach((r) => s.add(r.id))
  selected.value = s
}

async function batchSetMonitored(monitored: boolean) {
  const ids = [...selected.value]
  if (ids.length === 0) return
  try {
    const res = await api.batchMonitor(ids, monitored)
    for (const r of repos.value) {
      if (selected.value.has(r.id)) r.monitored = monitored
    }
    selected.value = new Set()
    toast.success(
      res.monitored ? `已将 ${res.updated} 个仓库加入监控` : `已取消 ${res.updated} 个仓库的监控`,
    )
  } catch (e) {
    toast.error((e as Error).message)
  }
}

async function remove(r: Repo) {
  if (!window.confirm(`确定删除仓库 ${r.full_name} 吗？`)) return
  try {
    await api.deleteRepo(r.id)
    repos.value = repos.value.filter((x) => x.id !== r.id)
    toast.success(`已删除 ${r.full_name}`)
  } catch (e) {
    toast.error((e as Error).message)
  }
}

async function submitAdd() {
  if (!addName.value.trim()) return
  adding.value = true
  try {
    await api.addRepo(addName.value.trim())
    toast.success('添加成功')
    showAdd.value = false
    addName.value = ''
    await load()
  } catch (e) {
    toast.error((e as Error).message)
  } finally {
    adding.value = false
  }
}

onMounted(async () => {
  await load()
  try {
    const st = await api.syncStarsStatus()
    if (st.running) {
      syncStatus.value = st
      syncing.value = true
      pollSync()
    }
  } catch {
    /* ignore */
  }
})

onUnmounted(() => window.clearTimeout(pollTimer))
</script>

<template>
  <div class="mx-auto max-w-6xl p-4 md:p-8">
    <div class="mb-6">
      <h1 class="text-2xl font-bold text-slate-900">仓库</h1>
      <p class="mt-1 text-sm text-slate-500">从 GitHub 同步你 Star 的仓库，并选择要监控的版本发布</p>
    </div>

    <div v-if="syncing" class="mb-4 overflow-hidden rounded-2xl border border-sky-200 bg-sky-50 p-4">
      <div class="flex items-center justify-between text-sm">
        <span class="flex items-center gap-2 font-medium text-sky-800">
          <svg class="h-4 w-4 animate-spin" viewBox="0 0 24 24" fill="none">
            <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" />
            <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 0 1 8-8V0C5.373 0 0 5.373 0 12h4z" />
          </svg>
          正在同步 Star 仓库...
        </span>
        <span class="font-semibold text-sky-700">{{ progressPct }}%</span>
      </div>
      <div class="mt-3 h-2 w-full overflow-hidden rounded-full bg-sky-100">
        <div class="h-full rounded-full bg-sky-500 transition-all duration-300" :style="{ width: progressPct + '%' }" />
      </div>
      <div class="mt-1.5 text-xs text-sky-600">
        已处理 {{ syncStatus?.repos ?? 0 }} 个仓库 · 新增 {{ syncStatus?.added ?? 0 }} 个
      </div>
    </div>

    <div class="mb-4 flex flex-wrap items-center gap-2">
      <div class="relative">
        <svg class="pointer-events-none absolute left-3 top-2.5 h-4 w-4 text-slate-400" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path stroke-linecap="round" stroke-linejoin="round" d="M21 21l-5.197-5.197m0 0A7.5 7.5 0 1 0 5.196 5.196a7.5 7.5 0 0 0 10.607 10.607z" />
        </svg>
        <input
          v-model="search"
          type="text"
          placeholder="搜索仓库..."
          class="w-56 rounded-xl border border-slate-300 bg-white py-2 pl-9 pr-3 text-sm outline-none transition focus:border-sky-500 focus:ring-2 focus:ring-sky-100"
        />
      </div>
      <select
        v-model="langFilter"
        class="rounded-xl border border-slate-300 bg-white px-3 py-2 text-sm outline-none transition focus:border-sky-500"
      >
        <option value="">全部语言</option>
        <option v-for="l in languages" :key="l" :value="l">{{ l }}</option>
      </select>
      <select
        v-model="monFilter"
        class="rounded-xl border border-slate-300 bg-white px-3 py-2 text-sm outline-none transition focus:border-sky-500"
      >
        <option value="">全部状态</option>
        <option value="monitored">监控中</option>
        <option value="unmonitored">未监控</option>
      </select>
      <div class="flex-1" />
      <button
        class="inline-flex items-center gap-2 rounded-xl bg-sky-600 px-4 py-2 text-sm font-medium text-white shadow-sm transition hover:bg-sky-500 disabled:opacity-50"
        :disabled="syncing"
        @click="syncStars"
      >
        <svg v-if="syncing" class="h-4 w-4 animate-spin" viewBox="0 0 24 24" fill="none">
          <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" />
          <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 0 1 8-8V0C5.373 0 0 5.373 0 12h4z" />
        </svg>
        <svg v-else class="h-4 w-4" viewBox="0 0 24 24" fill="currentColor">
          <path fill-rule="evenodd" d="M10.788 3.21c.448-1.077 1.976-1.077 2.424 0l2.082 5.007 5.404.433c1.164.093 1.636 1.545.749 2.305l-4.117 3.527 1.257 5.273c.271 1.136-.964 2.033-1.96 1.425L12 18.354 7.373 21.18c-.996.608-2.231-.29-1.96-1.425l1.257-5.273-4.117-3.527c-.887-.76-.415-2.212.749-2.305l5.404-.433 2.082-5.006z" clip-rule="evenodd" />
        </svg>
        同步我的 Star
      </button>
      <button
        class="inline-flex items-center gap-2 rounded-xl bg-slate-900 px-4 py-2 text-sm font-medium text-white shadow-sm transition hover:bg-slate-700"
        @click="showAdd = true"
      >
        <svg class="h-4 w-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path stroke-linecap="round" stroke-linejoin="round" d="M12 4.5v15m7.5-7.5h-15" />
        </svg>
        手动添加
      </button>
    </div>

    <div v-if="selectedCount > 0" class="mb-3 flex flex-wrap items-center gap-2 rounded-xl border border-slate-200 bg-white px-4 py-2.5 shadow-sm">
      <span class="text-sm font-medium text-slate-700">已选 {{ selectedCount }} 个仓库</span>
      <div class="flex-1" />
      <button
        class="inline-flex items-center gap-1.5 rounded-lg bg-emerald-600 px-3 py-1.5 text-xs font-medium text-white transition hover:bg-emerald-500"
        @click="batchSetMonitored(true)"
      >
        <svg class="h-3.5 w-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path stroke-linecap="round" stroke-linejoin="round" d="M12 4.5v15m7.5-7.5h-15" />
        </svg>
        加入监控
      </button>
      <button
        class="inline-flex items-center gap-1.5 rounded-lg border border-slate-300 px-3 py-1.5 text-xs font-medium text-slate-600 transition hover:bg-slate-50"
        @click="batchSetMonitored(false)"
      >
        取消监控
      </button>
      <button
        class="rounded-lg px-2.5 py-1.5 text-xs font-medium text-slate-400 transition hover:text-slate-600"
        @click="selected = new Set()"
      >
        清空
      </button>
    </div>

    <div class="overflow-hidden rounded-2xl border border-slate-200 bg-white shadow-sm">
      <div class="overflow-x-auto">
        <table class="w-full text-left text-sm">
          <thead>
            <tr class="border-b border-slate-100 text-xs text-slate-400">
              <th class="px-3 py-3">
                <input
                  type="checkbox"
                  :checked="allChecked"
                  :disabled="filtered.length === 0"
                  class="h-4 w-4 cursor-pointer accent-sky-600"
                  @change="toggleAll"
                />
              </th>
              <th class="px-4 py-3 font-medium">仓库</th>
              <th class="hidden px-4 py-3 font-medium md:table-cell">语言</th>
              <th class="hidden px-4 py-3 font-medium sm:table-cell">Star</th>
              <th class="px-4 py-3 font-medium">最新版本</th>
              <th class="px-4 py-3 text-right font-medium">监控</th>
            </tr>
          </thead>
          <tbody v-if="!loading && filtered.length">
            <tr
              v-for="r in filtered"
              :key="r.id"
              class="border-b border-slate-50 transition hover:bg-slate-50/70"
              :class="selected.has(r.id) ? 'bg-sky-50/60' : ''"
            >
              <td class="px-3 py-3.5">
                <input
                  type="checkbox"
                  :checked="selected.has(r.id)"
                  class="h-4 w-4 cursor-pointer accent-sky-600"
                  @change="toggleSelect(r.id)"
                />
              </td>
              <td class="max-w-xs px-4 py-3.5">
                <a
                  :href="r.html_url || `https://github.com/${r.full_name}`"
                  target="_blank"
                  rel="noopener"
                  class="flex items-center gap-3"
                >
                  <span class="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-slate-100 font-semibold text-slate-600">
                    {{ r.owner[0]?.toUpperCase() }}
                  </span>
                  <span class="min-w-0">
                    <span class="block truncate font-semibold text-slate-900 hover:text-sky-600">{{ r.full_name }}</span>
                    <span v-if="r.description" class="block truncate text-xs text-slate-400">{{ r.description }}</span>
                  </span>
                </a>
              </td>
              <td class="hidden px-4 py-3.5 md:table-cell">
                <span
                  v-if="r.language"
                  class="rounded-full bg-indigo-50 px-2.5 py-0.5 text-xs font-medium text-indigo-700"
                >
                  {{ r.language }}
                </span>
                <span v-else class="text-slate-300">-</span>
              </td>
              <td class="hidden px-4 py-3.5 text-slate-500 sm:table-cell">{{ r.stargazers_count.toLocaleString() }}</td>
              <td class="px-4 py-3.5">
                <span
                  v-if="r.last_known_tag"
                  class="rounded-lg bg-slate-100 px-2.5 py-1 font-mono text-xs font-medium text-slate-700"
                >
                  {{ r.last_known_tag }}
                </span>
                <span v-else class="text-slate-300">-</span>
              </td>
              <td class="px-4 py-3.5">
                <div class="flex items-center justify-end gap-2">
                  <ToggleSwitch :model-value="r.monitored" @update:model-value="(v: boolean) => toggle(r, v)" />
                  <button
                    class="ml-1 rounded-lg p-1.5 text-slate-300 transition hover:bg-rose-50 hover:text-rose-500"
                    title="删除"
                    @click="remove(r)"
                  >
                    <svg class="h-4 w-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8">
                      <path stroke-linecap="round" stroke-linejoin="round" d="M14.74 9l-.346 9m-4.788 0L9.26 9m9.968-3.21c.342.052.682.107 1.022.166m-1.022-.165L18.16 19.673a2.25 2.25 0 0 1-2.244 2.077H8.084a2.25 2.25 0 0 1-2.244-2.077L4.772 5.79m14.456 0a48.108 48.108 0 0 0-3.478-.397m-12 .562c.34-.059.68-.114 1.022-.165m0 0a48.11 48.11 0 0 1 3.478-.397m7.5 0v-.916c0-1.18-.91-2.164-2.09-2.201a51.964 51.964 0 0 0-3.32 0c-1.18.037-2.09 1.022-2.09 2.201v.916m7.5 0a48.667 48.667 0 0 0-7.5 0" />
                    </svg>
                  </button>
                </div>
              </td>
            </tr>
          </tbody>
        </table>
      </div>

      <div v-if="loading" class="space-y-3 p-5">
        <div v-for="i in 5" :key="i" class="h-12 animate-pulse rounded-xl bg-slate-100" />
      </div>
      <div v-else-if="filtered.length === 0" class="p-12 text-center text-sm text-slate-400">
        {{
          repos.length === 0
            ? '暂无仓库。点击「同步我的 Star」或「手动添加」开始。'
            : '没有匹配的仓库。'
        }}
      </div>
    </div>

    <div
      v-if="showAdd"
      class="fixed inset-0 z-40 flex items-center justify-center bg-slate-900/40 p-4"
      @click.self="showAdd = false"
    >
      <div class="w-full max-w-md rounded-2xl bg-white p-6 shadow-xl">
        <h3 class="text-lg font-semibold text-slate-900">手动添加仓库</h3>
        <p class="mt-1 text-sm text-slate-500">输入仓库的 owner/repo 名称，例如 <code class="rounded bg-slate-100 px-1 py-0.5">containrrr/shoutrrr</code></p>
        <form class="mt-4" @submit.prevent="submitAdd">
          <input
            v-model="addName"
            type="text"
            placeholder="owner/repo"
            autofocus
            class="w-full rounded-xl border border-slate-300 px-3.5 py-2.5 text-sm outline-none transition focus:border-sky-500 focus:ring-2 focus:ring-sky-100"
          />
          <div class="mt-5 flex justify-end gap-2">
            <button
              type="button"
              class="rounded-xl px-4 py-2 text-sm font-medium text-slate-500 transition hover:bg-slate-100"
              @click="showAdd = false"
            >
              取消
            </button>
            <button
              type="submit"
              :disabled="adding || !addName.trim()"
              class="rounded-xl bg-sky-600 px-4 py-2 text-sm font-medium text-white transition hover:bg-sky-500 disabled:opacity-50"
            >
              {{ adding ? '添加中...' : '添加' }}
            </button>
          </div>
        </form>
      </div>
    </div>
  </div>
</template>
