import { describe, it, expect, vi, beforeEach } from 'vitest'
import {
  getRoleList,
  getRoleDetail,
  createRole,
  updateRole,
  updateRoleStatus,
  deleteRole,
  batchDeleteRoles,
  batchUpdateRoleStatus,
  bindRolePermissions,
  getRoleListForSelect,
  getDeletedRoleList,
  restoreRole,
  permanentDeleteRole,
  batchPermanentDeleteRoles,
  type RoleItem,
  type RoleCreateParams,
} from './role'

vi.mock('@/api/request', () => ({
  get: vi.fn(),
  del: vi.fn(),
  post: vi.fn(),
  put: vi.fn(),
}))

import { get, post, put, del } from '@/api/request'

const sampleRole: RoleItem = {
  id: 'role-1',
  name: '租户管理员',
  code: 'tenant_admin',
  scope: 'tenant',
  status: 1,
}

describe('Role API - 查询', () => {
  beforeEach(() => vi.clearAllMocks())

  it('getRoleList 应以分页参数请求 /rbac/roles', async () => {
    vi.mocked(get).mockResolvedValue({ list: [sampleRole], total: 1 })
    const result = await getRoleList({ page: 1, page_size: 20, keyword: 'admin' })
    expect(get).toHaveBeenCalledWith('/rbac/roles', { page: 1, page_size: 20, keyword: 'admin' })
    expect(result.total).toBe(1)
    expect(result.list[0].code).toBe('tenant_admin')
  })

  it('getRoleDetail 应请求 /rbac/roles/{id}/detail', async () => {
    vi.mocked(get).mockResolvedValue(sampleRole)
    const result = await getRoleDetail('role-1')
    expect(get).toHaveBeenCalledWith('/rbac/roles/role-1/detail')
    expect(result.id).toBe('role-1')
  })

  it('getRoleListForSelect 应透传 scope 参数', async () => {
    vi.mocked(get).mockResolvedValue([sampleRole])
    await getRoleListForSelect('system')
    expect(get).toHaveBeenCalledWith('/rbac/roles/select', { scope: 'system' })
  })

  it('getDeletedRoleList 应请求回收站接口', async () => {
    vi.mocked(get).mockResolvedValue({ list: [sampleRole], total: 1 })
    await getDeletedRoleList({ page: 1, page_size: 10 })
    expect(get).toHaveBeenCalledWith('/rbac/roles/deleted', { page: 1, page_size: 10 })
  })
})

describe('Role API - 写操作', () => {
  beforeEach(() => vi.clearAllMocks())

  it('createRole 应 POST /rbac/roles 并携带请求体', async () => {
    const payload: RoleCreateParams = { name: '审计员', code: 'auditor', scope: 'tenant' }
    vi.mocked(post).mockResolvedValue(sampleRole)
    await createRole(payload)
    expect(post).toHaveBeenCalledWith('/rbac/roles', payload)
  })

  it('updateRole 应 PUT /rbac/roles/{id}/update', async () => {
    vi.mocked(put).mockResolvedValue(sampleRole)
    await updateRole('role-1', { name: '新名称' })
    expect(put).toHaveBeenCalledWith('/rbac/roles/role-1/update', { name: '新名称' })
  })

  it('updateRoleStatus 应以 { status } 请求状态接口', async () => {
    vi.mocked(put).mockResolvedValue(undefined)
    await updateRoleStatus('role-1', 0)
    expect(put).toHaveBeenCalledWith('/rbac/roles/role-1/status', { status: 0 })
  })

  it('deleteRole 应 DELETE /rbac/roles/{id}/delete', async () => {
    vi.mocked(del).mockResolvedValue(undefined)
    await deleteRole('role-1')
    expect(del).toHaveBeenCalledWith('/rbac/roles/role-1/delete')
  })

  it('bindRolePermissions 应以 permission_ids 覆盖式绑定', async () => {
    vi.mocked(put).mockResolvedValue(undefined)
    await bindRolePermissions('role-1', ['p-1', 'p-2'])
    expect(put).toHaveBeenCalledWith('/rbac/roles/role-1/permissions', { permission_ids: ['p-1', 'p-2'] })
  })
})

describe('Role API - 批量与回收站', () => {
  beforeEach(() => vi.clearAllMocks())

  it('batchDeleteRoles 应通过 del 的 data 传递 ids', async () => {
    vi.mocked(del).mockResolvedValue(undefined)
    await batchDeleteRoles(['a', 'b'])
    expect(del).toHaveBeenCalledWith('/rbac/roles/batch/delete', { data: { ids: ['a', 'b'] } })
  })

  it('batchUpdateRoleStatus 应 PUT 批量状态接口', async () => {
    vi.mocked(put).mockResolvedValue(undefined)
    await batchUpdateRoleStatus(['a', 'b'], 1)
    expect(put).toHaveBeenCalledWith('/rbac/roles/batch/status', { ids: ['a', 'b'], status: 1 })
  })

  it('restoreRole 应 PUT 恢复接口', async () => {
    vi.mocked(put).mockResolvedValue(undefined)
    await restoreRole('role-1')
    expect(put).toHaveBeenCalledWith('/rbac/roles/role-1/restore')
  })

  it('permanentDeleteRole 应 DELETE 永久删除接口', async () => {
    vi.mocked(del).mockResolvedValue(undefined)
    await permanentDeleteRole('role-1')
    expect(del).toHaveBeenCalledWith('/rbac/roles/role-1/permanent')
  })

  it('batchPermanentDeleteRoles 应通过 data 传递 ids', async () => {
    vi.mocked(del).mockResolvedValue(undefined)
    await batchPermanentDeleteRoles(['a', 'b'])
    expect(del).toHaveBeenCalledWith('/rbac/roles/batch/permanent', { data: { ids: ['a', 'b'] } })
  })
})
