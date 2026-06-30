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