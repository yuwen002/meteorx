import js from '@eslint/js'
import pluginVue from 'eslint-plugin-vue'
import globals from 'globals'
import prettier from 'eslint-config-prettier'
import tseslint from 'typescript-eslint'

export default [
  {
    ignores: ['dist/**', 'node_modules/**', 'auto-imports.d.ts', 'components.d.ts'],
  },

  js.configs.recommended,
  ...tseslint.configs.recommended,
  ...pluginVue.configs['flat/recommended'],

  {
    files: ['**/*.{ts,vue}'],
    languageOptions: {
      globals: globals.browser,
      parserOptions: {
        parser: tseslint.parser,
        extraFileExtensions: ['.vue'],
      },
    },
    rules: {
      // TypeScript 负责未使用变量检查（TS 编译时同样生效）
      'no-unused-vars': 'off',
      '@typescript-eslint/no-unused-vars': ['error', {
        argsIgnorePattern: '^_',
        varsIgnorePattern: '^_',
        // 兼容 catch (e) 惯用法：仅打印提示时无需使用异常对象
        caughtErrors: 'none',
      }],
      // 避免字面量全局（由 TS lib / globals 提供）
      'no-undef': 'off',
      // 全仓单字组件命名（如 index.vue、layout.vue）是约定用法
      'vue/multi-word-component-names': 'off',
      // 显式使用 any 时警告（保留类型信息以便后续治理）
      '@typescript-eslint/no-explicit-any': 'warn',
    },
  },

  // 与 Prettier 冲突的格式规则交给 Prettier 统一
  prettier,
]
