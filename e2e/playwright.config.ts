import { defineConfig, devices } from '@playwright/test'

/**
 * MeteorX 浏览器 E2E 配置。
 *
 * 依赖（不托管，由调用方保证）：
 * - 后端：E2E_API_URL（默认 http://127.0.0.1:8081），globalSetup 会检查 /health
 * - MySQL / Redis：后端自身的依赖
 * - 前端：由 webServer 托管 `vite preview`（生产构建产物，更接近真实部署）
 *
 * 本机默认用系统 Chrome（设置 E2E_CHANNEL=chrome），避免额外下载浏览器；
 * CI 不设置该变量，使用 Playwright 自带 chromium。
 */
const WEB_PORT = Number(process.env.E2E_WEB_PORT || 4173)
const BASE_URL = process.env.E2E_BASE_URL || `http://127.0.0.1:${WEB_PORT}`
const CHANNEL = process.env.E2E_CHANNEL

export default defineConfig({
  testDir: './tests',
  globalSetup: './global-setup.ts',
  globalTeardown: './global-teardown.ts',
  // 用例共享同一套真实数据库，串行执行避免互相干扰
  fullyParallel: false,
  workers: 1,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 1 : 0,
  timeout: 90_000,
  expect: { timeout: 15_000 },
  reporter: process.env.CI ? [['list'], ['html', { open: 'never' }]] : [['list']],
  use: {
    baseURL: BASE_URL,
    trace: 'retain-on-failure',
    screenshot: 'only-on-failure',
    video: 'off'
  },
  projects: [
    {
      name: 'chromium',
      use: {
        ...devices['Desktop Chrome'],
        ...(CHANNEL ? { channel: CHANNEL } : {})
      }
    }
  ],
  webServer: {
    command: `npm run preview -- --host 127.0.0.1 --port ${WEB_PORT} --strictPort`,
    cwd: '../web-admin',
    url: BASE_URL,
    reuseExistingServer: !process.env.CI,
    timeout: 180_000
  }
})
