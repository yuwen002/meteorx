import DOMPurify from 'dompurify'

/**
 * 对服务端返回的 HTML 片段做前端二次消毒（纵深防御）。
 *
 * 后端 Markdown 渲染已「转义优先 + 正则清洗」，但正则消毒存在被畸形编码绕过的理论风险，
 * 这里在用 v-html 注入 DOM 前再用 DOMPurify 拦一道，剥离脚本、事件处理器、危险协议等，
 * 规避共享页 / 文档正文等富文本渲染场景下的 XSS。空值安全，返回纯字符串。
 */
export function sanitizeHtml(html?: string | null): string {
  if (!html) return ''
  return DOMPurify.sanitize(html) as string
}
