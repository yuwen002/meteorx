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
  /** true 表示库里原本没有租户、由夹具现建，用例结束要删掉 */
  created: boolean
}

/**
 * 取一个可用租户；库里没有时夹具会现建一条。
 * CI 的 e2e 库由 AutoMigrate + Seed 建表，而 seed.sql 不写 tenants，
 * 不能假设环境里有存量租户。
 */
export function ensureTenant(): E2ETenantRef {
  return JSON.parse(runFixture(['ensure-tenant'])) as E2ETenantRef
}

export function deleteTenant(id: string): void {
  runFixture(['delete-tenant', id])
}

export interface E2EAlertRef {
  id: string
  /** 夹具生成：audit_alerts.user_id 列宽 26，调用方拼出来的长串会被数据库拒绝 */
  user_id: string
}

/** 造一条审计告警，供「告警详情 → 触发用户 → 时间线」用例定位。 */
export function seedAlert(username: string): E2EAlertRef {
  return JSON.parse(runFixture(['seed-alert', username])) as E2EAlertRef
}

export function deleteAlert(id: string): void {
  runFixture(['delete-alert', id])
}
