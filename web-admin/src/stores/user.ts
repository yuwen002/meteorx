import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { login, logout as apiLogout, type LoginParams, type LoginUserInfo } from '@/api/auth'

const TOKEN_KEY = 'meteorx_token'
const USER_KEY = 'meteorx_user'
const PERMS_KEY = 'meteorx_permissions'

export const useUserStore = defineStore('user', () => {
  const token = ref<string>(localStorage.getItem(TOKEN_KEY) || '')

  const initialUser = (() => {
    const raw = localStorage.getItem(USER_KEY)
    return raw ? (JSON.parse(raw) as LoginUserInfo) : null
  })()
  const userInfo = ref<LoginUserInfo | null>(initialUser)

  const initialPerms = (() => {
    const raw = localStorage.getItem(PERMS_KEY)
    return raw ? (JSON.parse(raw) as string[]) : []
  })()
  const permissions = ref<string[]>(initialPerms)

  const isLoggedIn = computed(() => !!token.value)
  const isAdmin = computed(() => !!userInfo.value?.is_master)

  async function doLogin(params: LoginParams) {
    const res = await login(params)
    // 根据后端返回结构取值：{ token: "...", user: {...}, permissions?: [...] }
    const data = (res as any).data || res
    token.value = data.token || ''
    userInfo.value = data.user || null
    // 如果后端没有返回 permissions，就用空数组（后续可以通过接口获取）
    permissions.value = data.permissions || []

    localStorage.setItem(TOKEN_KEY, token.value)
    if (userInfo.value) localStorage.setItem(USER_KEY, JSON.stringify(userInfo.value))
    localStorage.setItem(PERMS_KEY, JSON.stringify(permissions.value))
    return data
  }

  function setPermissions(list: string[]) {
    permissions.value = list
    localStorage.setItem(PERMS_KEY, JSON.stringify(list))
  }

  async function logout() {
    // 调用后端登出接口，将 token 加入黑名单
    try {
      await apiLogout()
    } catch (error) {
      // 即使后端调用失败，也继续清理本地状态
      console.error('Logout API call failed:', error)
    }

    // 清理本地状态
    clearLocalState()
  }

  // 同步登出（用于拦截器等不需要等待 API 响应的场景）
  function logoutSync() {
    clearLocalState()
  }

  function clearLocalState() {
    token.value = ''
    userInfo.value = null
    permissions.value = []
    localStorage.removeItem(TOKEN_KEY)
    localStorage.removeItem(USER_KEY)
    localStorage.removeItem(PERMS_KEY)
  }

  function hasPermission(code: string): boolean {
    if (!code) return true
    // 超级管理员默认拥有所有权限
    if (isAdmin.value) return true
    return permissions.value.includes(code)
  }

  function hasAnyPermission(codes: string[]): boolean {
    if (!codes || codes.length === 0) return true
    if (isAdmin.value) return true
    return codes.some((c) => permissions.value.includes(c))
  }

  return {
    token,
    userInfo,
    permissions,
    isLoggedIn,
    isAdmin,
    doLogin,
    setPermissions,
    logout,
    logoutSync,
    hasPermission,
    hasAnyPermission
  }
})