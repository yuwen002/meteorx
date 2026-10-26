import { describe, it, expect, vi, beforeEach } from 'vitest'
import { importUsers, type ImportUsersResult } from './user'

vi.mock('@/api/request', () => ({
  get: vi.fn(),
  post: vi.fn(),
  put: vi.fn(),
  del: vi.fn()
}))

import { post } from '@/api/request'

const sampleResult: ImportUsersResult = {
  total: 2,
  created: 2,
  users: [
    { row: 2, user_id: 'u1', username: 'alice', nickname: 'Alice', email: 'a@example.com', role_codes: ['member'], initial_password: 'Xy12!abcdEFG' }
  ],
  warnings: ['第 3 行未填写角色，已使用所选默认角色']
}

// makeFile 构造一个最小可用的上传文件对象
function makeFile(name = 'users.csv', content = 'username,nickname\nalice,Alice\n') {
  return new File([content], name, { type: 'text/csv' })
}

describe('User Import API', () => {
  beforeEach(() => vi.clearAllMocks())

  it('importUsers 应把文件与默认角色提交到租户端点', async () => {
    vi.mocked(post).mockResolvedValue(sampleResult)
    const file = makeFile()

    const res = await importUsers({ file, roleIds: ['r1', 'r2'] })

    expect(post).toHaveBeenCalledTimes(1)
    const [url, body, config] = vi.mocked(post).mock.calls[0]
    expect(url).toBe('/users/import')
    expect(config).toEqual({ timeout: 120000 })

    const form = body as FormData
    expect(form.get('file')).toBeInstanceOf(File)
    expect((form.get('file') as File).name).toBe('users.csv')
    expect(form.get('role_ids')).toBe('r1,r2')
    expect(form.get('tenant_id')).toBeNull()
    expect(res.created).toBe(2)
    expect(res.users[0].initial_password).toBe('Xy12!abcdEFG')
  })

  it('importUsers 未选默认角色时不提交 role_ids 字段', async () => {
    vi.mocked(post).mockResolvedValue(sampleResult)

    await importUsers({ file: makeFile() })

    const form = vi.mocked(post).mock.calls[0][1] as FormData
    expect(form.get('role_ids')).toBeNull()
  })

  it('importUsers 传入 tenantId 时应走管理端端点并带上 tenant_id', async () => {
    vi.mocked(post).mockResolvedValue(sampleResult)

    await importUsers({ file: makeFile('users.xlsx'), tenantId: 'tn-9' })

    const [url, body] = vi.mocked(post).mock.calls[0]
    expect(url).toBe('/admin/tenant-users/import')
    const form = body as FormData
    expect(form.get('tenant_id')).toBe('tn-9')
    expect(form.get('role_ids')).toBeNull()
  })

  it('importUsers 应透传后端返回的逐行告警', async () => {
    vi.mocked(post).mockResolvedValue(sampleResult)

    const res = await importUsers({ file: makeFile() })

    expect(res.warnings?.[0]).toContain('默认角色')
  })
})
