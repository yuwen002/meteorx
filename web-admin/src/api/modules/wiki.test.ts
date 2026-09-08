import { describe, it, expect } from 'vitest'
import type { WikiNodeTree } from './wiki'

describe('Wiki Types', () => {
  it('should have correct WikiNodeTree structure', () => {
    const node: WikiNodeTree = {
      id: '1',
      space_id: 'space-1',
      parent_id: '',
      type: 'folder',
      title: 'Test Folder',
      icon: '📁',
      sort: 1,
      status: 1,
      owner_id: 'user-1',
      child_count: 0,
      created_at: '2024-01-01T00:00:00Z',
      updated_at: '2024-01-01T00:00:00Z',
      children: [],
    }

    expect(node.id).toBe('1')
    expect(node.type).toBe('folder')
    expect(node.children).toEqual([])
  })

  it('should support nested tree structure', () => {
    const tree: WikiNodeTree[] = [
      {
        id: '1',
        space_id: 'space-1',
        parent_id: '',
        type: 'folder',
        title: 'Folder 1',
        icon: '📁',
        sort: 1,
        status: 1,
        owner_id: 'user-1',
        child_count: 1,
        created_at: '2024-01-01',
        updated_at: '2024-01-01',
        children: [
          {
            id: '2',
            space_id: 'space-1',
            parent_id: '1',
            type: 'document',
            title: 'Doc 1',
            icon: '📄',
            sort: 1,
            status: 1,
            owner_id: 'user-1',
            child_count: 0,
            created_at: '2024-01-01',
            updated_at: '2024-01-01',
            children: [],
          },
        ],
      },
    ]

    expect(tree).toHaveLength(1)
    expect(tree[0].children).toHaveLength(1)
    expect(tree[0].children![0].type).toBe('document')
  })
})