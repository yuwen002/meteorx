import { describe, it, expect, vi, beforeEach } from 'vitest'

// 用 vi.hoisted 构造跨 hoisted mock 工厂共享的桩，规避变量提升时序问题
const h = vi.hoisted(() => {
  const service = {
    request: vi.fn(),
    interceptors: {
      request: { use: vi.fn() },
      response: { use: vi.fn() },
    },
  }
  return {
    service,
    ElMessage: { error: vi.fn(), warning: vi.fn() },
    userStore: { token: '', logoutSync: vi.fn(), logout: vi.fn() },
    router: { push: vi.fn() },
  }
})

vi.mock('axios', () => ({ default: { create: () => h.service } }))
vi.mock('element-plus/es/components/message/index', () => ({ ElMessage: h.ElMessage }))
vi.mock('@/stores/user', () => ({ useUserStore: () => h.userStore }))
vi.mock('@/stores/i18n', () => ({ useI18nStore: () => ({ t: (key: string) => key }) }))
vi.mock('@/router', () => ({ default: h.router }))

import { get, post, put, del, request } from './request'

// 模块导入时已调用 interceptors.*.use，取出注册的处理器
const [reqFulfilled, reqRejected] = h.service.interceptors.request.use.mock.calls[0]
const [resFulfilled, resRejected] = h.service.interceptors.response.use.mock.calls[0]

describe('request 封装方法委托 service.request', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    h.service.request.mockResolvedValue('OK')
  })

  it('request 透传 config', async () => {
    const result = await request({ url: '/u', method: 'GET' })
    expect(h.service.request).toHaveBeenCalledWith({ url: '/u', method: 'GET' })
    expect(result).toBe('OK')
  })

  it('get 以 GET + params 调用', async () => {
    await get('/u', { a: 1 }, { timeout: 5 })
    expect(h.service.request).toHaveBeenCalledWith({ url: '/u', method: 'GET', params: { a: 1 }, timeout: 5 })
  })

  it('post 以 POST + data 调用', async () => {
    await post('/u', { b: 2 })
    expect(h.service.request).toHaveBeenCalledWith({ url: '/u', method: 'POST', data: { b: 2 } })
  })

  it('put 以 PUT + data 调用', async () => {
    await put('/u', { c: 3 })
    expect(h.service.request).toHaveBeenCalledWith({ url: '/u', method: 'PUT', data: { c: 3 } })
  })

  it('del 以 DELETE 调用', async () => {
    await del('/u')
    expect(h.service.request).toHaveBeenCalledWith({ url: '/u', method: 'DELETE' })
  })
})

describe('请求拦截器', () => {
  beforeEach(() => vi.clearAllMocks())

  it('存在 token 时注入 Authorization 头', () => {
    h.userStore.token = 'tk'
    const config = { headers: {} as Record<string, string> }
    expect(reqFulfilled(config).headers.Authorization).toBe('Bearer tk')
  })

  it('已显式设置 Authorization 时不被全局 token 覆盖', () => {
    h.userStore.token = 'tk'
    const config = { headers: { Authorization: 'Bearer pending-tk' } as Record<string, string> }
    expect(reqFulfilled(config).headers.Authorization).toBe('Bearer pending-tk')
  })

  it('无 token 时不注入 Authorization 头', () => {
    h.userStore.token = ''
    const config = { headers: {} as Record<string, string> }
    expect(reqFulfilled(config).headers.Authorization).toBeUndefined()
  })

  it('请求错误被 reject', async () => {
    const err = new Error('req fail')
    await expect(reqRejected(err)).rejects.toThrow('req fail')
  })
})

describe('响应拦截器 - 成功分支', () => {
  beforeEach(() => vi.clearAllMocks())

  it('业务错误码非 401 时提示并 reject', async () => {
    await expect(resFulfilled({ data: { code: 500, message: 'boom' } })).rejects.toThrow('boom')
    expect(h.ElMessage.error).toHaveBeenCalledWith('boom')
  })

  it('业务错误码 401 时触发登出同步并跳转登录', async () => {
    await expect(resFulfilled({ data: { code: 401 } })).rejects.toThrow()
    expect(h.userStore.logoutSync).toHaveBeenCalled()
    expect(h.ElMessage.warning).toHaveBeenCalled()
    expect(h.router.push).toHaveBeenCalledWith('/login')
  })

  it('分页响应保留 data 与 pagination', () => {
    const out = resFulfilled({ data: { code: 200, data: [1, 2], pagination: { page: 1, total: 2 } } })
    expect(out).toEqual({ data: [1, 2], pagination: { page: 1, total: 2 } })
  })

  it('正常响应返回 data 字段', () => {
    expect(resFulfilled({ data: { code: 0, data: 'ok' } })).toBe('ok')
  })

  it('无 data 字段时回退返回整体', () => {
    expect(resFulfilled({ data: [7, 8] })).toEqual([7, 8])
  })

  it('logoutSync 缺失时回退调用 logout', async () => {
    const origin = h.userStore.logoutSync
    // @ts-expect-error 临时移除以覆盖 else 分支
    h.userStore.logoutSync = undefined
    await expect(resFulfilled({ data: { code: 40001 } })).rejects.toThrow()
    expect(h.userStore.logout).toHaveBeenCalled()
    h.userStore.logoutSync = origin
  })
})

describe('响应拦截器 - 错误分支', () => {
  beforeEach(() => vi.clearAllMocks())

  it('HTTP 401 触发登出', async () => {
    const err = { response: { status: 401 } }
    await expect(resRejected(err)).rejects.toBe(err)
    expect(h.userStore.logoutSync).toHaveBeenCalled()
    expect(h.router.push).toHaveBeenCalledWith('/login')
  })

  it('非 401 状态码提示后端 message', async () => {
    const err = { response: { status: 500, data: { message: 'server err' }, statusText: '' } }
    await expect(resRejected(err)).rejects.toBe(err)
    expect(h.ElMessage.error).toHaveBeenCalledWith('server err')
  })

  it('非 401 无 message 时回退 statusText', async () => {
    const err = { response: { status: 404, data: {}, statusText: 'Not Found' } }
    await expect(resRejected(err)).rejects.toBe(err)
    expect(h.ElMessage.error).toHaveBeenCalledWith('Not Found')
  })

  it('无响应（网络错误）提示网络请求失败', async () => {
    const err = { message: 'Network Error' }
    await expect(resRejected(err)).rejects.toBe(err)
    expect(h.ElMessage.error).toHaveBeenCalledWith('Network Error')
  })
})
