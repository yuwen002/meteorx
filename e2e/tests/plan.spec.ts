import { test, expect } from '@playwright/test'
import { STORAGE_STATE_FILE, deletePlan, verifyPlan } from '../helpers'

// 复用 globalSetup 换到的真实登录态（localStorage: meteorx_token / meteorx_user / meteorx_permissions）
test.use({ storageState: STORAGE_STATE_FILE })

test.describe('套餐管理', () => {
  test('新增套餐写入真实数据库并在列表可见 @smoke', async ({ page }) => {
    const ts = Date.now()
    const name = `E2E套餐-${ts}`
    const code = `e2e-${ts}`

    try {
      await page.goto('/#/system/plan')
      const addBtn = page.locator('button:has-text("新增套餐")')
      await expect(addBtn).toBeVisible({ timeout: 30_000 })
      await addBtn.click()

      const dialog = page.locator('.el-dialog:visible')
      await expect(dialog).toBeVisible()
      await dialog.locator('input').nth(0).fill(name)
      await dialog.locator('input').nth(1).fill(code)

      const createResp = page.waitForResponse(
        (r) => r.url().includes('/api/v1/admin/plans') && r.request().method() === 'POST',
        { timeout: 30_000 }
      )
      await dialog.locator('.el-dialog__footer button:has-text("确定")').click()
      const resp = await createResp
      expect(resp.status()).toBe(200)

      // UI 断言：列表出现新套餐
      await expect(
        page.locator('.el-table__body').getByText(name, { exact: false })
      ).toBeVisible({ timeout: 20_000 })

      // 绕过后端直查 MySQL：证明写入的是真实数据库而非前端假象
      const row = verifyPlan(code)
      expect(row.found).toBe(true)
      expect(row.code).toBe(code)
      expect(row.status).toBe(1)
      // 空配额由写路径归一为 -1（不限）
      expect(row.user_limit).toBe(-1)
    } finally {
      deletePlan(code)
    }
  })
})
