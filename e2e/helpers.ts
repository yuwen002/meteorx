import { execFileSync } from 'node:child_process'
import fs from 'node:fs'
import path from 'node:path'

/** 仓库根目录（e2e 的上一级）。 */
export const REPO_ROOT = path.resolve(__dirname, '..')
/** 运行期凭据目录（已 gitignore，测试结束即删除）。 */
export const AUTH_DIR = path.resolve(__dirname, '.auth')
export const CREDS_FILE = path.join(AUTH_DIR, 'creds.json')
export const STORAGE_STATE_FILE = path.join(AUTH_DIR, 'admin-state.json')

export interface E2ECreds {
  username: string
  password: string
  id: string
}

export interface E2EPlanRow {
  found: boolean
  id?: string
  name?: string
  code?: string
  user_limit?: number
  storage_limit?: number
  price?: number
  status?: number
  created_at?: string
}

/**
 * 调用 Go 夹具并解析结果。
 * 夹具用 `E2E_JSON:` 前缀输出单行 JSON，避免 GORM 日志混入 stdout 干扰解析。
 */
export function runFixture(args: string[]): string {
  const out = execFileSync('go', ['run', './e2e/fixtures/fixture', ...args], {
    cwd: REPO_ROOT,
    encoding: 'utf8',
    timeout: 180_000
  })
  const line = out
    .split(/\r?\n/)
    .filter((l) => l.includes('E2E_JSON:'))
    .pop()
  if (!line) throw new Error(`fixture ${args.join(' ')} 未输出结果：\n${out}`)
  return line.slice(line.indexOf('E2E_JSON:') + 'E2E_JSON:'.length).trim()
}

export function readCreds(): E2ECreds {
  if (!fs.existsSync(CREDS_FILE)) {
    throw new Error('缺少 .auth/creds.json：请通过 npx playwright test 运行（globalSetup 会自动创建临时超管）')
  }
  return JSON.parse(fs.readFileSync(CREDS_FILE, 'utf8')) as E2ECreds
}

export function verifyPlan(code: string): E2EPlanRow {
  return JSON.parse(runFixture(['verify-plan', code])) as E2EPlanRow
}

export function deletePlan(code: string): void {
  runFixture(['delete-plan', code])
}

export interface E2ETenantRef {
  id: string
  name: string
}

/** 取一个未删除的租户：站内信 ?tenant_id= 深链用例只验证「按 ID 定位」，不关心是哪一家。 */
export function firstTenant(): E2ETenantRef {
  return JSON.parse(runFixture(['list-tenants'])) as E2ETenantRef
}
