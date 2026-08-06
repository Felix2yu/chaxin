<script setup lang="ts">
import { ref } from 'vue'
import { useRoute } from 'vue-router'
import Toast from './components/Toast.vue'

const route = useRoute()
const mobileMenuOpen = ref(false)

const navItems = [
  {
    to: '/',
    label: '总览',
    icon: `<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 12l2-2m0 0l7-7 7 7M5 10v10a1 1 0 001 1h3m10-11l2 2m-2-2v10a1 1 0 01-1 1h-3m-4 0a1 1 0 01-1-1v-4a1 1 0 011-1h2a1 1 0 011 1v4a1 1 0 01-1 1"/>`,
  },
  {
    to: '/repos',
    label: '仓库管理',
    icon: `<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M7 21a4 4 0 01-4-4V5a2 2 0 012-2h4a2 2 0 012 2v12a4 4 0 01-4 4zm0 0h12a2 2 0 002-2v-4a2 2 0 00-2-2h-2.343M11 7.343l1.657-1.657a2 2 0 012.828 0l2.829 2.829a2 2 0 010 2.828l-8.486 8.485M7 17h.01"/>`,
  },
  {
    to: '/notifications',
    label: '通知中心',
    icon: `<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 17h5l-1.405-1.405A2.032 2.032 0 0118 14.158V11a6 6 0 10-12 0v3.159c0 .538-.214 1.055-.595 1.436L4 17h5m6 0v1a3 3 0 11-6 0v-1m6 0H9"/>`,
  },
  {
    to: '/settings',
    label: '系统设置',
    icon: `<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.066 2.573c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.573 1.066c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.066-2.573c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z"/><circle cx="12" cy="12" r="3" stroke-linecap="round" stroke-linejoin="round" stroke-width="2"/>`,
  },
]

function isActive(to: string) {
  if (to === '/') return route.path === '/'
  return route.path.startsWith(to)
}

function toggleMobileMenu() {
  mobileMenuOpen.value = !mobileMenuOpen.value
}

function navigate(to: string) {
  mobileMenuOpen.value = false
}
</script>

