import { createRouter, createWebHashHistory, type RouteRecordRaw } from 'vue-router'
import { useUserStore } from '@/stores/user'
import { useBrandStore } from '@/stores/brand'
import { useI18nStore } from '@/stores/i18n'

const routes: RouteRecordRaw[] = [
  {
    path: '/login',
    name: 'Login',
    component: () => import('@/views/login/index.vue'),
    meta: { titleKey: 'route.login', public: true }
  },
  {
    path: '/register',
    name: 'Register',
    component: () => import('@/views/register/index.vue'),
    meta: { titleKey: 'route.register', public: true }
  },
  {
    path: '/register-user',
    name: 'RegisterUser',
    component: () => import('@/views/register-user/index.vue'),
    meta: { titleKey: 'route.registerUser', public: true }
  },
  {
    path: '/forgot-password',
    name: 'ForgotPassword',
    component: () => import('@/views/forgot-password/index.vue'),
    meta: { titleKey: 'route.forgotPassword', public: true }
  },
  {
    path: '/reset-password',
    name: 'ResetPassword',
    component: () => import('@/views/reset-password/index.vue'),
    meta: { titleKey: 'route.resetPassword', public: true }
  },
  {
    path: '/wiki/share/:token',
    name: 'WikiShareView',
    component: () => import('@/views/wiki/ShareView.vue'),
    meta: { titleKey: 'route.wikiShare', public: true }
  },
  {
    path: '/auth/:provider/callback',
    name: 'OAuthCallback',
    component: () => import('@/views/login/OAuthCallback.vue'),
    meta: { titleKey: 'route.oauthCallback', public: true }
  },
  {
    path: '/oauth/complete-email',
    name: 'OAuthCompleteEmail',
    component: () => import('@/views/login/CompleteEmail.vue'),
    meta: { titleKey: 'route.completeEmail', public: true }
  },
  {
    path: '/verify-email',
    name: 'VerifyEmail',
    component: () => import('@/views/verify-email/index.vue'),
    meta: { titleKey: 'route.verifyEmail', public: true }
  },
  {
    path: '/accept-invitation',
    name: 'AcceptInvitation',
    component: () => import('@/views/accept-invitation/index.vue'),
    meta: { titleKey: 'route.acceptInvitation', public: true }
  },
  {
    path: '/',
    component: () => import('@/layouts/DefaultLayout.vue'),
    redirect: '/dashboard',
    children: [
      {
        path: 'dashboard',
        name: 'Dashboard',
        component: () => import('@/views/dashboard/index.vue'),
        meta: { titleKey: 'route.dashboard', icon: 'HomeFilled' }
      },
      {
        path: 'profile',
        name: 'Profile',
        component: () => import('@/views/profile/index.vue'),
        meta: { titleKey: 'route.profile', icon: 'User' }
      },
      {
        path: 'tenant-settings',
        name: 'TenantSettings',
        component: () => import('@/views/tenant/settings/index.vue'),
        meta: { titleKey: 'route.tenantSettings', icon: 'Setting' }
      },
      {
        path: 'invitation',
        name: 'Invitation',
        component: () => import('@/views/invitation/index.vue'),
        meta: { titleKey: 'route.invitation', icon: 'Message', permission: 'user:list' }
      },
      {
        path: 'task',
        name: 'Task',
        component: () => import('@/views/task/index.vue'),
        meta: { titleKey: 'route.task', icon: 'List' }
      },
      {
        path: 'chat',
        name: 'Chat',
        component: () => import('@/views/chat/index.vue'),
        meta: { titleKey: 'route.chat', icon: 'ChatLineRound' }
      },
      {
        path: 'export',
        name: 'Export',
        component: () => import('@/views/export/index.vue'),
        meta: { titleKey: 'route.export', icon: 'Download' }
      },
      {
        path: 'wiki',
        name: 'WikiSpaceList',
        component: () => import('@/views/wiki/index.vue'),
        meta: { titleKey: 'route.wiki', icon: 'Reading', permission: 'wiki:list' }
      },
      {
        path: 'wiki/spaces/:id',
        name: 'WikiSpaceDetail',
        component: () => import('@/views/wiki/space.vue'),
        meta: { titleKey: 'route.wikiSpace', icon: 'Reading', permission: 'wiki:list' }
      },
      {
        path: 'wiki/search',
        name: 'WikiSearch',
        component: () => import('@/views/wiki/search.vue'),
        meta: { titleKey: 'route.wikiSearch', icon: 'Search', permission: 'wiki:list' }
      },
      {
        path: 'wiki/trash',
        name: 'WikiTrash',
        component: () => import('@/views/wiki/trash.vue'),
        meta: { titleKey: 'route.wikiTrash', icon: 'Delete', permission: 'wiki:list' }
      },
      {
        path: 'wiki/pending-reviews',
        name: 'WikiPendingReviews',
        component: () => import('@/views/wiki/pending-reviews.vue'),
        meta: { titleKey: 'route.wikiPending', icon: 'DocumentChecked', permission: 'wiki:list' }
      },
      {
        path: 'announcement',
        name: 'Announcement',
        component: () => import('@/views/announcement/index.vue'),
        meta: { titleKey: 'route.announcement', icon: 'Bell' }
      },
      {
        path: 'messages',
        name: 'Messages',
        component: () => import('@/views/message/index.vue'),
        meta: { titleKey: 'route.messages', icon: 'ChatDotRound' }
      },
      {
        path: 'system/user',
        name: 'User',
        component: () => import('@/views/system/user/index.vue'),
        meta: { titleKey: 'route.user', icon: 'User', permission: 'user:list' }
      },
      {
        path: 'system/role',
        name: 'Role',
        component: () => import('@/views/system/role/index.vue'),
        meta: { titleKey: 'route.role', icon: 'UserFilled', permission: 'rbac:role:list' }
      },
      {
        path: 'system/permission',
        name: 'Permission',
        component: () => import('@/views/system/permission/index.vue'),
        meta: { titleKey: 'route.permission', icon: 'Lock', permission: 'rbac:perm:list' }
      },
      {
        path: 'system/master-admin',
        name: 'MasterAdmin',
        component: () => import('@/views/system/master-admin/index.vue'),
        meta: { titleKey: 'route.masterAdmin', icon: 'Avatar', permission: 'admin:master:list', requireMaster: true }
      },
      {
        path: 'system/master-admin/recycle',
        name: 'MasterAdminRecycle',
        component: () => import('@/views/system/master-admin/recycle.vue'),
        meta: { titleKey: 'route.masterAdminRecycle', icon: 'DeleteFilled', permission: 'admin:master:list', requireMaster: true }
      },
      {
        path: 'system/role/recycle',
        name: 'RoleRecycle',
        component: () => import('@/views/system/role/recycle.vue'),
        meta: { titleKey: 'route.roleRecycle', icon: 'DeleteFilled', permission: 'rbac:role:list' }
      },
      {
        path: 'system/tenant',
        name: 'Tenant',
        component: () => import('@/views/system/tenant/index.vue'),
        meta: { titleKey: 'route.tenant', icon: 'OfficeBuilding', permission: 'admin:tenant:list', requireMaster: true }
      },
      {
        path: 'system/tenant/recycle',
        name: 'TenantRecycle',
        component: () => import('@/views/system/tenant/recycle.vue'),
        meta: { titleKey: 'route.tenantRecycle', icon: 'DeleteFilled', permission: 'admin:tenant:list', requireMaster: true }
      },
      {
        path: 'system/audit',
        name: 'AuditLog',
        component: () => import('@/views/system/audit/index.vue'),
        meta: { titleKey: 'route.audit', icon: 'Document', permission: 'audit:log:list', requireMaster: true }
      },
      {
        path: 'system/audit/login-history',
        name: 'AuditLoginHistory',
        component: () => import('@/views/system/audit/login-history.vue'),
        meta: { titleKey: 'route.loginHistory', icon: 'User', permission: 'audit:log:list', requireMaster: true }
      },
      {
        path: 'system/audit/timeline',
        name: 'AuditTimeline',
        component: () => import('@/views/system/audit/timeline.vue'),
        meta: { titleKey: 'route.timeline', icon: 'Timer', permission: 'audit:log:list', requireMaster: true }
      },
      {
        path: 'system/audit/anomaly',
        name: 'AuditAnomaly',
        component: () => import('@/views/system/audit/anomaly.vue'),
        meta: { titleKey: 'route.anomaly', icon: 'Warning', permission: 'audit:alert:list', requireMaster: true }
      },
      {
        path: 'system/audit/alert',
        name: 'AuditAlert',
        component: () => import('@/views/system/audit/alert.vue'),
        meta: { titleKey: 'route.alert', icon: 'Bell', permission: 'audit:alert-rule:list', requireMaster: true }
      },
      {
        path: 'system/audit/session',
        name: 'AuditSession',
        component: () => import('@/views/system/audit/session.vue'),
        meta: { titleKey: 'route.session', icon: 'Connection', permission: 'audit:session:list', requireMaster: true }
      },
      {
        path: 'system/audit/integrity',
        name: 'AuditIntegrity',
        component: () => import('@/views/system/audit/integrity.vue'),
        meta: { titleKey: 'route.integrity', icon: 'CircleCheck', permission: 'audit:integrity:verify', requireMaster: true }
      },
      {
        path: 'system/cancel-request',
        name: 'CancelRequest',
        component: () => import('@/views/system/cancel-request/index.vue'),
        meta: { titleKey: 'route.cancelRequest', icon: 'CloseBold', permission: 'admin:cancel_request:list', requireMaster: true }
      },
      {
        path: 'system/plan',
        name: 'Plan',
        component: () => import('@/views/system/plan/index.vue'),
        meta: { titleKey: 'route.plan', icon: 'Goods', permission: 'admin:plan:list', requireMaster: true }
      },
      {
        path: 'system/dashboard/tenant-health',
        name: 'TenantHealth',
        component: () => import('@/views/system/dashboard/tenant-health.vue'),
        meta: { titleKey: 'route.tenantHealth', icon: 'DataAnalysis', permission: 'admin:dashboard:list', requireMaster: true }
      },
      {
        path: 'system/dashboard/slow-queries',
        name: 'SlowQueries',
        component: () => import('@/views/system/dashboard/slow-queries.vue'),
        meta: { titleKey: 'route.slowQueries', icon: 'Odometer', permission: 'admin:dashboard:list', requireMaster: true }
      },
      {
        path: 'system/file',
        name: 'File',
        component: () => import('@/views/system/file/index.vue'),
        meta: { titleKey: 'route.file', icon: 'Folder', permission: 'file:list' }
      },
      {
        path: 'system/announcement',
        name: 'SystemAnnouncement',
        component: () => import('@/views/system/announcement/index.vue'),
        meta: { titleKey: 'route.announcementMgmt', icon: 'Bell', permission: 'admin:announcement:list', requireMaster: true }
      },
      {
        path: 'system/oauth-provider',
        name: 'OAuthProvider',
        component: () => import('@/views/system/oauth-provider/index.vue'),
        meta: { titleKey: 'route.oauthProvider', icon: 'Link', permission: 'admin:oauth_provider:list', requireMaster: true }
      }
    ]
  },
  {
    path: '/:pathMatch(.*)*',
    name: 'NotFound',
    component: () => import('@/views/dashboard/index.vue')
  }
]

