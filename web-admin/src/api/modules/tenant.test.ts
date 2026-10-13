import { describe, it, expect, vi, beforeEach } from 'vitest'
import {
  getTenantList,
  createTenant,
  getTenantDetail,
  updateTenant,
  deleteTenant,
  updateTenantStatus,
  batchUpdateTenantStatus,
  batchDeleteTenants,
  getDeletedTenantList,
  restoreTenant,
  getCancelRequestList,
  approveCancelRequest,
  rejectCancelRequest,
  getCurrentTenant,
  updateCurrentTenant,
  getTenantSettings,
  updateTenantSettings,
  type TenantItem,
  type CreateTenantParams,
} from './tenant'

vi.mock('@/api/request', () => ({
  get: vi.fn(),
  post: vi.fn(),
  put: vi.fn(),
  del: vi.fn(),
}))

import { get, post, put, del } from '@/api/request'

const sampleTenant: TenantItem = {
  id: 'tn-1',
  name: '测试租户',
  domain: 'test.example.com',
  status: 1,
  user_count: 3,
  plan_name: '基础版',
  created_at: '2026-08-13 23:45:03',
  updated_at: '2026-08-13 23:45:03',
}

const pageResp = { data: [sampleTenant], pagination: { total: 1, page: 1, page_size: 10 } }

describe('Tenant API', () => {
  beforeEach(() => vi.clearAllMocks())

  it('getTenantList 应携带查询参数请求 /admin/tenants', async () => {
    vi.mocked(get).mockResolvedValue(pageResp)
    const res = await getTenantList({ page: 1, page_size: 10, name: '测试' })
    expect(get).toHaveBeenCalledWith('/admin/tenants', { page: 1, page_size: 10, name: '测试' })
    expect(res.data[0].id).toBe('tn-1')
  })

  it('createTenant 应 POST /admin/tenants 并携带管理员信息', async () => {
    const payload: CreateTenantParams = {
      name: '新租户', domain: 'new.example.com',
      admin_user: { username: 'admin2', password: 'secret', nickname: '管理员' },
    }
    vi.mocked(post).mockResolvedValue(sampleTenant)
    await createTenant(payload)
    expect(post).toHaveBeenCalledWith('/admin/tenants', payload)
  })

  it('getTenantDetail 应 GET /admin/tenants/{id}/detail', async () => {
    vi.mocked(get).mockResolvedValue(sampleTenant)
    const res = await getTenantDetail('tn-1')
    expect(get).toHaveBeenCalledWith('/admin/tenants/tn-1/detail')
    expect(res.name).toBe('测试租户')
  })

  it('updateTenant 应 PUT /admin/tenants/{id}/update', async () => {
    vi.mocked(put).mockResolvedValue(sampleTenant)
    await updateTenant('tn-1', { name: '改名' })
    expect(put).toHaveBeenCalledWith('/admin/tenants/tn-1/update', { name: '改名' })
  })

  it('deleteTenant 应 DELETE /admin/tenants/{id}/delete', async () => {
    vi.mocked(del).mockResolvedValue(undefined)
    await deleteTenant('tn-1')
    expect(del).toHaveBeenCalledWith('/admin/tenants/tn-1/delete')
  })

  it('updateTenantStatus 应 PUT 状态端点并包装 status 请求体', async () => {
    vi.mocked(put).mockResolvedValue(undefined)
    await updateTenantStatus('tn-1', 0)
    expect(put).toHaveBeenCalledWith('/admin/tenants/tn-1/status', { status: 0 })
  })

  it('batchUpdateTenantStatus 应 PUT 批量状态端点', async () => {
    vi.mocked(put).mockResolvedValue(undefined)
    await batchUpdateTenantStatus({ ids: ['a', 'b'], status: 1 })
    expect(put).toHaveBeenCalledWith('/admin/tenants/batch/status', { ids: ['a', 'b'], status: 1 })
  })

  it('batchDeleteTenants 应 DELETE 批量端点并携带 data 请求体', async () => {
    vi.mocked(del).mockResolvedValue(undefined)
    await batchDeleteTenants({ ids: ['a', 'b'] })
    expect(del).toHaveBeenCalledWith('/admin/tenants/batch', { data: { ids: ['a', 'b'] } })
  })

  it('getDeletedTenantList 应 GET /admin/tenants/deleted 并透传分页', async () => {
    vi.mocked(get).mockResolvedValue(pageResp)
    await getDeletedTenantList({ page: 1, page_size: 10 })
    expect(get).toHaveBeenCalledWith('/admin/tenants/deleted', { page: 1, page_size: 10 })
  })

  it('restoreTenant 应 PUT 恢复端点', async () => {
    vi.mocked(put).mockResolvedValue(undefined)
    await restoreTenant('tn-1')
    expect(put).toHaveBeenCalledWith('/admin/tenants/tn-1/restore')
  })

  it('getCancelRequestList 应 GET /admin/cancel-requests 并透传过滤', async () => {
    vi.mocked(get).mockResolvedValue({ data: [], pagination: { total: 0, page: 1, page_size: 10 } })
    await getCancelRequestList({ page: 1, page_size: 10, status: 0 })
    expect(get).toHaveBeenCalledWith('/admin/cancel-requests', { page: 1, page_size: 10, status: 0 })
  })

  it('approveCancelRequest 应 PUT 审批端点并携带生效天数', async () => {
    vi.mocked(put).mockResolvedValue(undefined)
    await approveCancelRequest('cr-1', { effective_days: 7, review_remark: '同意' })
    expect(put).toHaveBeenCalledWith('/admin/cancel-requests/cr-1/approve', { effective_days: 7, review_remark: '同意' })
  })

  it('rejectCancelRequest 应 PUT 驳回端点并携带备注', async () => {
    vi.mocked(put).mockResolvedValue(undefined)
    await rejectCancelRequest('cr-1', { review_remark: '信息不全' })
    expect(put).toHaveBeenCalledWith('/admin/cancel-requests/cr-1/reject', { review_remark: '信息不全' })
  })

  it('getCurrentTenant 应 GET /tenants/current', async () => {
    vi.mocked(get).mockResolvedValue(sampleTenant)
    const res = await getCurrentTenant()
    expect(get).toHaveBeenCalledWith('/tenants/current')
    expect(res.domain).toBe('test.example.com')
  })

  it('updateCurrentTenant 应 PUT /tenants/current', async () => {
    vi.mocked(put).mockResolvedValue(sampleTenant)
    await updateCurrentTenant({ description: '新简介' })
    expect(put).toHaveBeenCalledWith('/tenants/current', { description: '新简介' })
  })

  it('getTenantSettings 应 GET /tenant-settings', async () => {
    vi.mocked(get).mockResolvedValue({ id: 'st-1', tenant_id: 'tn-1', theme: 'light' })
    const res = await getTenantSettings()
    expect(get).toHaveBeenCalledWith('/tenant-settings')
    expect(res.theme).toBe('light')
  })

  it('updateTenantSettings 应 PUT /tenant-settings 并透传部分字段', async () => {
    vi.mocked(put).mockResolvedValue(undefined)
    await updateTenantSettings({ theme: 'dark', language: 'zh-CN' })
    expect(put).toHaveBeenCalledWith('/tenant-settings', { theme: 'dark', language: 'zh-CN' })
  })
})
