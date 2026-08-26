# Wiki 知识库架构

## 概述

Wiki 知识库模块是 MeteorX 多租户 SaaS 平台的核心业务模块之一，提供多租户隔离的知识管理能力。本文档详细阐述 Wiki 模块的五层数据模型、节点树结构、版本管理、权限体系和 Markdown 安全机制。

---

## 五层数据模型

```
WikiSpace (空间)
  └── WikiNode (节点：folder / document)
        └── Document (文档内容，Markdown)
              └── DocumentRevision (历史版本)

WikiSpaceMember (空间成员：协作权限)
WikiNodePermission (节点级权限：补充授权)
TrashItem (回收站项目)
Attachment (文档附件)
```

### 模型关系图

```
┌──────────────┐     ┌──────────────┐     ┌──────────────┐
│  WikiSpace   │────<│  WikiNode    │────<│  WikiNode    │
│  (空间)      │1:N  │  (节点/文件夹)│1:N  │  (子节点)    │
└──────┬───────┘     └──────┬───────┘     └──────────────┘
       │                    │
       │                    │ Type=document
       │                    ▼
       │             ┌──────────────┐     ┌──────────────────┐
       │             │  Document    │────<│ DocumentRevision │
       │             │  (文档内容)  │1:N  │  (历史版本)       │
       │             └──────┬───────┘     └──────────────────┘
       │                    │
       │                    │ 1:N
       │                    ▼
       │             ┌──────────────┐
       │             │  Attachment  │
       │             │  (附件)       │
       │             └──────────────┘
       │
       │ 1:N
       ▼
┌──────────────────┐     ┌──────────────────────┐
│ WikiSpaceMember  │     │ WikiNodePermission   │
│ (空间成员/角色)  │     │ (节点级补充授权)     │
└──────────────────┘     └──────────────────────┘

┌──────────────┐
│  TrashItem   │  回收站（30 天自动过期）
└──────────────┘
```

### 模型详细说明

| 模型 | 表名 | 核心字段 | 说明 |
|------|------|----------|------|
| WikiSpace | wiki_spaces | id, tenant_id, name, visibility, created_by | 知识库空间，支持可见性设置 |
| WikiNode | wiki_nodes | id, space_id, parent_id, type(folder/document), title, sort | 节点树，支持文件夹和文档两种类型 |
| Document | wiki_documents | id, node_id, content, content_html, format, current_ver | 文档内容，支持 Markdown |
| DocumentRevision | wiki_document_revisions | id, document_id, version, content, content_html | 历史版本，乐观锁保护 |
| WikiSpaceMember | wiki_space_members | id, space_id, user_id, role | 空间成员及角色 |
| WikiNodePermission | wiki_node_permissions | id, node_id, user_id, permission | 节点级补充权限 |
| TrashItem | wiki_trash | id, item_type, item_id, space_id, expires_at | 回收站项目 |
| Attachment | wiki_attachments | id, document_id, file_name, file_url | 文档附件 |

---

## 节点树结构

### 树构建算法

```
1. 从仓库获取 Space 下所有节点（扁平列表）
2. 构建 nodeMap: map[nodeID] → TreeNode
3. 遍历节点列表：
   - parent_id 为空 → 加入 roots
   - parent_id 存在且父节点存在 → 挂载到父节点 Children
   - parent_id 存在但父节点不存在 → 加入 roots（容错）
4. 递归排序：按 Sort 字段对每层节点排序
```

### 树结构特点

- **无限层级**：支持任意深度的文件夹嵌套
- **同级排序**：按 Sort 字段升序排列
- **容错机制**：孤立节点自动归入根级
- **类型约束**：文档节点的父节点必须是文件夹类型

### 节点操作安全校验

```
MoveNode 校验链：
  ├── 不能移动到自身
  ├── 目标父节点存在
  ├── 不能跨 Space 移动
  ├── 目标父节点必须是文件夹
  └── 不能移动到自己的子节点下（防循环）
      └── isDescendant 递归检查
```

---

## 版本管理

### 版本创建流程

```
UpdateDocument:
  1. 读取当前文档（获取 current_ver 作为 expectedVer）
  2. 将当前内容保存为 DocumentRevision（版本号 = current_ver）
  3. 更新文档内容，current_ver++
  4. 使用乐观锁更新（WHERE current_ver = expectedVer）
  5. 冲突时返回 "文档已被其他用户修改"
```

