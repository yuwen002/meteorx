import { defineConfig, loadEnv } from 'vite'
import vue from '@vitejs/plugin-vue'
import AutoImport from 'unplugin-auto-import/vite'
import Components from 'unplugin-vue-components/vite'
import path from 'node:path'

/**
 * Element Plus 按需 resolver（组件级深路径引入）。
 *
 * 说明：官方 ElementPlusResolver 从 `element-plus/es`（index barrel）按名导入组件。
 * ES barrel 汇总引用了全部组件模块，rollup 无法对其进行 tree-shake，
 * 导致组件 JS 被整包打入产物。这里改为从
 * `element-plus/es/components/<kebab-name>` 深路径逐组件引入，
 * 只有页面实际用到某组件时才打包对应实现，样式同样按组件加载。
 */
function toKebabCase(name: string) {
  return name.replace(/([a-z0-9])([A-Z])/g, '$1-$2').toLowerCase()
}

function epSideEffects(dir: string) {
  return [
    'element-plus/es/components/base/style/css',
    `element-plus/es/components/${dir}/style/css`
  ]
}

// 无独立 index.mjs、从父组件目录聚合导出的子组件（ElXxx -> 实际模块目录）
const AGGREGATED_MODULES: Record<string, string> = {
  ElTableColumn: 'table',
  ElDropdownItem: 'dropdown',
  ElDropdownMenu: 'dropdown',
  ElOption: 'select',
  ElOptionGroup: 'select',
  ElTabPane: 'tabs',
  ElMenuItem: 'menu',
  ElMenuItemGroup: 'menu',
  ElSubMenu: 'menu',
  ElDescriptionsItem: 'descriptions',
  ElCheckboxGroup: 'checkbox',
  ElRadioGroup: 'radio',
  ElRadioButton: 'radio',
  ElTimelineItem: 'timeline',
  ElButtonGroup: 'button',
  ElAside: 'container',
  ElFooter: 'container',
  ElHeader: 'container',
  ElMain: 'container',
  ElFormItem: 'form',
  ElBreadcrumbItem: 'breadcrumb'
}

const elementPlusResolvers = [
  {
    type: 'component',
    resolve(name: string) {
      if (!/^El[A-Z]/.test(name) || name === 'ElAutoResizer') return
      // 样式目录始终用组件自身 kebab 名；JS 模块从聚合父目录导出
      const styleDir = toKebabCase(name.slice(2))
      const module = AGGREGATED_MODULES[name] ?? styleDir
      return {
        name,
        // 显式带 /index 以命中 element-plus 的 exports 映射
        from: `element-plus/es/components/${module}/index`,
        sideEffects: epSideEffects(styleDir)
      }
    }
  },
  {
    type: 'directive',
    resolve(name: string) {
      const map: Record<string, { importName: string; dir: string }> = {
        Loading: { importName: 'ElLoadingDirective', dir: 'loading' },
        Popover: { importName: 'ElPopoverDirective', dir: 'popover' },
        InfiniteScroll: { importName: 'ElInfiniteScroll', dir: 'infinite-scroll' }
      }
      const item = map[name]
      if (!item) return
      return {
        name: item.importName,
        from: `element-plus/es/components/${item.dir}/index`,
        sideEffects: epSideEffects(item.dir)
      }
    }
  }
]

export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, process.cwd())
  return {
    resolve: {
      alias: {
        '@': path.resolve(__dirname, 'src')
      }
    },
    plugins: [
      vue(),
      // 常用 API 自动导入：ref / computed / useRouter 等免手动 import（生成 src/auto-imports.d.ts）
      AutoImport({
        imports: ['vue', 'vue-router', 'pinia'],
        dts: 'src/auto-imports.d.ts'
      }),
      // Element Plus 组件与 v-loading 等指令按需自动引入，样式随组件按需加载。
      // 注意：不生成 components.d.ts，保持模板组件类型与全量时代一致（宽松），
      // 避免 el-table 等组件 slot 泛型让既有代码（如 row 透传）产生类型回归。
      Components({
        resolvers: elementPlusResolvers,
        dts: false
      })
    ],
    build: {
      // Element Plus 组件覆盖较广，按需后 vendor 仍有约 500KB，调高阈值避免误报
      chunkSizeWarningLimit: 550,
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