/**
 * 消息中心分类页签的持久化。
 *
 * 用户常按「任务」「告警」定点看消息，刷新一次就回到「全部」等于每次重选，
 * 所以页签值存在本地。但反过来的入口也同样要照顾：铃铛里的「查看全部消息」
 * 必须能把记住的分类清掉，否则点进去看到的仍是被筛过的列表。
 */
export const MESSAGE_CATEGORY_KEY = 'meteorx_message_category'

/** 读取记住的分类；空串表示「全部」。枚举外的存量值（旧版本残留）一律当「全部」。 */
export function readMessageCategory(validValues: readonly string[]): string {
  const stored = localStorage.getItem(MESSAGE_CATEGORY_KEY) || ''
  return validValues.includes(stored) ? stored : ''
}

export function saveMessageCategory(category: string): void {
  localStorage.setItem(MESSAGE_CATEGORY_KEY, category)
}

export function clearMessageCategory(): void {
  localStorage.removeItem(MESSAGE_CATEGORY_KEY)
}
