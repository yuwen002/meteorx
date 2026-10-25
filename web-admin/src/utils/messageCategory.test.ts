import { describe, it, expect, beforeEach } from 'vitest'
import {
  MESSAGE_CATEGORY_KEY,
  readMessageCategory,
  saveMessageCategory,
  clearMessageCategory
} from './messageCategory'

const VALID = ['', 'system', 'announcement', 'task', 'plan', 'audit'] as const

describe('messageCategory', () => {
  beforeEach(() => {
    localStorage.clear()
  })

  it('没有存值时回落到「全部」', () => {
    expect(readMessageCategory(VALID)).toBe('')
  })

  it('保存的分类可以读回', () => {
    saveMessageCategory('audit')
    expect(readMessageCategory(VALID)).toBe('audit')
  })

  it('「全部」按空串存取', () => {
    saveMessageCategory('task')
    saveMessageCategory('')
    expect(readMessageCategory(VALID)).toBe('')
  })

  it('枚举外的存量值不采纳，按「全部」处理', () => {
    localStorage.setItem(MESSAGE_CATEGORY_KEY, 'removed-category')
    expect(readMessageCategory(VALID)).toBe('')
  })

  it('清空后读回「全部」', () => {
    saveMessageCategory('plan')
    clearMessageCategory()
    expect(localStorage.getItem(MESSAGE_CATEGORY_KEY)).toBeNull()
    expect(readMessageCategory(VALID)).toBe('')
  })
})