<template>
  <div class="flex h-screen overflow-hidden">
    <!-- Sidebar Backdrop (mobile) -->
    <div
      v-if="mobileMenuOpen"
      class="fixed inset-0 bg-black/30 backdrop-blur-sm z-30 lg:hidden"
      @click="mobileMenuOpen = false"
    />

    <!-- Sidebar -->
    <aside
      :class="[
        'sidebar flex flex-col z-40 h-full',
        'lg:translate-x-0 lg:static lg:flex',
        mobileMenuOpen ? 'translate-x-0 fixed inset-y-0 left-0' : '-translate-x-full fixed inset-y-0 left-0'
      ]"
    >
      <!-- Logo -->
      <div class="flex items-center gap-3 px-5 pt-6 pb-5">
        <div class="w-9 h-9 shrink-0">
          <img src="/icon.svg" alt="察新" class="w-9 h-9 rounded-xl" />
        </div>
        <div class="flex flex-col leading-none">
          <span class="text-lg font-bold tracking-tight">察新</span>
          <span class="text-xs text-muted">版本更新通知</span>
        </div>
      </div>

      <!-- Navigation -->
      <nav class="flex-1 px-3 py-2">
        <div class="text-xs font-semibold text-muted/60 uppercase tracking-wider px-3 mb-2">导航菜单</div>
        <ul class="space-y-1">
          <li v-for="item in navItems" :key="item.to">
            <router-link
              :to="item.to"
              @click="navigate(item.to)"
              :class="[
                'nav-item group flex items-center gap-3 px-3 py-2.5 rounded-xl text-sm font-medium transition-all duration-200',
                isActive(item.to)
                  ? 'nav-item-active'
                  : 'text-muted hover:text-foreground'
              ]"
            >
              <svg
                class="w-5 h-5 shrink-0 transition-transform duration-200 group-hover:scale-110"
                :class="{ 'text-indigo-500': isActive(item.to) }"
                fill="none" viewBox="0 0 24 24" stroke="currentColor"
                v-html="item.icon"
              />
              <span>{{ item.label }}</span>
              <div
                v-if="isActive(item.to)"
                class="ml-auto w-1.5 h-1.5 rounded-full bg-indigo-500"
              />
            </router-link>
          </li>
        </ul>
      </nav>
    </aside>

    <!-- Main Content -->
    <main class="flex-1 flex flex-col min-w-0 overflow-hidden">
      <!-- Top Bar -->
      <header class="h-14 shrink-0 flex items-center justify-between px-5 lg:px-8 border-b border-border/50 backdrop-blur-sm bg-surface/60">
        <!-- Mobile menu toggle -->
        <button
          @click="toggleMobileMenu"
          class="lg:hidden p-2 -ml-2 rounded-lg hover:bg-surface-alt transition-colors"
        >
          <svg class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 6h16M4 12h16M4 18h16"/>
          </svg>
        </button>

        <div class="flex-1" />

        <!-- Quick actions -->
        <div class="flex items-center gap-3">
          <a
            href="https://github.com"
            target="_blank"
            class="p-2 rounded-lg text-muted hover:text-foreground hover:bg-surface-alt transition-all duration-200"
            title="GitHub"
          >
            <svg class="w-5 h-5" fill="currentColor" viewBox="0 0 24 24">
              <path d="M12 0c-6.626 0-12 5.373-12 12 0 5.302 3.438 9.8 8.207 11.387.599.111.793-.261.793-.577v-2.234c-3.338.726-4.033-1.416-4.033-1.416-.546-1.387-1.333-1.756-1.333-1.756-1.089-.745.083-.729.083-.729 1.205.084 1.839 1.237 1.839 1.237 1.07 1.834 2.807 1.304 3.492.997.107-.775.418-1.305.762-1.604-2.665-.305-5.467-1.334-5.467-5.931 0-1.311.469-2.381 1.236-3.221-.124-.303-.535-1.524.117-3.176 0 0 1.008-.322 3.301 1.23.957-.266 1.983-.399 3.003-.404 1.02.005 2.047.138 3.006.404 2.291-1.552 3.297-1.23 3.297-1.23.653 1.653.242 2.874.118 3.176.77.84 1.235 1.911 1.235 3.221 0 4.609-2.807 5.624-5.479 5.921.43.372.823 1.102.823 2.222v3.293c0 .319.192.694.801.576 4.765-1.589 8.199-6.086 8.199-11.386 0-6.627-5.373-12-12-12z"/>
            </svg>
          </a>
        </div>
      </header>

      <!-- Page Content -->
      <div class="flex-1 overflow-auto">
        <div class="animate-fade-in">
          <router-view />
        </div>
      </div>
    </main>

    <!-- Toast -->
    <Toast />
  </div>
</template>

<style scoped>
.sidebar {
  width: 256px;
  background: linear-gradient(180deg, var(--color-surface) 0%, var(--color-surface-alt) 100%);
  border-right: 1px solid color-mix(in srgb, var(--color-border) 50%, transparent);
}

.nav-item-active {
  background: linear-gradient(135deg, rgba(99, 102, 241, 0.1) 0%, rgba(139, 92, 246, 0.08) 100%);
  color: var(--color-primary);
  box-shadow: 0 1px 2px rgba(99, 102, 241, 0.05);
}

:root.dark .nav-item-active {
  background: linear-gradient(135deg, rgba(99, 102, 241, 0.2) 0%, rgba(139, 92, 246, 0.15) 100%);
}

.text-muted {
  color: var(--color-text-muted);
}

.text-foreground {
  color: var(--color-text);
}

.border-border\/50 {
  border-color: color-mix(in srgb, var(--color-border) 50%, transparent);
}

.bg-surface\/60 {
  background-color: color-mix(in srgb, var(--color-surface) 60%, transparent);
}

.bg-surface-alt\/80 {
  background-color: color-mix(in srgb, var(--color-surface-alt) 80%, transparent);
}

@media (max-width: 1023px) {
  .sidebar {
    width: 260px;
    box-shadow: 8px 0 30px rgba(0, 0, 0, 0.15);
  }
}
</style>
