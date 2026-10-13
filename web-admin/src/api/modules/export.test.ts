import { describe, it, expect, vi, beforeEach } from 'vitest'
import {
  getExportList,
  createExport,
  getExportDetail,
  downloadExport,
  deleteExport,
  type ExportJobItem,
  type CreateExportParams,
} from './export'

vi.mock('@/api/request', () => ({
  get: vi.fn(),
  post: vi.fn(),
  del: vi.fn(),
  default: { get: vi.fn() },
}))

import { get, post, del } from '@/api/request'
import request from '@/api/request'

const sampleJob: ExportJobItem = {
  id: 'job-1',
  tenant_id: 'tn-1',
  user_id: 'u-1',
  resource: 'user',
  status: 'completed',
  rows: 42,
  created_at: '2026-10-13 09:00:00',
  expires_at: '2026-10-20 09:00:00',
}

describe('Export API', () => {
  beforeEach(() => vi.clearAllMocks())

  it('getExportList 应携带查询参数请求 /exports', async () => {
    vi.mocked(get).mockResolvedValue({ list: [sampleJob], total: 1 })
    const res = await getExportList({ page: 1, page_size: 20, status: 'completed' })
    expect(get).toHaveBeenCalledWith('/exports', { page: 1, page_size: 20, status: 'completed' })
    expect(res.list[0].id).toBe('job-1')
  })

  it('getExportList 无过滤参数时应仅透传分页', async () => {
    vi.mocked(get).mockResolvedValue({ list: [], total: 0 })
    await getExportList({ page: 1, page_size: 10 })
    expect(get).toHaveBeenCalledWith('/exports', { page: 1, page_size: 10 })
  })

  it('createExport 应 POST /exports 并透传 resource 与 filters', async () => {
    const payload: CreateExportParams = { resource: 'audit_log', filters: { module: 'auth', result: 'failure' } }
    vi.mocked(post).mockResolvedValue(sampleJob)
    const res = await createExport(payload)
    expect(post).toHaveBeenCalledWith('/exports', payload)
    expect(res.status).toBe('completed')
  })

  it('getExportDetail 应 GET /exports/{id}', async () => {
    vi.mocked(get).mockResolvedValue(sampleJob)
    const res = await getExportDetail('job-1')
    expect(get).toHaveBeenCalledWith('/exports/job-1')
    expect(res.rows).toBe(42)
  })

  it('downloadExport 应以 blob 响应类型请求下载端点', async () => {
    const blob = new Blob(['a,b\n1,2\n'], { type: 'text/csv' })
    vi.mocked(request.get).mockResolvedValue(blob)
    const res = await downloadExport('job-1')
    expect(request.get).toHaveBeenCalledWith('/exports/job-1/download', { responseType: 'blob' })
    expect(res).toBe(blob)
  })

  it('deleteExport 应 DELETE /exports/{id}', async () => {
    vi.mocked(del).mockResolvedValue(undefined)
    await deleteExport('job-1')
    expect(del).toHaveBeenCalledWith('/exports/job-1')
  })
})
