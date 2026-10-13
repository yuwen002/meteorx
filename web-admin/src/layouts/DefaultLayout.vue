<template>
  <el-container class="layout-container">
    <!-- 侧边栏 -->
    <el-aside :width="appStore.sidebarCollapsed ? '64px' : '220px'" class="sidebar">
      <div class="logo">
        <img v-if="brand.logo" :src="brand.logo" class="logo-img" alt="logo" />
        <span v-if="!appStore.sidebarCollapsed" class="logo-text">{{ brand.name }}</span>
        <span v-else class="logo-text">{{ (brand.name || 'M').charAt(0).toUpperCase() }}</span>
      </div>
      <el-menu
        :default-active="activeMenu"
        :collapse="appStore.sidebarCollapsed"
        :collapse-transition="false"
        router
        background-color="#1f2937"
        text-color="#cbd5e1"
        active-text-color="#fff"
      >
        <el-menu-item index="/dashboard">
          <el-icon><HomeFilled /></el-icon>
          <template #title>{{ t('nav.dashboard') }}</template>
        </el-menu-item>

        <el-sub-menu index="/workspace">
          <template #title>
            <el-icon><Monitor /></el-icon>
            <span>{{ t('nav.workspace') }}</span>
          </template>
          <el-menu-item index="/announcement">
            <el-icon><Bell /></el-icon>
            <template #title>{{ t('nav.announcement') }}</template>
          </el-menu-item>
          <el-menu-item index="/task">
            <el-icon><List /></el-icon>
            <template #title>{{ t('nav.task') }}</template>
          </el-menu-item>
          <el-menu-item index="/export">
            <el-icon><Download /></el-icon>
            <template #title>{{ t('nav.export') }}</template>
          </el-menu-item>
          <el-menu-item index="/tenant-settings">
            <el-icon><Setting /></el-icon>
            <template #title>{{ t('nav.tenantSettings') }}</template>
          </el-menu-item>
          <el-menu-item v-if="userStore.hasPermission('user:list') || userStore.isAdmin" index="/invitation">
            <el-icon><Message /></el-icon>
            <template #title>{{ t('nav.invitation') }}</template>
          </el-menu-item>
          <el-menu-item v-if="userStore.hasPermission('wiki:list') || userStore.isAdmin" index="/wiki">
            <el-icon><Reading /></el-icon>
            <template #title>{{ t('nav.wiki') }}</template>
          </el-menu-item>
          <el-menu-item v-if="userStore.hasPermission('file:list') || userStore.isAdmin" index="/system/file">
            <el-icon><Folder /></el-icon>
            <template #title>{{ t('nav.file') }}</template>
          </el-menu-item>
        </el-sub-menu>

        <el-sub-menu v-if="userStore.isAdmin" index="/platform">
          <template #title>
            <el-icon><Platform /></el-icon>
            <span>{{ t('nav.platform') }}</span>
          </template>
          <el-menu-item index="/system/user">
            <el-icon><User /></el-icon>
            <template #title>{{ t('nav.user') }}</template>
          </el-menu-item>
          <el-menu-item index="/system/role">
            <el-icon><UserFilled /></el-icon>
            <template #title>{{ t('nav.role') }}</template>
          </el-menu-item>
          <el-menu-item index="/system/permission">
            <el-icon><Lock /></el-icon>
            <template #title>{{ t('nav.permission') }}</template>
          </el-menu-item>
          <el-menu-item index="/system/master-admin">
            <el-icon><Avatar /></el-icon>
            <template #title>{{ t('nav.masterAdmin') }}</template>
          </el-menu-item>
        </el-sub-menu>

        <el-sub-menu v-if="userStore.isAdmin" index="/tenant-mgmt">
          <template #title>
            <el-icon><OfficeBuilding /></el-icon>
            <span>{{ t('nav.tenantMgmt') }}</span>
          </template>
          <el-menu-item index="/system/tenant">
            <el-icon><OfficeBuilding /></el-icon>
            <template #title>{{ t('nav.tenant') }}</template>
          </el-menu-item>
          <el-menu-item index="/system/plan">
            <el-icon><Goods /></el-icon>
            <template #title>{{ t('nav.plan') }}</template>
          </el-menu-item>
          <el-menu-item index="/system/dashboard/tenant-health">
            <el-icon><DataAnalysis /></el-icon>
            <template #title>{{ t('nav.tenantHealth') }}</template>
          </el-menu-item>
          <el-menu-item index="/system/cancel-request">
            <el-icon><CloseBold /></el-icon>
            <template #title>{{ t('nav.cancelRequest') }}</template>
          </el-menu-item>
        </el-sub-menu>

        <el-sub-menu v-if="userStore.isAdmin" index="/system">
          <template #title>
            <el-icon><Setting /></el-icon>
            <span>{{ t('nav.system') }}</span>
          </template>
          <el-menu-item index="/system/audit">
            <el-icon><Document /></el-icon>
            <template #title>{{ t('nav.audit') }}</template>
          </el-menu-item>
          <el-menu-item index="/system/audit/login-history">
            <el-icon><User /></el-icon>
            <template #title>{{ t('nav.loginHistory') }}</template>
          </el-menu-item>
          <el-menu-item index="/system/audit/timeline">
            <el-icon><Timer /></el-icon>
            <template #title>{{ t('nav.timeline') }}</template>
          </el-menu-item>
          <el-menu-item index="/system/audit/alert">
            <el-icon><Bell /></el-icon>
            <template #title>{{ t('nav.alert') }}</template>
          </el-menu-item>
          <el-menu-item index="/system/audit/anomaly">
            <el-icon><Warning /></el-icon>
            <template #title>{{ t('nav.anomaly') }}</template>
          </el-menu-item>
          <el-menu-item index="/system/audit/session">
            <el-icon><Connection /></el-icon>
            <template #title>{{ t('nav.session') }}</template>
          </el-menu-item>
          <el-menu-item index="/system/announcement">
            <el-icon><Bell /></el-icon>
            <template #title>{{ t('nav.announcementMgmt') }}</template>
          </el-menu-item>
        </el-sub-menu>
      </el-menu>
    </el-aside>

    <el-container>
      <!-- 顶栏 -->
      <el-header class="header">
        <div class="header-left">
          <el-icon class="collapse-btn" @click="appStore.toggleSidebar()">
            <Fold v-if="!appStore.sidebarCollapsed" />
            <Expand v-else />
          </el-icon>
          <el-breadcrumb separator="/">
            <el-breadcrumb-item :to="{ path: '/dashboard' }">{{ t('nav.dashboard') }}</el-breadcrumb-item>
            <el-breadcrumb-item v-if="currentRoute.meta.title && currentRoute.path !== '/dashboard'">{{
              currentRoute.meta.title
            }}</el-breadcrumb-item>
          </el-breadcrumb>
        </div>
        <div class="header-right">
          <el-tooltip :content="theme === 'dark' ? t('theme.toLight') : t('theme.toDark')" placement="bottom">
            <el-icon class="header-action" @click="toggleTheme">
              <Sunny v-if="theme === 'dark'" />
              <Moon v-else />
            </el-icon>
          </el-tooltip>
          <el-dropdown trigger="click" @command="handleLang">
            <span class="header-action lang-toggle">{{ locale === 'zh-CN' ? '中' : 'EN' }}</span>
            <template #dropdown>
              <el-dropdown-menu>
                <el-dropdown-item command="zh-CN" :class="{ 'lang-active': locale === 'zh-CN' }">简体中文</el-dropdown-item>
                <el-dropdown-item command="en-US" :class="{ 'lang-active': locale === 'en-US' }">English</el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>
          <GlobalNotificationCenter />
          <el-tooltip
            :content="notificationStore.wsConnected ? t('header.wsConnected') : t('header.wsDisconnected')"
            placement="bottom"
          >
            <span class="ws-status" :class="{ connected: notificationStore.wsConnected }">
              <span class="ws-dot"></span>
            </span>
          </el-tooltip>

          <el-dropdown @command="handleCommand">
            <span class="user-info">
              <el-avatar :size="32" style="background: #3b82f6">
                {{ (userStore.userInfo?.nickname || userStore.userInfo?.username || 'U').charAt(0).toUpperCase() }}
              </el-avatar>
              <span class="username">{{ userStore.userInfo?.nickname || userStore.userInfo?.username }}</span>
              <el-icon><ArrowDown /></el-icon>
            </span>
            <template #dropdown>
              <el-dropdown-menu>
                <el-dropdown-item command="profile">
                  <el-icon><User /></el-icon>{{ t('profile.title') }}
                </el-dropdown-item>
                <el-dropdown-item v-if="userStore.isAdmin" command="brand">
                  <el-icon><Brush /></el-icon>{{ t('brand.settings') }}
                </el-dropdown-item>
                <el-dropdown-item divided command="logout">
                  <el-icon><SwitchButton /></el-icon>{{ t('auth.logout') }}
                </el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>
        </div>
      </el-header>

      <el-main class="main">
        <router-view v-slot="{ Component }">
          <transition name="fade" mode="out-in">
            <component :is="Component" />
          </transition>
        </router-view>
      </el-main>
    </el-container>

    <!-- 白标品牌设置弹窗（仅管理员） -->
    <el-dialog v-model="brandDialogVisible" :title="t('brand.settings')" width="460px" :close-on-click-modal="false">
      <p style="color:#6b7280;font-size:13px;margin:0 0 16px;">
        {{ t('brand.hint') }}
      </p>
      <el-form label-width="88px">
        <el-form-item :label="t('brand.nameLabel')">
          <el-input v-model="brandForm.name" :placeholder="t('brand.namePlaceholder')" maxlength="30" />
        </el-form-item>
        <el-form-item :label="t('brand.logoLabel')">
          <el-input v-model="brandForm.logo" :placeholder="t('brand.logoPlaceholder')" clearable />
        </el-form-item>
        <el-form-item :label="t('brand.colorLabel')">
          <el-color-picker v-model="brandForm.primaryColor" />
          <span style="margin-left:12px;color:#909399;font-size:12px;">{{ brandForm.primaryColor || t('brand.defaultColor') }}</span>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="resetBrandConfig">{{ t('brand.reset') }}</el-button>
        <el-button @click="brandDialogVisible = false">{{ t('common.cancel') }}</el-button>
        <el-button type="primary" @click="saveBrandConfig">{{ t('common.save') }}</el-button>
      </template>
    </el-dialog>
  </el-container>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import {
  ArrowDown,
  Avatar,
  Bell,
  Brush,
  CloseBold,
  Connection,
  DataAnalysis,
  Document,
  Download,
  Expand,
  Fold,
  Folder,
  Goods,
  HomeFilled,
  List,
  Lock,
  Message,
  Monitor,
  Moon,
  OfficeBuilding,
  Platform,
  Reading,
  Setting,
  Sunny,
  SwitchButton,
  Timer,
  User,
  UserFilled,
  Warning
} from '@element-plus/icons-vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessageBox } from 'element-plus/es/components/message-box/index'
import { ElMessage } from 'element-plus/es/components/message/index'
import { useUserStore } from '@/stores/user'
import { useAppStore } from '@/stores/app'
import { useNotificationStore } from '@/stores/notification'
import { useTheme } from '@/composables/useTheme'
import { useI18n } from '@/composables/useI18n'
import { useBrand } from '@/composables/useBrand'
import GlobalNotificationCenter from '@/views/wiki/components/GlobalNotificationCenter.vue'

