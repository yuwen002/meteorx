import { describe, it, expect } from 'vitest'
import { sanitizeHtml } from './sanitize'

describe('sanitizeHtml', () => {
  it('空值返回空字符串', () => {
    expect(sanitizeHtml()).toBe('')
    expect(sanitizeHtml(null)).toBe('')
    expect(sanitizeHtml('')).toBe('')
  })

  it('保留安全的普通标签', () => {
    const input = '<p>标题：<strong>粗体</strong>与<mark>高亮</mark></p>'
    expect(sanitizeHtml(input)).toBe(input)
  })

  it('移除 script 标签及其内容', () => {
    const out = sanitizeHtml('<p>hi</p><script>alert(1)</script>')
    expect(out).not.toContain('<script')
    expect(out).not.toContain('alert(1)')
    expect(out).toContain('<p>hi</p>')
  })

  it('移除内联事件处理器', () => {
    const out = sanitizeHtml('<img src="x.png" onerror="alert(1)">')
    expect(out).not.toContain('onerror')
    expect(out).not.toContain('alert(1)')
  })

  it('移除 javascript: 协议的链接', () => {
    const out = sanitizeHtml('<a href="javascript:alert(1)">点我</a>')
    expect(out).not.toContain('javascript:')
  })

  it('保留正常的 https 链接', () => {
    const out = sanitizeHtml('<a href="https://example.com">官网</a>')
    expect(out).toContain('href="https://example.com"')
    expect(out).toContain('官网')
  })
})
