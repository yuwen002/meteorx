import { test, expect } from '@playwright/test'
import {
  STORAGE_STATE_FILE,
  deleteAlert,
  deleteTenant,
  ensureTenant,
  readCreds,
  seedAlert
} from '../helpers'

// 复用 globalSetup 换到的真实登录态
test.use({ storageState: STORAGE_STATE_FILE })

/**
 * 站内信深链：三类消息都用「业务 ID 进 query + 目标页按 ID 取详情」定位。
 * 这里从落地页一侧验证参数被真正消费，而不是只改地址栏。
 */
test.describe('站内信深链', () => {
  test('告警详情可定位到触发用户的时间线 @smoke', async ({ page }) => {
    const { username } = readCreds()
    // 刻意不用登录用户本人：那样「读了 query」与「回退默认值」结果相同，用例会失去区分力
    const probe = `${username}-probe`

    const timelineResp = page.waitForResponse(
      r => r.url().includes('/audit/user-timeline') && r.url().includes(`user_id=${probe}`),
      { timeout: 30_000 }
    )
    await page.goto(`/#/system/audit/timeline?user_id=${encodeURIComponent(probe)}`)
    expect((await timelineResp).ok()).toBe(true)

    // 查询框回填：用户能一眼看出当前看的是谁的时间线
    await expect(page.locator('input[placeholder="输入用户ID或用户名"]')).toHaveValue(probe)

    // 没有 query 时仍按登录用户查询（原有默认行为）
    const fallback = page.waitForResponse(
      r => r.url().includes('/audit/user-timeline') && r.url().includes(`user_id=${username}`),
      { timeout: 30_000 }
    )
    await page.goto('/#/system/audit/timeline')
    expect((await fallback).ok()).toBe(true)
    await expect(page.locator('input[placeholder="输入用户ID或用户名"]')).toHaveValue(username)
  })

  test('订阅临期可定位到该租户的套餐配置 @smoke', async ({ page }) => {
    const tenant = ensureTenant()
    expect(tenant.id, 'ensure-tenant 应返回一个可用租户').toBeTruthy()

    try {
      const detailResp = page.waitForResponse(
        r => r.url().includes(`/admin/tenants/${tenant.id}/detail`),
        { timeout: 30_000 }
      )
      const planResp = page.waitForResponse(
        r => r.url().includes(`/admin/tenants-plan/${tenant.id}`),
        { timeout: 30_000 }
      )
      await page.goto(`/#/system/tenant?tenant_id=${encodeURIComponent(tenant.id)}`)
      expect((await detailResp).ok()).toBe(true)
      expect((await planResp).ok()).toBe(true)

      // 不依赖该租户出现在当前页：按 ID 取详情后直接开弹窗
      const dialog = page.locator('.el-dialog:visible')
      await expect(dialog).toBeVisible({ timeout: 20_000 })
      await expect(dialog.locator('.el-dialog__title')).toContainText(`套餐配置 - ${tenant.name}`)
    } finally {
      // 只删夹具现建的那条，不动环境里的存量租户
      if (tenant.created) deleteTenant(tenant.id)
    }
  })

  test('告警详情点「触发用户」直达该用户的时间线 @smoke', async ({ page }) => {
    const { username } = readCreds()
    const probeName = `${username}-alert`
    // user_id 用夹具生成（列宽 26），同时验证详情页优先用 ID 而不是退化成用户名
    const alert = seedAlert(probeName)

    try {
      const detailResp = page.waitForResponse(
        r => r.url().includes(`/audit/alerts/${alert.id}`),
        { timeout: 30_000 }
      )
      await page.goto(`/#/system/audit/alert?alert_id=${encodeURIComponent(alert.id)}`)
      expect((await detailResp).ok()).toBe(true)

      const dialog = page.locator('.el-dialog:visible')
      await expect(dialog).toBeVisible({ timeout: 20_000 })

      // 「触发用户」必须是一个真入口，且优先用 user_id（而非退化成用户名文本）
      const userLink = dialog.getByRole('button', { name: probeName })
      await expect(userLink).toBeVisible()

      const timelineResp = page.waitForResponse(
        r => r.url().includes('/audit/user-timeline') && r.url().includes(`user_id=${alert.user_id}`),
        { timeout: 30_000 }
      )
      await userLink.click()
      expect((await timelineResp).ok()).toBe(true)
      await expect(page).toHaveURL(new RegExp(`user_id=${alert.user_id}`))
      await expect(page.locator('input[placeholder="输入用户ID或用户名"]')).toHaveValue(alert.user_id)
    } finally {
      deleteAlert(alert.id)
    }
  })
})
