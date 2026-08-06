<script setup lang="ts">
import { toasts } from './toast'

const icons: Record<string, string> = {
  success: `<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z"/>`,
  error: `<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 14l2-2m0 0l2-2m-2 2l-2-2m2 2l2 2m7-2a9 9 0 11-18 0 9 9 0 0118 0z"/>`,
  info: `<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"/>`,
}

const colors: Record<string, string> = {
  success: 'border-emerald-200 dark:border-emerald-500/20 shadow-emerald-500/5',
  error: 'border-rose-200 dark:border-rose-500/20 shadow-rose-500/5',
  info: 'border-sky-200 dark:border-sky-500/20 shadow-sky-500/5',
}

const iconColors: Record<string, string> = {
  success: 'text-emerald-500',
  error: 'text-rose-500',
  info: 'text-sky-500',
}
</script>

<template>
  <div class="fixed bottom-6 right-6 z-50 flex flex-col-reverse gap-2 pointer-events-none">
    <div
      v-for="t in toasts"
      :key="t.id"
      :class="[
        'animate-slide-down pointer-events-auto flex items-center gap-3 px-4 py-3 rounded-xl border bg-surface/90 backdrop-blur-md shadow-lg min-w-[280px] max-w-sm',
        colors[t.type] || ''
      ]"
    >
      <svg :class="['w-5 h-5 shrink-0', iconColors[t.type]]" fill="none" viewBox="0 0 24 24" stroke="currentColor" v-html="icons[t.type]" />
      <span class="text-sm flex-1">{{ t.message }}</span>
    </div>
  </div>
</template>

<style scoped>
.bg-surface\/90 {
  background: color-mix(in srgb, var(--color-surface) 90%, transparent);
}
</style>
