import { defineConfig } from 'vitest/config'
import vue from '@vitejs/plugin-vue'
import { resolve } from 'path'

export default defineConfig({
  plugins: [vue()],
  resolve: {
    alias: {
      '@': resolve(__dirname, 'src'),
    },
  },
  test: {
    environment: 'jsdom',
    globals: true,
    include: ['src/**/*.{test,spec}.{ts,tsx}'],
    coverage: {
      provider: 'v8',
      // 阶梯式门槛：仅统计已有测试基础设施的 API/工具/组合式层（视图组件待引入组件测试后逐步纳入）；
      // 阈值锁定在 2026-10 基线（86.24/97.84/73.21/86.24）附近防回退，
      // 后续补齐 api/auth.ts、audit.ts、role.ts 存量测试后应逐步上调至 90+
      include: ['src/api/**', 'src/utils/**', 'src/composables/**'],
      exclude: ['src/**/*.{test,spec}.{ts,tsx}'],
      // 仅统计被测试触达的文件：未被任何测试引用的模块计 0% 会淹没门槛信号；
      // 新模块接入测试后自动纳入门槛约束
      all: false,
      thresholds: {
        lines: 85,
        functions: 70,
        branches: 95,
        statements: 85,
      },
    },
  },
})