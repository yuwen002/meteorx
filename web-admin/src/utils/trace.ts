// 链路追踪跳链：把审计记录里的 trace_id 变成可点击的 Jaeger 链路查询链接。
// 仅在配置了 VITE_JAEGER_BASE_URL 时生成；未配置或无 trace_id 时返回空串，
// 调用方据此回退为纯文本，避免渲染出指向无效站点的死链。

// buildTraceUrl 返回该 trace id 对应的 Jaeger 链路页 URL；不可用时返回空串。
export function buildTraceUrl(traceId?: string): string {
  const base = (import.meta.env.VITE_JAEGER_BASE_URL || '').trim().replace(/\/+$/, '')
  if (!base || !traceId) return ''
  return `${base}/trace/${traceId}`
}