### 版本恢复流程

```
RestoreRevision:
  1. 获取目标版本内容
  2. 将当前状态保存为新 Revision（自动保存快照）
  3. 用历史版本内容覆盖当前文档
  4. current_ver = 原版本号 + 1
  5. 乐观锁更新
```

### 乐观锁实现

```go
// UpdateDocument 使用版本号作为乐观锁
doc.CurrentVer++
repo.UpdateDocument(ctx, doc, expectedVer)
// SQL: UPDATE ... SET current_ver = ? WHERE id = ? AND current_ver = ?
```

---

## 权限体系

### 双层权限模型

```
┌─────────────────────────────────────────────────┐
│  Layer 1: Space 角色权限（基础权限）             │
│                                                  │
│  Owner  → 全部权限                               │
│  Admin  → 除删除 Space 外的所有权限              │
│  Editor → 节点/文档的创建、编辑、读取            │
│  Viewer → 仅读取                                 │
├─────────────────────────────────────────────────┤
│  Layer 2: Node 节点权限（补充授权）             │
│                                                  │
│  view   → 仅查看                                 │
│  edit   → 查看 + 编辑                            │
│  delete → 查看 + 编辑 + 删除                     │
└─────────────────────────────────────────────────┘
```

### 权限检查流程

```
CheckNodePermission(nodeID, userID, action):
  1. 获取节点 → 得到 spaceID
  2. mapActionToSpaceLevel(action) → spaceAction
  3. CheckSpacePermission(spaceID, userID, spaceAction)
     ├── 获取成员角色
     ├── 查表 spaceRolePermissions[role][action]
     └── 通过 → 返回 nil
  4. checkNodeLevelPermission(nodeID, userID, action)
     ├── 查询 WikiNodePermission
     ├── 检查权限类型是否匹配 action
     └── 通过 → 返回 nil
  5. 两个都失败 → 返回 Space 权限错误
```

### Space 可见性

| 类型 | 说明 | 权限要求 |
|------|------|----------|
| Private (1) | 私有空间 | 必须是成员 |
| Tenant (2) | 租户可见 | 同租户用户可查看 |
| Public (3) | 公开空间 | 所有人可查看 |

---

## Markdown 安全

### 渲染流程

```
Markdown 原文
    ↓
basicMarkdownToHTML()
    ├── html.EscapeString() — HTML 转义
    ├── 正则替换标题 (h1-h6)
    ├── 正则替换 **粗体** / *斜体*
    ├── 正则替换 `行内代码`
    ├── 正则替换 ```代码块```
    ├── 正则替换 [链接]()
    ├── 正则替换 ![图片]()
    └── 正则替换列表项
    ↓
SanitizeHTML()
    ├── 移除 <script> 标签
    ├── 移除事件处理器 (onclick, onload 等)
    ├── 移除 javascript: 协议
    ├── 移除危险标签 (iframe, object, embed, form 等)
    ├── 移除 data: 协议
    └── 移除 vbscript: 协议
    ↓
ContentHTML (安全的 HTML)
```

### XSS 防护策略

1. **HTML 转义**：`html.EscapeString()` 先转义原始内容
2. **标签白名单**：只保留 Markdown 生成的安全标签
3. **属性清洗**：移除所有事件处理器和危险协议
4. **双重验证**：先转义再渲染，最后再 sanitize

---

## 回收站

### 生命周期

```
删除操作
    ↓
MoveToTrash() — 创建 TrashItem (expires_at = now + 30 days)
    ↓
级联物理删除（节点递归删除 + 文档 + 附件）
    ↓
30 天内可恢复
    ↓
30 天后自动过期（定时任务清理）
```

### 恢复逻辑

```
RestoreTrashItem:
  1. 获取 TrashItem
  2. 检查 Space 权限（space:update）
  3. 根据 item_type 调用对应恢复逻辑
     ├── space → restoreSpaceFromTrash
     ├── node → restoreNodeFromTrash
     └── document → restoreDocumentFromTrash
  4. 删除 TrashItem 记录
```

---

## 搜索

### 搜索流程

