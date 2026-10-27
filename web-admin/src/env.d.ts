/// <reference types="vite/client" />

interface ImportMetaEnv {
  VITE_API_BASE_URL: string
  // Jaeger UI 基址（如 http://localhost:16686）；留空则审计详情的 trace_id 不渲染为跳链
  VITE_JAEGER_BASE_URL: string
}

declare module '*.vue' {
  import type { DefineComponent } from 'vue'
  const component: DefineComponent<Record<string, never>, Record<string, never>, unknown>
  export default component
}