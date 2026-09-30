import { describe, it, expect, vi, beforeEach } from 'vitest'
import {
  getInvitationList,
  createInvitation,
  cancelInvitation,
  resendInvitation,
  deleteInvitation,
  acceptInvitation,
  getInvitationInfo,
  getAssignableRoles,
  type InvitationItem,
  type CreateInvitationParams,
  type AcceptInvitationParams,
} from './invitation'

vi.mock('@/api/request', () => ({
  get: vi.fn(),
  del: vi.fn(),
  post: vi.fn(),
  put: vi.fn(),
}))

import { get, post, put, del } from '@/api/request'

const sampleInvite: InvitationItem = {
  id: 'inv-1',
  tenant_id: 't-1',
  email: 'alice@example.com',
  role_ids: ['role-1'],
  status: 'pending',
  invited_by: 'u-1',
  expires_at: '2026-10-08 00:00:00',
  created_at: '2026-10-01 00:00:00',
}

describe('Invitation API - 管理端', () => {
  beforeEach(() => vi.clearAllMocks())

  it('getInvitationList 应携带查询参数请求 /invitations', async () => {
    vi.mocked(get).mockResolvedValue({ items: [sampleInvite], total: 1 })
    const result = await getInvitationList({ page: 1, page_size: 20, status: 'pending' })
    expect(get).toHaveBeenCalledWith('/invitations', { page: 1, page_size: 20, status: 'pending' })
    expect(result.total).toBe(1)
  })

  it('createInvitation 应 POST /invitations 并携带邮箱与角色', async () => {
    const payload: CreateInvitationParams = { email: 'bob@example.com', role_ids: ['role-2'] }
    vi.mocked(post).mockResolvedValue(sampleInvite)
    await createInvitation(payload)
    expect(post).toHaveBeenCalledWith('/invitations', payload)
  })

  it('cancelInvitation 应 PUT 取消接口（无请求体）', async () => {
    vi.mocked(put).mockResolvedValue(undefined)
    await cancelInvitation('inv-1')
    expect(put).toHaveBeenCalledWith('/invitations/inv-1/cancel')
  })

  it('resendInvitation 应 PUT 重发接口', async () => {
    vi.mocked(put).mockResolvedValue(sampleInvite)
    const result = await resendInvitation('inv-1')
    expect(put).toHaveBeenCalledWith('/invitations/inv-1/resend')
    expect(result.id).toBe('inv-1')
  })

  it('deleteInvitation 应 DELETE 删除接口', async () => {
    vi.mocked(del).mockResolvedValue(undefined)
    await deleteInvitation('inv-1')
    expect(del).toHaveBeenCalledWith('/invitations/inv-1/delete')
  })
})

describe('Invitation API - 公开端', () => {
  beforeEach(() => vi.clearAllMocks())

  it('getInvitationInfo 应以 token 查询邀请信息', async () => {
    vi.mocked(get).mockResolvedValue(sampleInvite)
    const result = await getInvitationInfo('tok-abc')
    expect(get).toHaveBeenCalledWith('/invitations/info', { token: 'tok-abc' })
    expect(result.status).toBe('pending')
  })

  it('acceptInvitation 应 POST 接受邀请并注册', async () => {
    const payload: AcceptInvitationParams = {
      token: 'tok-abc',
      username: 'bob',
      password: 'P@ssw0rd!',
      nickname: 'Bob',
    }
    vi.mocked(post).mockResolvedValue({ message: 'ok' })
    await acceptInvitation(payload)
    expect(post).toHaveBeenCalledWith('/invitations/accept', payload)
  })

  it('getAssignableRoles 应 GET 可分配角色列表', async () => {
    vi.mocked(get).mockResolvedValue([{ id: 'r1', name: 'Admin', code: 'admin' }])
    const roles = await getAssignableRoles()
    expect(get).toHaveBeenCalledWith('/invitations/roles')
    expect(roles).toEqual([{ id: 'r1', name: 'Admin', code: 'admin' }])
  })
})
