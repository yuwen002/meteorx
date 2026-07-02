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