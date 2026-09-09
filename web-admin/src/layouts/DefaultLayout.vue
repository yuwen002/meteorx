<template>
  <el-container class="layout-container">
    <!-- 侧边栏 -->
    <el-aside :width="appStore.sidebarCollapsed ? '64px' : '220px'" class="sidebar">
      <div class="logo">
        <span v-if="!appStore.sidebarCollapsed" style="font-size: 18px; font-weight: 600">MeteorX</span>
        <span v-else style="font-size: 18px; font-weight: 600">M</span>
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
          <template #title>首页</template>
        </el-menu-item>

        <el-sub-menu index="/workspace">
          <template #title>
            <el-icon><Monitor /></el-icon>
            <span>工作台</span>
          </template>
          <el-menu-item index="/announcement">
            <el-icon><Bell /></el-icon>
            <template #title>平台公告</template>
          </el-menu-item>
          <el-menu-item index="/tenant-settings">
            <el-icon><Setting /></el-icon>
            <template #title>租户设置</template>
          </el-menu-item>
          <el-menu-item v-if="userStore.hasPermission('wiki:list') || userStore.isAdmin" index="/wiki">
            <el-icon><Reading /></el-icon>
            <template #title>知识库</template>
          </el-menu-item>
          <el-menu-item v-if="userStore.hasPermission('file:list') || userStore.isAdmin" index="/system/file">
            <el-icon><Folder /></el-icon>
            <template #title>文件管理</template>
          </el-menu-item>
        </el-sub-menu>

        <el-sub-menu v-if="userStore.isAdmin" index="/platform">
          <template #title>
            <el-icon><Platform /></el-icon>
            <span>平台管理</span>
          </template>
          <el-menu-item index="/system/user">
            <el-icon><User /></el-icon>
            <template #title>用户管理</template>
          </el-menu-item>
          <el-menu-item index="/system/role">
            <el-icon><UserFilled /></el-icon>
            <template #title>角色管理</template>
          </el-menu-item>
          <el-menu-item index="/system/permission">
            <el-icon><Lock /></el-icon>
            <template #title>权限管理</template>
          </el-menu-item>
          <el-menu-item index="/system/master-admin">
            <el-icon><Avatar /></el-icon>
            <template #title>系统管理员</template>
          </el-menu-item>
        </el-sub-menu>

        <el-sub-menu v-if="userStore.isAdmin" index="/tenant-mgmt">
          <template #title>
            <el-icon><OfficeBuilding /></el-icon>
            <span>租户运营</span>
          </template>
          <el-menu-item index="/system/tenant">
            <el-icon><OfficeBuilding /></el-icon>
            <template #title>租户管理</template>
          </el-menu-item>
          <el-menu-item index="/system/plan">
            <el-icon><Goods /></el-icon>
            <template #title>套餐管理</template>
          </el-menu-item>
          <el-menu-item index="/system/cancel-request">
            <el-icon><CloseBold /></el-icon>
            <template #title>注销审批</template>
          </el-menu-item>
        </el-sub-menu>

        <el-sub-menu v-if="userStore.isAdmin" index="/system">
          <template #title>
            <el-icon><Setting /></el-icon>
            <span>系统</span>
          </template>
          <el-menu-item index="/system/audit">
            <el-icon><Document /></el-icon>
            <template #title>审计日志</template>
          </el-menu-item>
          <el-menu-item index="/system/audit/login-history">
            <el-icon><User /></el-icon>
            <template #title>登录历史</template>
          </el-menu-item>
          <el-menu-item index="/system/audit/timeline">
            <el-icon><Timer /></el-icon>
            <template #title>用户时间线</template>
          </el-menu-item>
          <el-menu-item index="/system/audit/alert">
            <el-icon><Bell /></el-icon>
            <template #title>告警管理</template>
          </el-menu-item>
          <el-menu-item index="/system/audit/anomaly">
            <el-icon><Warning /></el-icon>
            <template #title>异常检测</template>
          </el-menu-item>
          <el-menu-item index="/system/audit/session">
            <el-icon><Connection /></el-icon>
            <template #title>会话分析</template>
          </el-menu-item>
          <el-menu-item index="/system/announcement">
            <el-icon><Bell /></el-icon>
            <template #title>公告管理</template>
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
            <el-breadcrumb-item :to="{ path: '/dashboard' }">首页</el-breadcrumb-item>
            <el-breadcrumb-item v-if="currentRoute.meta.title && currentRoute.path !== '/dashboard'">{{
              currentRoute.meta.title
            }}</el-breadcrumb-item>
          </el-breadcrumb>
        </div>
        <div class="header-right">
          <!-- 告警通知红点 -->
          <el-badge
            :value="notificationStore.unreadAlertCount"
            :hidden="notificationStore.unreadAlertCount === 0"
            class="alert-badge"
          >
            <el-tooltip content="告警通知" placement="bottom">
              <el-button
                :icon="Warning"
                circle
                :type="notificationStore.hasUnreadAlerts ? 'danger' : 'default'"
                @click="handleAlertClick"
              />
            </el-tooltip>
          </el-badge>

          <!-- WebSocket 连接状态指示器 -->
          <el-tooltip
            :content="notificationStore.wsConnected ? '实时连接已建立' : '实时连接已断开'"
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
                  <el-icon><User /></el-icon>个人信息
                </el-dropdown-item>
                <el-dropdown-item divided command="logout">
                  <el-icon><SwitchButton /></el-icon>退出登录
                </el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>
        </div>
      </el-header>

      <!-- 主内容 -->
      <el-main class="main">
        <router-view v-slot="{ Component }">
          <transition name="fade" mode="out-in">
            <component :is="Component" />
          </transition>
        </router-view>
      </el-main>
    </el-container>
  </el-container>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import {
  ArrowDown,
  Avatar,
  Bell,
  CloseBold,
  Document,
  Expand,
  Fold,
  Folder,
  Goods,
  HomeFilled,
  Lock,
  Monitor,
  OfficeBuilding,
  Platform,
  Reading,
  Setting,
  SwitchButton,
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

const router = useRouter()
const currentRoute = useRoute()
const userStore = useUserStore()
const appStore = useAppStore()
const notificationStore = useNotificationStore()

const activeMenu = computed(() => currentRoute.path)

async function handleCommand(cmd: string) {
  if (cmd === 'logout') {
    ElMessageBox.confirm('确定要退出登录吗？', '提示', {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: 'warning'
    })
      .then(async () => {
        await userStore.logout()
        ElMessage.success('已退出登录')
        router.push('/login')
      })
      .catch(() => {})
  } else if (cmd === 'profile') {
    router.push('/profile')
  }
}

function handleAlertClick() {
  notificationStore.markAllAlertsAsRead()
  router.push('/system/audit/alert')
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
}
.sidebar :deep(.el-menu) {
  border-right: none;
}
.logo {
  height: 56px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #fff;
  background-color: #111827;
  border-bottom: 1px solid #374151;
}
.header {
  background: #fff;
  border-bottom: 1px solid #e5e7eb;
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
  color: #4b5563;
}
.collapse-btn:hover {
  color: #111827;
}
.user-info {
  display: flex;
  align-items: center;
  gap: 8px;
  cursor: pointer;
}
.username {
  font-size: 14px;
  color: #374151;
}
.main {
  background: #f3f4f6;
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

/* 告警通知红点 */
.alert-badge {
  margin-right: 12px;
}
.alert-badge :deep(.el-badge__content) {
  background-color: #f56c6c;
}

/* WebSocket 连接状态指示器 */
.ws-status {
  display: inline-flex;
  align-items: center;
  margin-right: 16px;
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