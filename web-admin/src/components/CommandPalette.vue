<template>
  <!-- 命令面板：Ctrl/Cmd+K 唤起，↑↓ 选择、Enter 执行、Esc 关闭（由 el-dialog 焦点陷阱兜底） -->
  <el-dialog
    class="cmdk-dialog"
    :model-value="visible"
    :show-close="false"
    width="520px"
    top="12vh"
    append-to-body
    @update:model-value="onPaletteModel"
  >
    <div class="cmdk" :aria-label="t('cmdk.open')">
      <input
        ref="searchEl"
        v-model="query"
        class="cmdk-input"
        type="text"
        role="combobox"
        aria-expanded="true"
        aria-controls="cmdk-list"
        :placeholder="t('cmdk.placeholder')"
        @keydown.down.prevent="move(1)"
        @keydown.up.prevent="move(-1)"
        @keydown.enter.prevent="runActive"
      />
      <ul id="cmdk-list" class="cmdk-list" role="listbox" :aria-label="t('cmdk.open')">
        <li
          v-for="(c, i) in filtered"
          :key="c.id"
          role="option"
          :aria-selected="i === activeIndex"
          :class="['cmdk-item', c.section, { active: i === activeIndex }]"
          @mouseenter="activeIndex = i"
          @click="run(c)"
        >
          <span class="cmdk-label">{{ c.label }}</span>
          <span class="cmdk-hint">{{ c.hint || sectionLabel(c.section) }}</span>
        </li>
      </ul>
      <div v-if="filtered.length === 0" class="cmdk-empty">{{ t('cmdk.empty') }}</div>
      <div class="cmdk-footer">
        <span>{{ t('cmdk.hint.navigate') }}</span>
        <span>{{ t('cmdk.hint.enter') }}</span>
        <span>{{ t('cmdk.hint.esc') }}</span>
      </div>
    </div>
  </el-dialog>

  <!-- 快捷键总览：非输入态按 ? 唤起 -->
  <el-dialog v-model="helpVisible" class="cmdk-help" :title="t('cmdk.help.title')" width="440px" append-to-body>
    <dl class="cmdk-help-list">
      <div class="cmdk-help-row">
        <dt><kbd>Ctrl</kbd> / <kbd>⌘</kbd> + <kbd>K</kbd></dt>
        <dd>{{ t('cmdk.help.palette') }}</dd>
      </div>
      <div class="cmdk-help-row">
        <dt><kbd>?</kbd></dt>
        <dd>{{ t('cmdk.help.shortcuts') }}</dd>
      </div>
      <div class="cmdk-help-row">
        <dt><kbd>Esc</kbd></dt>
        <dd>{{ t('cmdk.help.escape') }}</dd>
      </div>
    </dl>
  </el-dialog>
</template>

<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessageBox } from 'element-plus/es/components/message-box/index'
import { ElMessage } from 'element-plus/es/components/message/index'
import { useCommandPalette } from '@/composables/useCommandPalette'
import { useUserStore } from '@/stores/user'
import { useAppStore } from '@/stores/app'
import { useTheme } from '@/composables/useTheme'
import { useI18n } from '@/composables/useI18n'
import { buildCommands, filterCommands, type CommandSection, type PaletteCommand } from '@/utils/commands'

const router = useRouter()
const userStore = useUserStore()
const appStore = useAppStore()
const { theme, toggleTheme } = useTheme()
const { locale, setLocale, t } = useI18n()
const { visible, close, toggle } = useCommandPalette()

const query = ref('')
const activeIndex = ref(0)
const helpVisible = ref(false)
const searchEl = ref<HTMLInputElement>()

// 组装命令上下文：导航关闭面板后跳转；操作类先关面板再执行，避免遮罩残留。
const commands = computed(() =>
  buildCommands({
    t,
    theme: theme.value,
    locale: locale.value,
    isAdmin: userStore.isAdmin,
    hasPermission: (code: string) => userStore.hasPermission(code),
    navigate: (path: string) => {
      close()
      void router.push(path)
    },
    toggleSidebar: () => {
      close()
      appStore.toggleSidebar()
    },
    toggleTheme: () => {
      close()
      toggleTheme()
    },
    setLocale: (l: string) => {
      close()
      setLocale(l)
    },
    logout: () => {
      close()
      confirmLogout()
    },
  })
)