const router = useRouter()
const currentRoute = useRoute()
const userStore = useUserStore()
const appStore = useAppStore()
const notificationStore = useNotificationStore()

const { theme, toggleTheme } = useTheme()
const { locale, setLocale, t } = useI18n()
const { brand, setBrand, resetBrand } = useBrand()

// 白标品牌设置
const brandDialogVisible = ref(false)
const brandForm = reactive({ name: '', logo: '', primaryColor: '' })

function openBrandDialog() {
  brandForm.name = brand.value.name
  brandForm.logo = brand.value.logo
  brandForm.primaryColor = brand.value.primaryColor
  brandDialogVisible.value = true
}

function saveBrandConfig() {
  setBrand({ name: brandForm.name.trim() || 'MeteorX', logo: brandForm.logo.trim(), primaryColor: brandForm.primaryColor })
  ElMessage.success(t('brand.saved'))
  brandDialogVisible.value = false
}

function resetBrandConfig() {
  resetBrand()
  brandForm.name = 'MeteorX'
  brandForm.logo = ''
  brandForm.primaryColor = '#409eff'
  ElMessage.success(t('brand.resetDone'))
}

function handleLang(lang: string) {
  setLocale(lang)
}

const activeMenu = computed(() => currentRoute.path)

onMounted(() => {
  notificationStore.loadAnnouncements()
})

