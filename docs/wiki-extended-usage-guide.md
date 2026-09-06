# Wiki 扩展功能使用指南

## 📚 目录

1. [标签系统](#1-标签系统)
2. [评论系统](#2-评论系统)
3. [分享链接](#3-分享链接)
4. [文档模板](#4-文档模板)
5. [访问统计](#5-访问统计)
6. [订阅管理](#6-订阅管理)
7. [通知系统](#7-通知系统)
8. [编辑锁](#8-编辑锁)
9. [批量操作](#9-批量操作)
10. [版本对比](#10-版本对比)
11. [导入导出](#11-导入导出)

---

## 1. 标签系统

### 功能说明
为文档添加标签，方便分类和检索。

### 使用方法

#### 创建标签
1. 点击工具栏的"管理标签"按钮
2. 在弹出的对话框中输入标签名称
3. 选择标签颜色
4. 点击"创建标签"

#### 为文档添加标签
```typescript
import { addDocumentTag } from '@/api/modules/wiki'

// 为文档添加标签
await addDocumentTag(documentId, tagId)
```

#### 查看文档标签
```vue
<template>
  <div class="tags">
    <el-tag 
      v-for="tag in tags" 
      :key="tag.id" 
      :color="tag.color"
    >
      {{ tag.name }}
    </el-tag>
  </div>
</template>

<script setup lang="ts">
import { listDocumentTags } from '@/api/modules/wiki'

const tags = await listDocumentTags(documentId)
</script>
```

---

## 2. 评论系统

### 功能说明
支持对文档进行评论和回复，可以@提及其他用户。

### 使用方法

#### 发表评论
```vue
<CommentSection :document-id="documentId" />
```

#### API调用
```typescript
import { createComment } from '@/api/modules/wiki'

// 发表评论
await createComment({
  document_id: documentId,
  content: '这是一条评论'
})

// 回复评论
await createComment({
  document_id: documentId,
  parent_id: parentCommentId,
  content: '这是对评论的回复'
})
```

---

## 3. 分享链接

### 功能说明
生成文档的分享链接，支持密码保护、过期时间和浏览次数限制。

### 使用方法

#### 创建分享链接
```typescript
import { createShareLink } from '@/api/modules/wiki'

const link = await createShareLink({
  document_id: documentId,
  password: '123456',  // 可选
  expires_at: '2024-12-31 23:59:59',  // 可选
  max_views: 100  // 可选
})
```

#### 复制链接
```typescript
const link = `${window.location.origin}/wiki/share/${token}`
navigator.clipboard.writeText(link)
```

#### 访问分享链接
```typescript
import { getShareLink } from '@/api/modules/wiki'

const shareLink = await getShareLink(token, password)
```

---

## 4. 文档模板

### 功能说明
创建和使用文档模板，快速创建标准化文档。

### 使用方法

#### 创建模板
```typescript
import { createTemplate } from '@/api/modules/wiki'

await createTemplate({
  name: '会议纪要模板',
  description: '用于记录会议内容',
  category: '会议',
  content: '# 会议纪要\n\n## 时间\n\n## 参会人员\n\n## 会议内容\n',
  is_public: true  // 是否公开
})
```

#### 使用模板
```vue
<TemplateSelector @select="handleTemplateSelect" />

<script setup lang="ts">
function handleTemplateSelect(template: DocumentTemplate) {
  // 使用模板内容创建新文档
  createDocument({
    node_id: nodeId,
    content: template.content
  })
}
</script>
```

---

## 5. 访问统计

### 功能说明
统计文档的浏览量、编辑次数、下载次数等数据。

### 使用方法

```vue
<StatsPanel :document-id="documentId" />
```

#### API调用
```typescript
import { getDocumentStats, listAccessLogs } from '@/api/modules/wiki'

// 获取统计数据
const stats = await getDocumentStats(documentId)
console.log(stats.total_views)  // 总浏览量
console.log(stats.total_edits)  // 总编辑次数

// 获取访问日志
const logs = await listAccessLogs(documentId, 1, 20)
```

---

## 6. 订阅管理

### 功能说明
订阅文档，当文档有更新时接收通知。

### 使用方法

```typescript
import { subscribeDocument, unsubscribeDocument } from '@/api/modules/wiki'

// 订阅文档
await subscribeDocument(documentId)

// 取消订阅
await unsubscribeDocument(documentId)

// 获取订阅列表
const subscriptions = await listSubscriptions(documentId)
```

---

## 7. 通知系统

### 功能说明
接收文档相关的通知，如评论、分享、编辑等。

### 使用方法

```vue
<NotificationCenter />
```

#### API调用
```typescript
import { 
  listNotifications,
  markNotificationAsRead,
  markAllNotificationsAsRead,
  getUnreadNotificationCount
} from '@/api/modules/wiki'

// 获取通知列表
const { notifications, total, unread_count } = await listNotifications(1, 20)

// 标记为已读
await markNotificationAsRead(notificationId)

// 全部标记为已读
await markAllNotificationsAsRead()

// 获取未读数量
const { count } = await getUnreadNotificationCount()
```

---

## 8. 编辑锁

### 功能说明
防止多人同时编辑同一文档，避免内容冲突。

### 使用方法

```vue
<EditLockIndicator 
  :document-id="documentId" 
  :current-user-id="userId"
  @locked="handleLocked"
  @unlocked="handleUnlocked"
/>
```

#### API调用
```typescript
import { 
  acquireEditLock,
  releaseEditLock,
  refreshEditLock,
  getEditLock
} from '@/api/modules/wiki'

// 获取编辑锁
const lock = await acquireEditLock(documentId)

// 释放编辑锁
await releaseEditLock(documentId)

// 刷新编辑锁（延长锁定时间）
await refreshEditLock(documentId)

// 获取编辑锁状态
try {
  const lock = await getEditLock(documentId)
  console.log('文档被锁定')
} catch (error) {
  console.log('文档未被锁定')
}
```

---

## 9. 批量操作

### 功能说明
批量删除或移动节点。

### 使用方法

```typescript
import { batchDeleteNodes, batchMoveNodes } from '@/api/modules/wiki'

// 批量删除
await batchDeleteNodes({
  node_ids: ['id1', 'id2', 'id3']
})

// 批量移动
await batchMoveNodes({
  node_ids: ['id1', 'id2', 'id3'],
  new_parent_id: 'newParentId'
})
```

---

## 10. 版本对比

### 功能说明
对比文档的两个历史版本，查看差异。

### 使用方法

```typescript
import { compareRevisions } from '@/api/modules/wiki'

const { diff } = await compareRevisions(documentId, version1, version2)
console.log(diff)  // 差异内容
```

---

## 11. 导入导出

### 功能说明
支持文档的导入和导出，格式包括Markdown、HTML等。

### 使用方法

#### 导出文档
```typescript
import { exportDocument } from '@/api/modules/wiki'

const { content, filename } = await exportDocument(documentId, 'markdown')

// 下载文件
const blob = new Blob([content], { type: 'text/markdown' })
const url = URL.createObjectURL(blob)
const a = document.createElement('a')
a.href = url
a.download = filename
a.click()
URL.revokeObjectURL(url)
```

#### 导入文档
```typescript
import { importDocument } from '@/api/modules/wiki'

const file = fileInput.files[0]
const result = await importDocument(spaceId, file)
console.log('导入的文档ID:', result.document_id)
```

---

## 🎯 最佳实践

### 1. 标签管理
- 建议创建统一的标签规范
- 使用不同颜色区分不同类型的标签
- 定期清理未使用的标签

### 2. 评论系统
- 使用@提及功能通知相关人员
- 及时回复他人的评论
- 删除无意义的评论

### 3. 分享链接
- 敏感文档务必设置密码保护
- 设置合理的过期时间
- 限制浏览次数防止滥用

### 4. 编辑锁
- 编辑完成后及时释放编辑锁
- 避免长时间锁定文档
- 使用强制获取功能前请先沟通

### 5. 订阅管理
- 只订阅真正关心的文档
- 定期清理不再关注的订阅

---

## ⚠️ 注意事项

1. **权限控制**: 所有操作都需要相应的权限，请确保用户有正确的权限
2. **数据一致性**: 批量操作使用事务，确保数据一致性
3. **性能优化**: 访问日志建议定期清理，避免数据量过大
4. **安全性**: 分享链接的密码应加密存储
5. **用户体验**: 编辑锁的默认超时时间为30分钟

---

## 🐛 常见问题

### Q: 编辑锁超时怎么办？
A: 使用 `refreshEditLock()` 刷新锁定时间，或重新获取编辑锁。

### Q: 如何查看谁订阅了我的文档？
A: 使用 `listSubscriptions(documentId)` 获取订阅列表。

### Q: 分享链接失效了怎么办？
A: 重新创建一个新的分享链接。

### Q: 如何批量导入文档？
A: 使用 `importDocument()` API，支持Markdown格式文件。

---

## 📞 技术支持

如有问题，请查看：
- [Wiki扩展功能文档](./wiki-extended-features.md)
- [前端组件文档](./wiki-frontend-components.md)
- API文档: `docs/api/wiki-api.md`