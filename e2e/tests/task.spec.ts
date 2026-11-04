import { test, expect } from '@playwright/test'
import {
  STORAGE_STATE_FILE,
  readCreds,
  seedTaskTree,
  seedMultiAssignee,
  cleanupTasks,
  cleanupUser
} from '../helpers'

// 复用 globalSetup 换到的真实登录态（localStorage: meteorx_token / meteorx_user / meteorx_permissions）
test.use({ storageState: STORAGE_STATE_FILE })

// 临时超管所在租户（与 e2e/fixtures create-admin 写入的 tenant_id 保持一致）
const ADMIN_TENANT = 'SYSTEM_ROOT'
// 后端 API 基址：globalSetup / helpers 约定，直连数据层校验时复用
const API_URL = process.env.E2E_API_URL || 'http://127.0.0.1:8081'

// 列表后端信封为 { code, message, data: { data: [...], pagination } }：任务数组在 body.data.data
const tasksFromBody = (body: any): any[] => body?.data?.data ?? []

/**
 * 进入任务页并切到指定视图：先落地（goto 触发鉴权与首屏），写 localStorage 视图偏好后 reload，
 * 使组件按偏好初始化（viewMode / groupBy 均在挂载时从 localStorage 读取）。
 * waitFor 由调用方在 reload 前注册，避免竞态漏接首个响应。
 */
async function enterTaskPage(page: import('@playwright/test').Page, viewMode: 'list' | 'board') {
  await page.goto('/#/task')
  await page.evaluate((vm) => localStorage.setItem('meteorx.task.viewMode', vm), viewMode)
}

test.describe('任务模块（看板分组 / 子任务 / 多负责人）', () => {
  test('看板走全量 board 端点，切换分组维度渲染对应列数（状态3列→优先级固定4列）@smoke', async ({ page }) => {
    await page.goto('/#/task')
    await page.evaluate(() => {
      localStorage.setItem('meteorx.task.viewMode', 'board')
      localStorage.setItem('meteorx.task.groupBy', 'status')
    })
    // 看板全量加载应命中独立的 /tasks/board 端点（非分页的 /tasks），且默认仅含根任务
    const boardResp = page.waitForResponse((r) => r.url().includes('/api/v1/tasks/board'), { timeout: 30_000 })
    await page.reload()
    const resp = await boardResp
    expect(resp.ok()).toBe(true)
    expect(resp.url()).toContain('root_only=true')

    // 按状态分组：固定三列
    await expect(page.locator('.board-col')).toHaveCount(3, { timeout: 20_000 })

    // 切到“按优先级”：priority 维度固定四列（即便无任务也渲染 urgent/high/normal/low）
    // Element Plus 2.14 单选 el-select 不再渲染 placeholder 输入框，选中态显示“按状态”，
    // 故按结构定位看板工具栏的第一个下拉（即分组维度）触发器点击展开，再按文案选项。
    await page.locator('.filter-bar .el-select').first().click()
    await page.locator('.el-select-dropdown__item:visible').filter({ hasText: '按优先级' }).click()
    await expect(page.locator('.board-col')).toHaveCount(4)
    // 分组偏好持久化
    expect(await page.evaluate(() => localStorage.getItem('meteorx.task.groupBy'))).toBe('priority')
  })

  test('列表行首展开根任务，懒加载其直接子任务 @smoke', async ({ page }) => {
    const creds = readCreds()
    const tree = seedTaskTree(creds.id, ADMIN_TENANT)
    try {
      await enterTaskPage(page, 'list')
      await page.reload()
      // 用关键词精确命中刚种下的根任务，规避其它存量数据干扰行定位
      await page.getByPlaceholder('搜索标题/描述').fill(tree.root_title)
      const listResp = page.waitForResponse(
        (r) => r.url().includes('/api/v1/tasks?') && r.url().includes('keyword='),
        { timeout: 30_000 }
      )
      await page.getByRole('button', { name: '查询' }).click()
      await listResp

      const row = page.locator('.el-table__row').filter({ hasText: tree.root_title }).first()
      await expect(row).toBeVisible({ timeout: 20_000 })

      // 展开：应发起带 parent_id=<root> 的子任务懒加载请求，而非一次性全量下发
      const childResp = page.waitForResponse(
        (r) => r.url().includes('/api/v1/tasks?') && r.url().includes(`parent_id=${tree.root_id}`),
        { timeout: 20_000 }
      )
      await row.locator('.el-table__expand-icon').click()
      const resp = await childResp
      expect(resp.ok()).toBe(true)
      expect(tasksFromBody(await resp.json()).length).toBe(2)

      // 展开面板渲染两条子任务行，各带“子任务”标签
      await expect(page.locator('.subtask-row')).toHaveCount(2, { timeout: 10_000 })
    } finally {
      cleanupTasks([tree.root_id, ...tree.child_ids])
    }
  })

  test('双负责人租户任务列表显示主负责人 +1，且副负责人可按负责人筛选命中 @smoke', async ({ page, request }) => {
    const creds = readCreds()
    const seed = seedMultiAssignee(creds.id, ADMIN_TENANT)
    try {
      await enterTaskPage(page, 'list')
      await page.reload()
      await page.getByPlaceholder('搜索标题/描述').fill(seed.task_title)
      const listResp = page.waitForResponse(
        (r) => r.url().includes('/api/v1/tasks?') && r.url().includes('keyword='),
        { timeout: 30_000 }
      )
      await page.getByRole('button', { name: '查询' }).click()
      await listResp

      const row = page.locator('.el-table__row').filter({ hasText: seed.task_title }).first()
      await expect(row).toBeVisible({ timeout: 20_000 })
      // 负责人列：主负责人名 + 折叠的“+1”（除主负责人外的额外负责人数）
      await expect(row.locator('.assignee-more')).toHaveText('+1')

      // 数据层：按“副负责人”筛选应命中该任务（applyListFilters 经 task_assignees 子查询匹配）
      const token = await page.evaluate(() => localStorage.getItem('meteorx_token'))
      const resp = await request.get(`${API_URL}/api/v1/tasks`, {
        params: { assignee_id: seed.member_id, page: '1', page_size: '50' },
        headers: { Authorization: `Bearer ${token}` }
      })
      expect(resp.ok()).toBe(true)
      const ids: string[] = tasksFromBody(await resp.json()).map((t) => t.id)
      expect(ids).toContain(seed.task_id)
    } finally {
      cleanupTasks([seed.task_id])
      cleanupUser(seed.member_id)
    }
  })
})