async function handleCommand(cmd: string) {
  if (cmd === 'logout') {
    ElMessageBox.confirm(t('auth.logoutConfirm'), t('common.tip'), {
      confirmButtonText: t('common.confirm'),
      cancelButtonText: t('common.cancel'),
      type: 'warning'
    })
      .then(async () => {
        await userStore.logout()
        ElMessage.success(t('auth.loggedOut'))
        router.push('/login')
      })
      .catch(() => {})
  } else if (cmd === 'profile') {
    router.push('/profile')
  } else if (cmd === 'brand') {
    openBrandDialog()
  }
}
</script>

<style scoped>
.layout-container {
  height: 100vh;
}
.sidebar {
  background-color: #1f2937;
  transition: width 0.25s;
  overflow: hidden;
  display: flex;
  flex-direction: column;
}

.sidebar :deep(.el-menu) {
  border-right: none;
  flex: 1;
  overflow-y: auto;
}

.sidebar :deep(.el-menu::-webkit-scrollbar) {
  width: 4px;
}

.sidebar :deep(.el-menu::-webkit-scrollbar-thumb) {
  background: rgba(255, 255, 255, 0.15);
  border-radius: 2px;
}

.sidebar :deep(.el-menu::-webkit-scrollbar-thumb:hover) {
  background: rgba(255, 255, 255, 0.25);
}
.logo {
  height: 56px;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  color: #fff;
  background-color: #111827;
  border-bottom: 1px solid #374151;
}
.logo-text {
  font-size: 18px;
  font-weight: 600;
}
.logo-img {
  height: 24px;
  width: 24px;
  object-fit: contain;
}
.header {
  background: var(--mx-header-bg);
  border-bottom: 1px solid var(--mx-header-border);
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 20px;
  height: 56px;
}
.header-left {
  display: flex;
  align-items: center;
  gap: 20px;
}
.collapse-btn {
  font-size: 20px;
  cursor: pointer;
  color: var(--mx-text-muted);
}
.collapse-btn:hover {
  color: var(--mx-text-strong);
}
.header-action {
  font-size: 18px;
  cursor: pointer;
  color: var(--mx-text-muted);
  margin-right: 16px;
  outline: none;
}
.header-action:hover {
  color: var(--mx-text-strong);
}
.lang-toggle {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 20px;
  height: 20px;
  padding: 0 6px;
  border: 1px solid var(--mx-header-border);
  border-radius: 4px;
  font-size: 12px;
  font-weight: 600;
  outline: none;
  cursor: pointer;
}
.lang-active {
  color: var(--el-color-primary);
  font-weight: 600;
}
.user-info {
  display: flex;
  align-items: center;
  gap: 8px;
  cursor: pointer;
}
.username {
  font-size: 14px;
  color: var(--mx-text);
}
.main {
  background: var(--mx-main-bg);
  padding: 16px;
}
.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.2s;
}
.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}
.ws-status {
  display: inline-flex;
  align-items: center;
  margin-right: 12px;
}
.ws-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background-color: #f56c6c;
  transition: background-color 0.3s;
}
.ws-status.connected .ws-dot {
  background-color: #67c23a;
  box-shadow: 0 0 4px rgba(103, 194, 58, 0.5);
}
</style>