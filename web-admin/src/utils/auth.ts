import { useUserStore } from '@/stores/user'

// 按钮级权限控制
export function hasPerm(code: string): boolean {
  const userStore = useUserStore()
  return userStore.hasPermission(code)
}

export function hasAnyPerm(codes: string[]): boolean {
  const userStore = useUserStore()
  return userStore.hasAnyPermission(codes)
}

// 指令：v-permission="'user:create'"
import type { Directive, DirectiveBinding } from 'vue'
export const permission: Directive = {
  mounted(el: HTMLElement, binding: DirectiveBinding<string>) {
    const code = binding.value
    if (!code) return
    if (!hasPerm(code)) {
      el.parentNode?.removeChild(el)
    }
  }
}