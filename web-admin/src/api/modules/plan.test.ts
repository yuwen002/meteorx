import { describe, it, expect, vi, beforeEach } from 'vitest'
import {
  getPlanList,
  getPlanSelect,
  createPlan,
  updatePlan,
  deletePlan,
  getTenantPlan,
  assignTenantPlan,
  getMyPlan,
  type PlanItem,
  type CurrentPlan,
  type CreatePlanParams,
} from './plan'

vi.mock('@/api/request', () => ({
  get: vi.fn(),
  post: vi.fn(),
  put: vi.fn(),
  del: vi.fn(),
}))

import { get, post, put, del } from '@/api/request'

const samplePlan: PlanItem = {
  id: 'plan-1',
  name: '专业版',
  code: 'professional',
  user_limit: 1000,
  storage_limit: 1073741824,
  api_rate_limit: 600,
  wiki_space_limit: 20,
  price: 999,
  status: 1,
  subscriber_cnt: 3,
}

const sampleCurrent: CurrentPlan = {
  tenant_id: 'tn-1',
  plan_id: 'plan-1',
  plan_name: '专业版',
  plan_code: 'professional',
  user_limit: 1000,
  current_users: 12,
  storage_limit: -1,
  current_storage: 1024,
  wiki_space_limit: -1,
  current_wiki_spaces: 2,
  api_rate_limit: -1,
  status: 1,
}

describe('Plan API', () => {
  beforeEach(() => vi.clearAllMocks())

  it('getPlanList 应携带分页与过滤参数请求 /admin/plans', async () => {
    vi.mocked(get).mockResolvedValue({ data: [samplePlan], pagination: { total: 1, page: 1, page_size: 10 } })
    const res = await getPlanList({ page: 1, page_size: 10, keyword: '专业', status: 1 })
    expect(get).toHaveBeenCalledWith('/admin/plans', { page: 1, page_size: 10, keyword: '专业', status: 1 })
    expect(res.data[0].code).toBe('professional')
  })

  it('getPlanList 无参数时应以 undefined 请求', async () => {
    vi.mocked(get).mockResolvedValue({ data: [], pagination: { total: 0, page: 1, page_size: 10 } })
    await getPlanList()
    expect(get).toHaveBeenCalledWith('/admin/plans', undefined)
  })

  it('getPlanSelect 应 GET /admin/plans/select', async () => {
    vi.mocked(get).mockResolvedValue([samplePlan])
    const res = await getPlanSelect()
    expect(get).toHaveBeenCalledWith('/admin/plans/select')
    expect(res[0].id).toBe('plan-1')
  })

  it('createPlan 应 POST /admin/plans 并透传三类配额字段', async () => {
    const payload: CreatePlanParams = {
      name: '企业版', code: 'enterprise', user_limit: 500,
      storage_limit: 10737418240, api_rate_limit: 1200, wiki_space_limit: 50,
      price: 1999, status: 1,
    }
    vi.mocked(post).mockResolvedValue({ ...samplePlan, ...payload })
    await createPlan(payload)
    expect(post).toHaveBeenCalledWith('/admin/plans', payload)
  })

  it('updatePlan 应 PUT /admin/plans/{id}/update', async () => {
    vi.mocked(put).mockResolvedValue(samplePlan)
    await updatePlan('plan-1', { name: '改名', storage_limit: -1 })
    expect(put).toHaveBeenCalledWith('/admin/plans/plan-1/update', { name: '改名', storage_limit: -1 })
  })

  it('deletePlan 应 DELETE /admin/plans/{id}/delete', async () => {
    vi.mocked(del).mockResolvedValue(undefined)
    await deletePlan('plan-1')
    expect(del).toHaveBeenCalledWith('/admin/plans/plan-1/delete')
  })

  it('getTenantPlan 应 GET /admin/tenants-plan/{tenantId}', async () => {
    vi.mocked(get).mockResolvedValue(sampleCurrent)
    const res = await getTenantPlan('tn-1')
    expect(get).toHaveBeenCalledWith('/admin/tenants-plan/tn-1')
    expect(res.storage_limit).toBe(-1)
    expect(res.current_storage).toBe(1024)
  })

  it('assignTenantPlan 应 PUT 分配端点并透传 plan_id 与到期时间', async () => {
    vi.mocked(put).mockResolvedValue(undefined)
    await assignTenantPlan('tn-1', { plan_id: 'plan-1', expires_at: '2027-01-01 00:00:00' })
    expect(put).toHaveBeenCalledWith('/admin/tenants-plan/tn-1', { plan_id: 'plan-1', expires_at: '2027-01-01 00:00:00' })
  })

  it('assignTenantPlan 不传到期时间时请求体仅含 plan_id', async () => {
    vi.mocked(put).mockResolvedValue(undefined)
    await assignTenantPlan('tn-1', { plan_id: 'plan-2' })
    expect(put).toHaveBeenCalledWith('/admin/tenants-plan/tn-1', { plan_id: 'plan-2' })
  })

  it('getMyPlan 应 GET /tenant/current/plan', async () => {
    vi.mocked(get).mockResolvedValue(sampleCurrent)
    const res = await getMyPlan()
    expect(get).toHaveBeenCalledWith('/tenant/current/plan')
    expect(res.plan_code).toBe('professional')
    expect(res.current_users).toBe(12)
  })
})
