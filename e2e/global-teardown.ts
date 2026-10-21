import fs from 'node:fs'
import { AUTH_DIR, CREDS_FILE, runFixture, type E2ECreds } from './helpers'

/**
 * 全局清理：删除 globalSetup 创建的临时超管，并清掉本地凭据文件。
 * 测试失败时同样执行，避免把测试账号留在真实库里。
 */
export default async function globalTeardown() {
  if (fs.existsSync(CREDS_FILE)) {
    try {
      const creds = JSON.parse(fs.readFileSync(CREDS_FILE, 'utf8')) as E2ECreds
      runFixture(['delete-admin', creds.username])
    } catch (e) {
      console.warn('[e2e] 清理临时超管失败：' + (e as Error).message)
    }
  }
  fs.rmSync(AUTH_DIR, { recursive: true, force: true })
}
