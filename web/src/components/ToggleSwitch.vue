<script setup lang="ts">
const props = defineProps<{
  modelValue: boolean
  label?: string
  disabled?: boolean
}>()

const emit = defineEmits<{
  'update:modelValue': [value: boolean]
}>()

function toggle() {
  if (!props.disabled) {
    emit('update:modelValue', !props.modelValue)
  }
}
</script>

<template>
  <label
    :class="[
      'inline-flex items-center gap-3',
      disabled ? 'opacity-50 cursor-not-allowed' : 'cursor-pointer'
    ]"
  >
    <button
      type="button"
      role="switch"
      :aria-checked="modelValue"
      :disabled="disabled"
      @click="toggle"
      :class="[
        'relative inline-flex h-5 w-9 shrink-0 rounded-full transition-all duration-200',
        'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-indigo-500/30 focus-visible:ring-offset-2',
        'focus-visible:ring-offset-surface',
        modelValue
          ? 'bg-gradient-to-r from-indigo-500 to-violet-500 shadow-sm shadow-indigo-500/30'
          : 'bg-border/60 hover:bg-border'
      ]"
    >
      <span
        :class="[
          'inline-block h-4 w-4 rounded-full bg-white shadow-sm ring-0 transition-all duration-200 absolute top-0.5',
          modelValue ? 'translate-x-4' : 'translate-x-0.5'
        ]"
      />
    </button>
    <span v-if="label" class="text-sm font-medium">{{ label }}</span>
  </label>
</template>

<style scoped>
.bg-border\/60 {
  background: color-mix(in srgb, var(--color-border) 60%, transparent);
}

.ring-offset-surface {
  --tw-ring-offset-color: var(--color-surface);
}
</style>