const filtered = computed(() => filterCommands(commands.value, query.value))

// 打开时重置查询与高亮，并把焦点交给搜索框（键盘直达输入）。
watch(visible, (v) => {
  if (v) {
    query.value = ''
    activeIndex.value = 0
    nextTick(() => searchEl.value?.focus())
  }
})
watch(query, () => {
  activeIndex.value = 0
})

function move(delta: number) {
  const n = filtered.value.length
  if (!n) return
  activeIndex.value = (activeIndex.value + delta + n) % n
}

function runActive() {
  const c = filtered.value[activeIndex.value]
  if (c) run(c)
}

function run(c: PaletteCommand) {
  c.run()
}

function sectionLabel(s: CommandSection) {
  return s === 'nav' ? t('cmdk.section.nav') : t('cmdk.section.action')
}

function onPaletteModel(v: boolean) {
  if (!v) close()
}

function confirmLogout() {
  ElMessageBox.confirm(t('auth.logoutConfirm'), t('common.tip'), {
    confirmButtonText: t('common.confirm'),
    cancelButtonText: t('common.cancel'),
    type: 'warning',
  })
    .then(async () => {
      await userStore.logout()
      ElMessage.success(t('auth.loggedOut'))
      router.push('/login')
    })
    .catch(() => {})
}

// 判定焦点是否在可编辑元素内，避免在输入框打字时误触发全局快捷键。
function isEditable(el: EventTarget | null): boolean {
  const node = el as HTMLElement | null
  if (!node || typeof node.tagName !== 'string') return false
  const tag = node.tagName
  return tag === 'INPUT' || tag === 'TEXTAREA' || tag === 'SELECT' || node.isContentEditable
}

function onGlobalKey(e: KeyboardEvent) {
  const mod = e.ctrlKey || e.metaKey
  if (mod && (e.key === 'k' || e.key === 'K')) {
    e.preventDefault()
    toggle()
    return
  }
  if (e.key === '?' && !isEditable(e.target) && !visible.value && !helpVisible.value) {
    e.preventDefault()
    helpVisible.value = true
  }
}

onMounted(() => window.addEventListener('keydown', onGlobalKey))
onBeforeUnmount(() => window.removeEventListener('keydown', onGlobalKey))
</script>

<style scoped>
.cmdk {
  margin: -12px -20px;
}
.cmdk-input {
  width: 100%;
  border: none;
  border-bottom: 1px solid var(--el-border-color, #dcdfe6);
  padding: 8px 4px 12px;
  font-size: 15px;
  outline: none;
  background: transparent;
  color: inherit;
}
.cmdk-list {
  list-style: none;
  margin: 8px 0 0;
  padding: 0;
  max-height: 320px;
  overflow-y: auto;
}
.cmdk-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 8px 10px;
  border-radius: 6px;
  cursor: pointer;
  font-size: 14px;
}
.cmdk-item.active {
  background: var(--el-fill-color-light, #f5f7fa);
}
.cmdk-item .cmdk-hint {
  color: #909399;
  font-size: 12px;
}
.cmdk-item.action .cmdk-hint {
  color: var(--el-color-primary, #409eff);
}
.cmdk-empty {
  padding: 20px 0;
  text-align: center;
  color: #909399;
  font-size: 13px;
}
.cmdk-footer {
  display: flex;
  gap: 16px;
  padding-top: 10px;
  margin-top: 8px;
  border-top: 1px solid var(--el-border-color-lighter, #ebeef5);
  color: #909399;
  font-size: 12px;
}
.cmdk-help-list {
  margin: 0;
}
.cmdk-help-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 8px 0;
  border-bottom: 1px solid var(--el-border-color-lighter, #ebeef5);
}
.cmdk-help-row:last-child {
  border-bottom: none;
}
.cmdk-help-row dd {
  margin: 0;
  color: #606266;
}
.cmdk-help-row kbd {
  background: #f5f7fa;
  border: 1px solid #dcdfe6;
  border-radius: 4px;
  padding: 1px 6px;
  font-family: inherit;
  font-size: 12px;
}
</style>