const router = createRouter({
  history: createWebHashHistory(),
  routes
})

// 路由访问决策（纯函数，便于单测）：在登录校验通过后，按 permission 与 requireMaster 判定是否放行。
export interface GuardUser {
  isAdmin: boolean
  hasPermission: (code: string) => boolean
}
export interface GuardMeta {
  permission?: string
  requireMaster?: boolean
}
export function hasRouteAccess(meta: GuardMeta, user: GuardUser): boolean {
  const perm = meta.permission
  if (perm && !user.hasPermission(perm)) return false
  // 平台级门禁：requireMaster 路由仅限平台超管（is_master）
  if (meta.requireMaster && !user.isAdmin) return false
  return true
}

// 全局前置守卫：登录态校验 + 权限校验
router.beforeEach((to, _from, next) => {
  const userStore = useUserStore()

  // 公开页面（登录页）直接放行
  if (to.meta.public) {
    if (to.path === '/login' && userStore.isLoggedIn) {
      // 已登录用户访问登录页，跳首页
      return next('/')
    }
    return next()
  }

  // 未登录，跳登录页
  if (!userStore.isLoggedIn) {
    return next({ path: '/login', query: { redirect: to.fullPath } })
  }

  // 页面级权限校验 + 平台级 requireMaster 门禁
  // 修正此前 requireMaster 声明却从未校验的缺口——避免拥有 admin:* 权限码的租户管理员越级进入平台页
  if (!hasRouteAccess(to.meta as GuardMeta, userStore)) {
    return next('/')
  }

  next()
})

router.afterEach((to) => {
  const titleKey = to.meta.titleKey as string | undefined
  // Resolve the page title in the current locale; brand name comes from the white-label store
  const brandName = useBrandStore().brand.name || 'MeteorX'
  const title = titleKey ? useI18nStore().t(titleKey) : ''
  document.title = title ? `${title} - ${brandName}` : brandName
})

export default router