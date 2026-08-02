import { reactive } from 'vue'

export interface ToastItem {
  id: number
  type: 'success' | 'error' | 'info'
  message: string
}

export const toasts = reactive<ToastItem[]>([])

let seq = 0

export function useToast() {
  function push(type: ToastItem['type'], message: string, duration = 3500) {
    const id = ++seq
    toasts.push({ id, type, message })
    setTimeout(() => {
      const idx = toasts.findIndex((t) => t.id === id)
      if (idx !== -1) toasts.splice(idx, 1)
    }, duration)
  }
  return {
    success: (m: string) => push('success', m),
    error: (m: string) => push('error', m),
    info: (m: string) => push('info', m),
  }
}
