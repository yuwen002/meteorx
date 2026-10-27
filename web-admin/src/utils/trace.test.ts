import { describe, it, expect } from 'vitest'
import { buildTraceUrl } from './trace'

describe('buildTraceUrl', () => {
  it('未配置 Jaeger base 时返回空串（回退纯文本）', () => {
    import.meta.env.VITE_JAEGER_BASE_URL = ''
    expect(buildTraceUrl('a0e95349f05edac1')).toBe('')
  })

  it('无 trace_id 时返回空串', () => {
    import.meta.env.VITE_JAEGER_BASE_URL = 'http://localhost:16686'
    expect(buildTraceUrl('')).toBe('')
    expect(buildTraceUrl(undefined)).toBe('')
  })

  it('拼接 Jaeger trace 页 URL', () => {
    import.meta.env.VITE_JAEGER_BASE_URL = 'http://localhost:16686'
    expect(buildTraceUrl('a0e95349f05edac1')).toBe('http://localhost:16686/trace/a0e95349f05edac1')
  })

  it('容忍 base 结尾斜杠与首尾空白', () => {
    import.meta.env.VITE_JAEGER_BASE_URL = '  http://jaeger.local:16686/  '
    expect(buildTraceUrl('deadbeef')).toBe('http://jaeger.local:16686/trace/deadbeef')
  })
})
