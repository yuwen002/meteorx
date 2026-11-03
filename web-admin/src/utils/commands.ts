// 命令面板（Ctrl/Cmd+K）的命令目录与检索逻辑。
//
// 设计：buildCommands/filterCommands 均为纯函数，不直接依赖 Pinia、router 或 DOM，
// 由调用方（CommandPalette.vue）注入上下文，便于脱离挂载做单元测试与权限门控回归。
// 导航项与侧边栏（DefaultLayout）保持一致的可见性规则：admin 项仅平台超管可见，
// perm 项需具备对应权限码或为超管。

/** 命令分组：导航跳转 / 全局操作 */
export type CommandSection = 'nav' | 'action'

/** 单条命令：解析后的展示文案 + 检索关键词 + 右侧提示 + 执行动作 */
export interface PaletteCommand {
  id: string
  section: CommandSection
  /** 已用 t() 解析的展示文案 */
  label: string
  /** 附加检索词（中英文别名、路径等），大小写不敏感 */
  keywords: string
  /** 右栏提示（导航为路径，操作为空） */
  hint?: string
  run: () => void
}

/** 构建命令所需的外部上下文，由组件从 store/router 组装后注入 */
export interface CommandContext {
  t: (key: string, ...args: (string | number)[]) => string
  theme: 'light' | 'dark'
  locale: string
  isAdmin: boolean
  hasPermission: (code: string) => boolean
  navigate: (path: string) => void
  toggleSidebar: () => void
  toggleTheme: () => void
  setLocale: (locale: string) => void
  logout: () => void
}

interface NavItem {
  path: string
  titleKey: string
  keywords: string
  /** 仅平台超管可见 */
  admin?: boolean
  /** 需要的权限码（超管始终可见） */
  perm?: string
}

// 导航命令目录：覆盖工作区高频页面与超管常用入口，标题复用既有 nav.* / profile.* i18n 键。
const NAV_ITEMS: NavItem[] = [
  { path: '/dashboard', titleKey: 'nav.dashboard', keywords: 'dashboard home 工作台 首页 概览' },
  { path: '/task', titleKey: 'nav.task', keywords: 'task todo 任务 待办' },
  { path: '/chat', titleKey: 'nav.chat', keywords: 'chat im 私聊 群聊 聊天 会话' },
  { path: '/messages', titleKey: 'nav.messages', keywords: 'message inbox 消息 站内信' },
  { path: '/announcement', titleKey: 'nav.announcement', keywords: 'announcement 公告' },
  { path: '/export', titleKey: 'nav.export', keywords: 'export download 导出 下载' },
  { path: '/wiki', titleKey: 'nav.wiki', keywords: 'wiki doc 文档 知识库', perm: 'wiki:list' },
  { path: '/invitation', titleKey: 'nav.invitation', keywords: 'invite member 邀请 成员', perm: 'user:list' },
  { path: '/tenant-settings', titleKey: 'nav.tenantSettings', keywords: 'tenant settings 租户 设置' },
  { path: '/profile', titleKey: 'profile.title', keywords: 'profile account 个人 中心 账号' },
  { path: '/system/settings', titleKey: 'nav.settings', keywords: 'settings config 配置 系统设置', admin: true },
  { path: '/system/user', titleKey: 'nav.user', keywords: 'user 用户 管理', admin: true },
  { path: '/system/role', titleKey: 'nav.role', keywords: 'role 角色 权限', admin: true },
]

/** 按上下文可见性规则组装全部命令（导航在前、操作在后）。 */
export function buildCommands(ctx: CommandContext): PaletteCommand[] {
  const nav: PaletteCommand[] = NAV_ITEMS.filter(
    (it) => (!it.admin || ctx.isAdmin) && (!it.perm || ctx.isAdmin || ctx.hasPermission(it.perm))
  ).map((it) => ({
    id: `nav:${it.path}`,
    section: 'nav',
    label: ctx.t(it.titleKey),
    keywords: it.keywords,
    hint: it.path,
    run: () => ctx.navigate(it.path),
  }))

  const zh = ctx.locale === 'zh-CN'
  const actions: PaletteCommand[] = [
    {
      id: 'act:sidebar',
      section: 'action',
      label: ctx.t('a11y.toggleSidebar'),
      keywords: 'sidebar collapse toggle 侧边栏 折叠 展开',
      run: ctx.toggleSidebar,
    },
    {
      id: 'act:theme',
      section: 'action',
      label: ctx.t(ctx.theme === 'dark' ? 'theme.toLight' : 'theme.toDark'),
      keywords: 'theme dark light 主题 深色 浅色 暗色',
      run: ctx.toggleTheme,
    },
    {
      id: 'act:lang',
      section: 'action',
      label: ctx.t(zh ? 'cmdk.lang.en' : 'cmdk.lang.zh'),
      keywords: 'language locale switch 语言 english 中文',
      run: () => ctx.setLocale(zh ? 'en-US' : 'zh-CN'),
    },
    {
      id: 'act:logout',
      section: 'action',
      label: ctx.t('auth.logout'),
      keywords: 'logout sign out 退出 登出',
      run: ctx.logout,
    },
  ]

  return [...nav, ...actions]
}

/**
 * 按关键词过滤命令：对 label + keywords 做大小写不敏感的子序列匹配
 * （字符顺序出现即可，容忍 "tsk" 命中 "task"），空查询返回全量。
 */
export function filterCommands(cmds: PaletteCommand[], query: string): PaletteCommand[] {
  const q = query.trim().toLowerCase()
  if (!q) return cmds
  const spaceless = q.replace(/\s+/g, '')
  return cmds.filter((c) => {
    const hay = `${c.label} ${c.keywords}`.toLowerCase()
    if (hay.includes(q)) return true
    return isSubsequence(spaceless, hay.replace(/\s+/g, ''))
  })
}

// 判断 sub 是否为 full 的子序列（字符按序出现，无需连续）
function isSubsequence(sub: string, full: string): boolean {
  if (!sub) return true
  let i = 0
  for (let j = 0; j < full.length && i < sub.length; j++) {
    if (full[j] === sub[i]) i++
  }
  return i === sub.length
}
