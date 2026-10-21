import { test, expect } from '@playwright/test'
import { readCreds } from '../helpers'

test.describe('认证', () => {
  test('未登录访问后台重定向到登录页 @smoke', async ({ page }) => {
    await page.goto('/#/dashboard')
    await expect(page).toHaveURL(/#\/login/, { timeout: 30_000 })
    await expect(page.locator('input[placeholder="请输入用户名"]')).toBeVisible({ timeout: 45_000 })
  })

  test('密码错误时给出提示且停留在登录页 @smoke', async ({ page }) => {
    const { username } = readCreds()
    await page.goto('/#/login')
    await page.locator('input[placeholder="请输入用户名"]').fill(username)
    await page.locator('input[type="password"]').fill('E2e!WrongPass')
    await page.locator('button:has-text("登 录")').click()
    await expect(page.locator('.login-error')).toBeVisible({ timeout: 20_000 })
    await expect(page).toHaveURL(/#\/login/)
  })

  test('临时超管登录进入工作台 @smoke', async ({ page }) => {
    const { username, password } = readCreds()
    await page.goto('/#/login')
    await page.locator('input[placeholder="请输入用户名"]').fill(username)
    await page.locator('input[type="password"]').fill(password)
    await page.locator('button:has-text("登 录")').click()
    await expect(page).toHaveURL(/#\/dashboard/, { timeout: 45_000 })
    await expect(page.locator('.el-menu').first()).toBeVisible({ timeout: 30_000 })
  })
})
