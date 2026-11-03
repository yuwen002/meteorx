import { ref, readonly, type Ref } from 'vue'

// 命令面板开关状态：模块级单例，使顶栏触发按钮（DefaultLayout）与面板组件
// （CommandPalette.vue）跨组件共享同一 open 态，无需 props/emit 层层传递。
const visible = ref(false)

export interface CommandPaletteApi {
  visible: Readonly<Ref<boolean>>
  open: () => void
  close: () => void
  toggle: () => void
}

export function useCommandPalette(): CommandPaletteApi {
  return {
    visible: readonly(visible),
    open: () => {
      visible.value = true
    },
    close: () => {
      visible.value = false
    },
    toggle: () => {
      visible.value = !visible.value
    },
  }
}