```
Search(query, spaceID, userID):
  1. 标题搜索 → SearchNodesByTitle (权重 1.0)
  2. 内容搜索 → SearchDocumentsByContent (权重 0.8)
  3. 权限过滤 → 对每条结果执行 CheckNodePermission
  4. 生成摘要 → generateSnippet(content, query)
     ├── 定位关键字位置
     ├── 前后各取 50 字符
     └── 省略号标记
  5. 合并结果 + 分页
```

### 搜索结果排序

- 标题匹配权重 1.0
- 内容匹配权重 0.8
- 按更新时间排序

---

## 多租户隔离

所有 Wiki API 自动通过 `tenantctx.FilterQuery()` 进行租户隔离：

- 每个空间、节点、文档、版本、成员、附件均绑定 `tenant_id`
- 普通用户只能访问本租户下的数据
- 系统管理员（`IsMaster=true`）可跨租户查看
- 回收站按 `tenant_id` 隔离

### Repository 层隔离

```go
func (r *wikiRepository) ListSpaces(ctx context.Context, ...) ([]*model.WikiSpace, int64, error) {
    query := tenantctx.FilterQuery(ctx, r.db.WithContext(ctx), "tenant_id")
    // ↑ 自动注入 WHERE tenant_id = ?
    query.Where("tenant_id = ?", tenantID)
    // ...
}
```

---

## 事务管理

### 事务边界

| 操作 | 事务范围 | 说明 |
|------|----------|------|
| CreateSpace | Space + Member | 原子创建空间并添加 Owner |
| CreateNode(Document) | Node + Document | 原子创建文档节点及关联文档 |
| UpdateDocument | Revision + Document | 原子保存历史版本并更新 |
| DeleteNode | Trash + 递归删除 | 先创建回收站再级联删除 |
| RestoreRevision | Revision + Document | 原子保存快照并恢复 |
| RestoreTrashItem | 恢复 + 清理 | 原子恢复并清除回收站 |

### 使用方式

```go
s.tx.WithTx(ctx, func(txCtx context.Context, _ *gorm.DB) error {
    // txCtx 已绑定事务
    // 所有 Repository 操作通过 txCtx 执行
    // 出错自动回滚
    return nil
})
```

---

## API 路由总览

```
/wiki
├── GET    /stats                              → Wiki 统计
├── GET    /search                             → Wiki 搜索
├── GET    /trash                              → 回收站列表
├── POST   /trash/{id}/restore                 → 恢复回收站
├── DELETE /trash/{id}                         → 永久删除
│
├── /spaces
│   ├── GET    /                                → 空间列表
│   ├── POST   /                                → 创建空间
│   ├── GET    /{id}                            → 空间详情
│   ├── PUT    /{id}                            → 更新空间
│   ├── DELETE /{id}                            → 删除空间
│   │
│   ├── /{spaceId}/nodes
│   │   ├── GET    /tree                        → 节点树
│   │   ├── POST   /                            → 创建节点
│   │   ├── GET    /{id}                        → 节点详情
│   │   ├── PUT    /{id}                        → 更新节点
│   │   ├── DELETE /{id}                        → 删除节点
│   │   ├── POST   /{id}/move                   → 移动节点
│   │   ├── PUT    /{id}/sort                   → 节点排序
│   │   ├── GET    /{id}/permissions            → 节点权限列表
│   │   ├── POST   /{id}/permissions            → 设置节点权限
│   │   └── DELETE /{id}/permissions/{uid}/{perm} → 移除节点权限
│   │
│   └── /{spaceId}/members
│       ├── GET    /                            → 成员列表
│       ├── POST   /                            → 添加成员
│       └── DELETE /{userId}                    → 移除成员
│
└── /documents
    ├── POST   /nodes/{nodeId}                 → 创建文档
    ├── GET    /nodes/{nodeId}                 → 获取文档
    ├── PUT    /{id}                           → 更新文档
    ├── DELETE /{id}                           → 删除文档
    ├── GET    /{id}/revisions                 → 版本列表
    ├── GET    /{id}/revisions/{version}       → 指定版本
    ├── POST   /{id}/revisions/{version}/restore → 恢复版本
    ├── POST   /attachments                    → 创建附件
    ├── GET    /{documentId}/attachments       → 附件列表
    └── DELETE /attachments/{id}               → 删除附件
```