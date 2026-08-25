import { createRouter, createWebHashHistory, type RouteRecordRaw } from 'vue-router'
import { useUserStore } from '@/stores/user'

const routes: RouteRecordRaw[] = [
  {
    path: '/login',
    name: 'Login',
    component: () => import('@/views/login/index.vue'),
    meta: { title: '登录', public: true }
  },
  {
    path: '/forgot-password',
    name: 'ForgotPassword',
    component: () => import('@/views/forgot-password/index.vue'),
    meta: { title: '忘记密码', public: true }
  },
  {
    path: '/reset-password',
    name: 'ResetPassword',
    component: () => import('@/views/reset-password/index.vue'),
    meta: { title: '重置密码', public: true }
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
        meta: { title: '首页', icon: 'HomeFilled' }
      },
      {
        path: 'profile',
        name: 'Profile',
        component: () => import('@/views/profile/index.vue'),
        meta: { title: '个人中心', icon: 'User' }
      },
      {
        path: 'tenant-settings',
        name: 'TenantSettings',
        component: () => import('@/views/tenant/settings/index.vue'),
        meta: { title: '租户设置', icon: 'Setting' }
      },
      {
        path: 'wiki',
        name: 'WikiSpaceList',
        component: () => import('@/views/wiki/index.vue'),
        meta: { title: '知识库', icon: 'Reading', permission: 'wiki:list' }
      },
      {
        path: 'wiki/spaces/:id',
        name: 'WikiSpaceDetail',
        component: () => import('@/views/wiki/space.vue'),
        meta: { title: '知识空间', icon: 'Reading', permission: 'wiki:list' }
      },
      {
        path: 'announcement',
        name: 'Announcement',
        component: () => import('@/views/announcement/index.vue'),
        meta: { title: '平台公告', icon: 'Bell' }
      },
      {
        path: 'system/user',
        name: 'User',
        component: () => import('@/views/system/user/index.vue'),
        meta: { title: '用户管理', icon: 'User', permission: 'user:list' }
      },
      {
        path: 'system/role',
        name: 'Role',
        component: () => import('@/views/system/role/index.vue'),
        meta: { title: '角色管理', icon: 'UserFilled', permission: 'rbac:role:list' }
      },
      {
        path: 'system/permission',
        name: 'Permission',
        component: () => import('@/views/system/permission/index.vue'),
        meta: { title: '权限管理', icon: 'Lock', permission: 'rbac:perm:list' }
      },
      {
        path: 'system/master-admin',
        name: 'MasterAdmin',
        component: () => import('@/views/system/master-admin/index.vue'),
        meta: { title: '系统管理员', icon: 'Avatar', permission: 'admin:master:list', requireMaster: true }
      },
      {
        path: 'system/master-admin/recycle',
        name: 'MasterAdminRecycle',
        component: () => import('@/views/system/master-admin/recycle.vue'),
        meta: { title: '系统管理员回收站', icon: 'DeleteFilled', permission: 'admin:master:list', requireMaster: true }
      },
      {
        path: 'system/role/recycle',
        name: 'RoleRecycle',
        component: () => import('@/views/system/role/recycle.vue'),
        meta: { title: '角色回收站', icon: 'DeleteFilled', permission: 'rbac:role:list' }
      },
      {
        path: 'system/tenant',
        name: 'Tenant',
        component: () => import('@/views/system/tenant/index.vue'),
        meta: { title: '租户管理', icon: 'OfficeBuilding', permission: 'admin:tenant:list', requireMaster: true }
      },
      {
        path: 'system/tenant/recycle',
        name: 'TenantRecycle',
        component: () => import('@/views/system/tenant/recycle.vue'),
        meta: { title: '租户回收站', icon: 'DeleteFilled', permission: 'admin:tenant:list', requireMaster: true }
      },
      {
        path: 'system/audit',
        name: 'AuditLog',
        component: () => import('@/views/system/audit/index.vue'),
        meta: { title: '审计日志', icon: 'Document', permission: 'audit:log:list', requireMaster: true }
      },
      {
        path: 'system/cancel-request',
        name: 'CancelRequest',
        component: () => import('@/views/system/cancel-request/index.vue'),
        meta: { title: '注销审批', icon: 'CloseBold', permission: 'admin:cancel_request:list', requireMaster: true }
      },
      {
        path: 'system/plan',
        name: 'Plan',
        component: () => import('@/views/system/plan/index.vue'),
        meta: { title: '套餐管理', icon: 'Goods', permission: 'admin:plan:list', requireMaster: true }
      },
      {
        path: 'system/file',
        name: 'File',
        component: () => import('@/views/system/file/index.vue'),
        meta: { title: '文件管理', icon: 'Folder', permission: 'file:list' }
      },
      {
        path: 'system/announcement',
        name: 'SystemAnnouncement',
        component: () => import('@/views/system/announcement/index.vue'),
        meta: { title: '公告管理', icon: 'Bell', permission: 'admin:announcement:list', requireMaster: true }
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

  // 页面级权限校验
  const perm = to.meta.permission as string | undefined
  if (perm && !userStore.hasPermission(perm)) {
    // 没有该页面权限，跳首页并提示
    return next('/')
  }

  next()
})

router.afterEach((to) => {
  const title = to.meta.title as string | undefined
  if (title) {
    document.title = `${title} - MeteorX`
  }
})

export default router