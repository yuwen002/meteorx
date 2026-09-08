import { describe, it, expect, vi, beforeEach } from 'vitest'
import { formatWikiNodeTree, visibilityText } from './wiki'
import type { WikiNodeTree } from './wiki'

describe('Wiki API Utils', () => {
  describe('formatWikiNodeTree', () => {
    it('should flatten tree to array', () => {
      const tree: WikiNodeTree[] = [
        {
          id: '1',
          space_id: 'space-1',
          parent_id: '',
          type: 'folder',
          title: 'Folder 1',
          sort_order: 1,
          created_at: '2024-01-01',
          children: [
            {
              id: '2',
              space_id: 'space-1',
              parent_id: '1',
              type: 'document',
              title: 'Doc 1',
              sort_order: 1,
              created_at: '2024-01-01',
              children: [],
            },
          ],
        },
      ]

      const result = formatWikiNodeTree(tree)
      expect(result).toHaveLength(2)
      expect(result[0].id).toBe('1')
      expect(result[1].id).toBe('2')
    })

    it('should handle empty tree', () => {
      expect(formatWikiNodeTree([])).toEqual([])
    })

    it('should handle tree without children', () => {
      const tree: WikiNodeTree[] = [
        {
          id: '1',
          space_id: 'space-1',
          parent_id: '',
          type: 'document',
          title: 'Doc 1',
          sort_order: 1,
          created_at: '2024-01-01',
        },
      ]

      const result = formatWikiNodeTree(tree)
      expect(result).toHaveLength(1)
    })
  })

  describe('visibilityText', () => {
    it('should return correct text for visibility values', () => {
      expect(visibilityText(1)).toBe('私有')
      expect(visibilityText(2)).toBe('空间')
      expect(visibilityText(3)).toBe('租户')
      expect(visibilityText(4)).toBe('公开')
    })

    it('should return unknown for invalid visibility', () => {
      expect(visibilityText(0)).toBe('未知')
      expect(visibilityText(99)).toBe('未知')
    })
  })
})