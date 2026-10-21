import { request } from '@playwright/test'
import fs from 'node:fs'
import {
  AUTH_DIR,
  CREDS_FILE,
  STORAGE_STATE_FILE,
  runFixture,
  type E2ECreds
} from './helpers'

const API_URL = process.env.E2E_API_URL || 'http://127.0.0.1:8081'
const WEB_PORT = Number(process.env.E2E_WEB_PORT || 4173)
const BASE_URL = process.env.E2E_BASE_URL || `http://127.0.0.1:${WEB_PORT}`

/**
 * 全局准备：
 * 1. 检查后端 /health（后端不由 Playwright 托管）
 * 2. 创建临时超管（is_master=true），不改动任何既有账号
 * 3. 调真实登录接口换 token，写入前端 localStorage 的 storageState，供各用例复用
 */
export default async function globalSetup() {
  const api = await request.newContext({ baseURL: API_URL })
  try {
    const health = await api.get('/health', { timeout: 60_000 })
    if (!health.ok()) throw new Error(`/health 返回 ${health.status()}`)
  } catch (e) {
    throw new Error(
      `E2E 需要后端运行在 ${API_URL}：${(e as Error).message}\n` +
        '本地请先启动：go run ./cmd/server'
    )
  } finally {
    await api.dispose()
  }

  const creds = JSON.parse(runFixture(['create-admin'])) as E2ECreds
  const ctx = await request.newContext({ baseURL: API_URL })
  try {
    const res = await ctx.post('/api/v1/auth/login', {
      data: { username: creds.username, password: creds.password },
      timeout: 60_000
    })
    if (!res.ok()) throw new Error(`临时超管登录失败：HTTP ${res.status()} ${await res.text()}`)
    const body = await res.json()
    const data = body.data ?? body
    if (!data.token) throw new Error('登录响应缺少 token：' + JSON.stringify(body).slice(0, 500))

    fs.mkdirSync(AUTH_DIR, { recursive: true })
    fs.writeFileSync(CREDS_FILE, JSON.stringify(creds, null, 2), 'utf8')
    fs.writeFileSync(
      STORAGE_STATE_FILE,
      JSON.stringify(
        {
          cookies: [],
          origins: [
            {
              origin: BASE_URL,
              localStorage: [
                { name: 'meteorx_token', value: data.token },
                { name: 'meteorx_user', value: JSON.stringify(data.user ?? null) },
                { name: 'meteorx_permissions', value: JSON.stringify(data.permissions ?? []) }
              ]
            }
          ]
        },
        null,
        2
      ),
      'utf8'
    )
  } catch (e) {
    // 登录失败时不在库里留残留账号
    try {
      runFixture(['delete-admin', creds.username])
    } catch {
      /* 清理失败不掩盖原始错误 */
    }
    throw e
  } finally {
    await ctx.dispose()
  }
}
