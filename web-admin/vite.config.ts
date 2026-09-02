import { defineConfig, loadEnv } from 'vite'
import vue from '@vitejs/plugin-vue'
import path from 'node:path'

export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, process.cwd())
  return {
    resolve: {
      alias: {
        '@': path.resolve(__dirname, 'src')
      }
    },
    plugins: [vue()],
    build: {
      rollupOptions: {
        output: {
          // 第三方依赖按功能拆分 chunk，利用浏览器缓存，降低首屏加载主 chunk 体积
          manualChunks(id) {
            if (!id.includes('node_modules')) return
            if (id.includes('element-plus')) return 'vendor-element'
            if (id.includes('@element-plus')) return 'vendor-element'
            if (id.includes('axios')) return 'vendor-axios'
            if (id.includes('pinia') || id.includes('vue-router') || id.includes('vue') || id.includes('@vue')) {
              return 'vendor-vue'
            }
            return 'vendor'
          }
        }
      }
    },
    server: {
      host: '0.0.0.0',
      port: 5173,
      open: false,
      proxy: {
        '/api': {
          target: env.VITE_API_BASE_URL || 'http://localhost:8081',
          changeOrigin: true
        }
      }
    }
  }
})