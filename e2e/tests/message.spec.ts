import { test, expect } from '@playwright/test'
import { STORAGE_STATE_FILE } from '../helpers'

// 复用 globalSetup 换到的真实登录态（localStorage: meteorx_token / meteorx_user / meteorx_permissions）
test.use({ storageState: STORAGE_STATE_FILE })

// 与 web-admin/src/utils/messageCategory.ts 保持同一键名
const CATEGORY_KEY = 'meteorx_message_category'

const tabItems = (page: import('@playwright/test').Page) =>
  page.locator('.category-tabs .el-tabs__item')
const activeTab = (page: import('@playwright/test').Page) =>
  page.locator('.category-tabs .el-tabs__item.is-active')

test.describe('消息中心', () => {
  test('分类页签跨会话记住，刷新后仍按该分类查询 @smoke', async ({ page }) => {
    await page.goto('/#/messages')
    await expect(tabItems(page).first()).toBeVisible({ timeout: 30_000 })

    // 从确定的「全部」起跑：本地存储可能带着上一轮的记忆
    await page.evaluate(key => localStorage.removeItem(key), CATEGORY_KEY)
    await page.reload()
    await expect(activeTab(page)).toContainText('全部')

    // 切到「任务」：请求必须带 category=task，否则只是页签高亮变了、列表仍是全部
    const switchResp = page.waitForResponse(
      r => r.url().includes('/api/v1/messages') && r.url().includes('category=task'),
      { timeout: 30_000 }
    )
    await tabItems(page).filter({ hasText: '任务' }).click()
    expect((await switchResp).ok()).toBe(true)
    expect(await page.evaluate(key => localStorage.getItem(key), CATEGORY_KEY)).toBe('task')

    // 刷新：页签仍停在「任务」，重新发出的列表请求同样带 category=task
    const reloadResp = page.waitForResponse(
      r => r.url().includes('/api/v1/messages') && r.url().includes('category=task'),
      { timeout: 30_000 }
    )
    await page.reload()
    expect((await reloadResp).ok()).toBe(true)
    await expect(activeTab(page)).toContainText('任务')
    expect(await page.evaluate(key => localStorage.getItem(key), CATEGORY_KEY)).toBe('task')
  })

  test('清除记忆后回到「全部」，与铃铛「查看全部消息」同一口径 @smoke', async ({ page }) => {
    await page.goto('/#/messages')
    await expect(tabItems(page).first()).toBeVisible({ timeout: 30_000 })

    await page.evaluate(key => localStorage.setItem(key, 'audit'), CATEGORY_KEY)
    await page.reload()
    await expect(activeTab(page)).toContainText('告警')

    // 铃铛里的「查看全部消息」做的就是这一步：清键后落地，看到的是全部而不是被筛过的列表
    await page.evaluate(key => localStorage.removeItem(key), CATEGORY_KEY)
    await page.reload()
    await expect(activeTab(page)).toContainText('全部')
  })
})
