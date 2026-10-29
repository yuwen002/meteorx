import { describe, it, expect } from 'vitest'
import { toPageResult, asArray } from './pagination'

describe('asArray', () => {
  it('数组本体直接返回（列表接口非空时拦截器已解包为数组）', () => {
    expect(asArray([1, 2, 3])).toEqual([1, 2, 3])
    expect(asArray([])).toEqual([])
  })

  it('空列表信封 { data: null } 兜底为空数组（拦截器 res.data ?? res 回退产物）', () => {
    expect(asArray({ data: null })).toEqual([])
  })

  it('{ data: T[] } 形态取 data 数组', () => {
    expect(asArray({ data: ['a', 'b'] })).toEqual(['a', 'b'])
  })

  it('null / undefined / 非数组标量一律兜底为空数组', () => {
    expect(asArray(null)).toEqual([])
    expect(asArray(undefined)).toEqual([])
    expect(asArray(0)).toEqual([])
    expect(asArray('x')).toEqual([])
    expect(asArray({})).toEqual([])
  })
})

describe('toPageResult', () => {
  it('归一化 { data, pagination } 分页包裹', () => {
    const res = toPageResult<string>({ data: ['a'], pagination: { total: 42 } })
    expect(res).toEqual({ list: ['a'], total: 42 })
  })

  it('空/非法输入返回空列表', () => {
    expect(toPageResult(null)).toEqual({ list: [], total: 0 })
    expect(toPageResult({ data: null })).toEqual({ list: [], total: 0 })
  })
})
