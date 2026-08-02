<script setup lang="ts">
import { RouterLink, RouterView, useRoute } from 'vue-router'
import Toast from './components/Toast.vue'

const route = useRoute()

const navs = [
  {
    to: '/',
    label: '概览',
    icon: 'M3.75 6A2.25 2.25 0 0 1 6 3.75h2.25A2.25 2.25 0 0 1 10.5 6v2.25a2.25 2.25 0 0 1-2.25 2.25H6a2.25 2.25 0 0 1-2.25-2.25V6zM3.75 15.75A2.25 2.25 0 0 1 6 13.5h2.25a2.25 2.25 0 0 1 2.25 2.25V18a2.25 2.25 0 0 1-2.25 2.25H6A2.25 2.25 0 0 1 3.75 18v-2.25zM13.5 6a2.25 2.25 0 0 1 2.25-2.25H18A2.25 2.25 0 0 1 20.25 6v2.25A2.25 2.25 0 0 1 18 10.5h-2.25a2.25 2.25 0 0 1-2.25-2.25V6zM13.5 15.75a2.25 2.25 0 0 1 2.25-2.25H18a2.25 2.25 0 0 1 2.25 2.25V18A2.25 2.25 0 0 1 18 20.25h-2.25A2.25 2.25 0 0 1 13.5 18v-2.25z',
  },
  {
    to: '/repos',
    label: '仓库',
    icon: 'M2.25 12.75V12A2.25 2.25 0 0 1 4.5 9.75h15A2.25 2.25 0 0 1 21.75 12v.75m-8.69-6.44l-2.12-2.12a1.5 1.5 0 0 0-1.061-.44H4.5A2.25 2.25 0 0 0 2.25 6v12a2.25 2.25 0 0 0 2.25 2.25h15A2.25 2.25 0 0 0 21.75 18V9a2.25 2.25 0 0 0-2.25-2.25h-5.379a1.5 1.5 0 0 1-1.06-.44z',
  },
  {
    to: '/notifications',
    label: '通知记录',
    icon: 'M14.857 17.082a23.848 23.848 0 0 0 5.454-1.31A8.967 8.967 0 0 1 18 9.75v-.7V9A6 6 0 0 0 6 9v.75a8.967 8.967 0 0 1-2.312 6.022c1.733.64 3.56 1.085 5.455 1.31m5.714 0a24.255 24.255 0 0 1-5.714 0m5.714 0a3 3 0 1 1-5.714 0M3.124 7.5A8.969 8.969 0 0 1 5.292 3.13 8.969 8.969 0 0 1 18 5.292l.03.004',
  },
  {
    to: '/settings',
    label: '设置',
    icon: 'M9.594 3.94c.09-.542.56-.94 1.11-.94h2.593c.55 0 1.02.398 1.11.94l.213 1.281c.063.374.313.686.645.87.074.04.147.083.22.127.324.196.72.257 1.075.124l1.217-.456a1.125 1.125 0 0 1 1.37.49l1.296 2.247a1.125 1.125 0 0 1-.26 1.431l-1.003.827c-.293.24-.438.613-.431.992a6.759 6.759 0 0 1 0 .255c-.007.378.138.75.43.99l1.005.828c.424.35.534.954.26 1.43l-1.298 2.247a1.125 1.125 0 0 1-1.369.491l-1.217-.456c-.355-.133-.75-.072-1.076.124a6.57 6.57 0 0 1-.22.128c-.331.183-.581.495-.644.869l-.213 1.28c-.09.543-.56.941-1.11.941h-2.594c-.55 0-1.02-.398-1.11-.94l-.213-1.281c-.062-.374-.312-.686-.644-.87a6.52 6.52 0 0 1-.22-.127c-.325-.196-.72-.257-1.076-.124l-1.217.456a1.125 1.125 0 0 1-1.369-.49l-1.297-2.247a1.125 1.125 0 0 1 .26-1.431l1.004-.827c.292-.24.437-.613.43-.992a6.932 6.932 0 0 1 0-.255c.007-.378-.138-.75-.43-.99l-1.004-.828a1.125 1.125 0 0 1-.26-1.43l1.297-2.247a1.125 1.125 0 0 1 1.37-.491l1.216.456c.356.133.751.072 1.076-.124.072-.044.146-.087.22-.128.332-.183.582-.495.644-.869l.214-1.28z',
  },
]
</script>

<template>
  <div class="flex h-full">
    <aside class="hidden w-60 shrink-0 flex-col bg-slate-900 md:flex">
      <div class="flex h-16 items-center gap-2.5 border-b border-slate-800 px-5">
        <span class="flex h-8 w-8 items-center justify-center rounded-lg bg-gradient-to-br from-sky-500 to-indigo-600 text-white">
          <svg class="h-4.5 w-4.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path stroke-linecap="round" stroke-linejoin="round" d="M3 8.5l9-5 9 5-9 5-9-5z" />
            <path stroke-linecap="round" stroke-linejoin="round" d="M3 12.5l9 5 9-5" />
            <path stroke-linecap="round" stroke-linejoin="round" d="M3 16.5l9 5 9-5" />
          </svg>
        </span>
        <span class="text-xl font-bold text-white">察新</span>
      </div>
      <nav class="flex-1 space-y-1 px-3 py-4">
        <RouterLink
          v-for="item in navs"
          :key="item.to"
          :to="item.to"
          class="flex items-center gap-3 rounded-lg px-3 py-2.5 text-sm font-medium transition-colors"
          :class="
            route.path === item.to
              ? 'bg-slate-800 text-white'
              : 'text-slate-400 hover:bg-slate-800/60 hover:text-slate-200'
          "
        >
          <svg class="h-5 w-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">
            <path :d="item.icon" />
          </svg>
          {{ item.label }}
        </RouterLink>
      </nav>
      <div class="border-t border-slate-800 px-5 py-4 text-xs text-slate-500">
        GitHub Release 监控
        <div class="mt-1">shoutrrr 通知 · 自动轮询</div>
      </div>
    </aside>

    <div class="flex min-h-screen flex-1 flex-col">
      <header class="sticky top-0 z-30 flex h-14 items-center gap-3 border-b border-slate-200 bg-white/80 px-4 backdrop-blur dark:border-slate-800 dark:bg-slate-900/80 md:hidden">
        <span class="text-lg font-bold text-slate-900 dark:text-slate-100">察新</span>
        <nav class="ml-auto flex items-center gap-1">
          <RouterLink
            v-for="item in navs"
            :key="item.to"
            :to="item.to"
            class="rounded-lg px-2.5 py-1.5 text-sm font-medium"
            :class="route.path === item.to ? 'bg-slate-900 text-white dark:bg-slate-100 dark:text-slate-900' : 'text-slate-500 dark:text-slate-400'"
          >
            {{ item.label }}
          </RouterLink>
        </nav>
      </header>

      <main class="flex-1">
        <RouterView />
      </main>
    </div>

    <Toast />
  </div>
</template>
