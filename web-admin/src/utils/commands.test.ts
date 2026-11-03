// 命令面板数据层单测：命令可见性门控、动态标签、run 动作与检索过滤均为纯函数，脱 DOM 验证。
import { describe, it, expect, vi } from 'vitest'
import { buildCommands, filterCommands, type CommandContext, type PaletteCommand } from './commands'

// t 回显 key（便于按 titleKey 断言标签）
const t = (key: string) => key

function makeCtx(overrides: Partial<CommandContext> = {}): CommandContext {
  return {
    t,
    theme: 'light',
    locale: 'zh-CN',
    isAdmin: false,
    hasPermission: () => false,
    navigate: vi.fn(),
    toggleSidebar: vi.fn(),
    toggleTheme: vi.fn(),
    setLocale: vi.fn(),
    logout: vi.fn(),
    ...overrides,
  }
}

const ids = (cmds: PaletteCommand[]) => cmds.map((c) => c.id)

describe('buildCommands', () => {
  it('普通用户隐藏超管项，权限门控项按权限码取舍', () => {
    const ctx = makeCtx({ hasPermission: (code) => code === 'wiki:list' })
    const got = ids(buildCommands(ctx))
    // 超管专属入口不出现
    expect(got).not.toContain('nav:/system/settings')
    expect(got).not.toContain('nav:/system/user')
    // wiki:list 命中，user:list 未授予
    expect(got).toContain('nav:/wiki')
    expect(got).not.toContain('nav:/invitation')
    // 无权限的基础导航仍在
    expect(got).toContain('nav:/dashboard')
    expect(got).toContain('nav:/task')
  })

  it('超管可见全部导航项（含 admin 与权限门控项）', () => {
    const got = ids(buildCommands(makeCtx({ isAdmin: true })))
    expect(got).toContain('nav:/system/settings')
    expect(got).toContain('nav:/system/user')
    expect(got).toContain('nav:/wiki')
    expect(got).toContain('nav:/invitation')
  })

  it('始终包含四个操作命令，且导航在前、操作在后', () => {
    const got = buildCommands(makeCtx())
    const acts = got.filter((c) => c.section === 'action').map((c) => c.id)
    expect(acts).toEqual(['act:sidebar', 'act:theme', 'act:lang', 'act:logout'])
    expect(got[0].section).toBe('nav')
    expect(got[got.length - 1].section).toBe('action')
  })

  it('主题/语言动作标签随当前状态切换', () => {
    const light = buildCommands(makeCtx({ theme: 'light' })).find((c) => c.id === 'act:theme')
    expect(light?.label).toBe('theme.toDark')
    const dark = buildCommands(makeCtx({ theme: 'dark' })).find((c) => c.id === 'act:theme')
    expect(dark?.label).toBe('theme.toLight')
    const zh = buildCommands(makeCtx({ locale: 'zh-CN' })).find((c) => c.id === 'act:lang')
    expect(zh?.label).toBe('cmdk.lang.en')
    const en = buildCommands(makeCtx({ locale: 'en-US' })).find((c) => c.id === 'act:lang')
    expect(en?.label).toBe('cmdk.lang.zh')
  })

  it('run 动作分派到对应回调；语言命令按当前语言切到另一语', () => {
    const ctx = makeCtx({ locale: 'zh-CN' })
    buildCommands(ctx).find((c) => c.id === 'act:lang')?.run()
    expect(ctx.setLocale).toHaveBeenCalledWith('en-US')
    buildCommands(ctx).find((c) => c.id === 'act:logout')?.run()
    expect(ctx.logout).toHaveBeenCalled()
    buildCommands(ctx).find((c) => c.id === 'nav:/task')?.run()
    expect(ctx.navigate).toHaveBeenCalledWith('/task')
  })
})

describe('filterCommands', () => {
  const cmds = buildCommands(makeCtx({ isAdmin: true }))

  it('空查询返回全量', () => {
    expect(filterCommands(cmds, '   ')).toHaveLength(cmds.length)
  })

  it('按关键词（英文别名）大小写不敏感匹配', () => {
    const got = ids(filterCommands(cmds, 'TASK'))
    expect(got).toContain('nav:/task')
    expect(got).not.toContain('nav:/dashboard')
    expect(got).not.toContain('nav:/export')
  })

  it('支持子序列模糊（tsk 命中 task 的关键词）', () => {
    const got = ids(filterCommands(cmds, 'tsk'))
    expect(got).toContain('nav:/task')
  })

  it('中文关键词可命中', () => {
    const got = ids(filterCommands(cmds, '聊天'))
    expect(got).toContain('nav:/chat')
  })

  it('无匹配返回空', () => {
    expect(filterCommands(cmds, 'zzzqqq')).toHaveLength(0)
  })
})
