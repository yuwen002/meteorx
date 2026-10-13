import { describe, it, expect } from 'vitest'
import { hasRouteAccess } from './index'

// 构造最小 user：hasPermission 依据给定权限码列表
function mkUser(opts: { isAdmin?: boolean; perms?: string[] }) {
  const perms = opts.perms ?? []
  return {
    isAdmin: !!opts.isAdmin,
    hasPermission: (code: string) => !!opts.isAdmin || perms.includes(code)
  }
}

describe('router guard: hasRouteAccess', () => {
  it('放行无 meta 的普通已登录用户', () => {
    expect(hasRouteAccess({}, mkUser({}))).toBe(true)
  })

  it('缺少 permission 时拦截', () => {
    const user = mkUser({ perms: ['user:list'] })
    expect(hasRouteAccess({ permission: 'wiki:list' }, user)).toBe(false)
    expect(hasRouteAccess({ permission: 'user:list' }, user)).toBe(true)
  })

  it('requireMaster 拦截非平台超管（即便拥有 admin:* 权限码）', () => {
    // 回归此前声明却未校验的缺口：租户管理员被授予 admin:tenant:list 也不得进入平台页
    const tenantAdmin = mkUser({ isAdmin: false, perms: ['admin:tenant:list'] })
    expect(hasRouteAccess({ permission: 'admin:tenant:list', requireMaster: true }, tenantAdmin)).toBe(false)
  })

  it('requireMaster 放行平台超管', () => {
    const master = mkUser({ isAdmin: true })
    expect(hasRouteAccess({ permission: 'admin:tenant:list', requireMaster: true }, master)).toBe(true)
  })

  it('仅 requireMaster 无 permission 时按超管身份判定', () => {
    expect(hasRouteAccess({ requireMaster: true }, mkUser({ isAdmin: false }))).toBe(false)
    expect(hasRouteAccess({ requireMaster: true }, mkUser({ isAdmin: true }))).toBe(true)
  })
})
