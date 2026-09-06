# Wiki模块扩展功能文档

本文档详细说明了Wiki模块新增的所有扩展功能。

## 功能概览

### 1. 标签系统 (Tag System)
- **功能描述**: 为文档添加标签，支持多维度分类和检索
- **数据模型**: `Tag`, `DocumentTag`
- **API端点**:
  - `POST /wiki/spaces/tags` - 创建标签
  - `GET /wiki/spaces/tags` - 获取标签列表
  - `DELETE /wiki/spaces/tags/:id` - 删除标签
  - `POST /wiki/documents/:id/tags/:tagId` - 为文档添加标签
  - `DELETE /wiki/documents/:id/tags/:tagId` - 移除文档标签
  - `GET /wiki/documents/:id/tags` - 获取文档标签列表

### 2. 评论/批注系统 (Comment System)
- **功能描述**: 支持文档评论、回复、@提及功能
- **数据模型**: `Comment`
- **API端点**:
  - `POST /wiki/documents/:id/comments` - 创建评论
  - `GET /wiki/documents/:id/comments` - 获取评论列表
  - `PUT /wiki/comments/:id` - 更新评论
  - `DELETE /wiki/comments/:id` - 删除评论
- **特性**:
  - 支持多级回复
  - @提及通知
  - 评论状态管理（活跃/已解决/已删除）

### 3. 文档分享 (Share Links)
- **功能描述**: 生成可分享的文档链接，支持密码保护和访问控制
- **数据模型**: `ShareLink`
- **API端点**:
  - `POST /wiki/documents/:id/share` - 创建分享链接
  - `GET /wiki/documents/:id/shares` - 获取分享链接列表
  - `DELETE /wiki/shares/:id` - 删除分享链接
  - `GET /wiki/share/:token` - 通过token访问分享文档
- **特性**:
  - 密码保护
  - 过期时间设置
  - 最大访问次数限制
  - 下载权限控制

### 4. 文档模板 (Templates)
- **功能描述**: 预定义文档模板，快速创建标准化文档
- **数据模型**: `DocumentTemplate`
- **API端点**:
  - `POST /wiki/spaces/templates` - 创建模板
  - `GET /wiki/spaces/templates` - 获取模板列表
  - `GET /wiki/spaces/templates/:id` - 获取模板详情
  - `PUT /wiki/spaces/templates/:id` - 更新模板
  - `DELETE /wiki/spaces/templates/:id` - 删除模板
- **特性**:
  - 分类管理
  - 公开/私有模板
  - 支持多种格式（Markdown等）

### 5. 访问统计与分析 (Analytics)
- **功能描述**: 追踪文档访问、编辑、下载等行为
- **数据模型**: `DocumentAccessLog`, `DocumentStats`
- **API端点**:
  - `GET /wiki/documents/:id/stats` - 获取文档统计
  - `GET /wiki/documents/:id/access-logs` - 获取访问日志
- **统计指标**:
  - 总浏览量
  - 总编辑次数
  - 总下载次数
  - 总分享次数
  - 独立访客数
  - 最后访问时间

### 6. 订阅与通知 (Subscriptions & Notifications)
- **功能描述**: 订阅文档变更，接收实时通知
- **数据模型**: `DocumentSubscription`, `Notification`
- **API端点**:
  - `POST /wiki/documents/:id/subscribe` - 订阅文档
  - `DELETE /wiki/documents/:id/subscribe` - 取消订阅
  - `GET /wiki/spaces/subscriptions` - 获取订阅列表
  - `GET /wiki/spaces/notifications` - 获取通知列表
  - `PUT /wiki/notifications/:id/read` - 标记通知为已读
  - `PUT /wiki/notifications/read-all` - 全部标记为已读
  - `GET /wiki/notifications/unread-count` - 获取未读数量
- **通知类型**:
  - 文档编辑通知
  - 评论通知
  - @提及通知
  - 分享通知

### 7. 编辑锁 (Edit Lock)
- **功能描述**: 防止多人同时编辑冲突
- **数据模型**: `EditLock`
- **API端点**:
  - `POST /wiki/documents/:id/edit-lock` - 获取编辑锁
  - `DELETE /wiki/documents/:id/edit-lock` - 释放编辑锁
  - `PUT /wiki/documents/:id/edit-lock` - 刷新编辑锁
  - `GET /wiki/documents/:id/edit-lock` - 获取编辑锁状态
- **特性**:
  - 30分钟自动过期
  - 支持刷新延长
  - 显示当前编辑者信息

### 8. 批量操作 (Batch Operations)
- **功能描述**: 批量移动、删除节点
- **API端点**:
  - `POST /wiki/spaces/nodes/batch` - 批量操作
- **支持操作**:
  - 批量删除
  - 批量移动

### 9. 版本对比 (Revision Comparison)
- **功能描述**: 对比文档不同版本的差异
- **API端点**:
  - `GET /wiki/documents/:id/revisions/compare?version1=1&version2=2` - 对比版本
- **输出格式**:
  - 行级diff
  - 新增/删除/未变更标记
  - 行号对照

### 10. 文档导入/导出 (Import/Export)
- **功能描述**: 支持文档导出为多种格式
- **API端点**:
  - `POST /wiki/documents/:id/export` - 导出文档
  - `POST /wiki/documents/:id/import` - 导入文档
- **支持格式**:
  - Markdown (.md)
  - HTML (.html)
  - PDF (.pdf)

## 数据模型

### Tag (标签)
```go
type Tag struct {
    ID        string
    TenantID  string
    Name      string
    Color     string
    CreatedBy string
    CreatedAt time.Time
}
```

### Comment (评论)
```go
type Comment struct {
    ID         string
    TenantID   string
    DocumentID string
    NodeID     string
    ParentID   string
    Content    string
    CreatedBy  string
    MentionIDs string
    Status     int
    CreatedAt  time.Time
    UpdatedAt  time.Time
}
```

### ShareLink (分享链接)
```go
type ShareLink struct {
    ID            string
    TenantID      string
    DocumentID    string
    NodeID        string
    Token         string
    Password      string
    ExpireAt      *time.Time
    MaxViews      int
    ViewCount     int
    AllowDownload bool
    CreatedBy     string
    CreatedAt     time.Time
}
```

### DocumentTemplate (文档模板)
```go
type DocumentTemplate struct {
    ID          string
    TenantID    string
    Name        string
    Description string
    Content     string
    Format      string
    Category    string
    IsPublic    bool
    CreatedBy   string
    CreatedAt   time.Time
    UpdatedAt   time.Time
}
```

### DocumentAccessLog (访问日志)
```go
type DocumentAccessLog struct {
    ID         string
    TenantID   string
    DocumentID string
    NodeID     string
    UserID     string
    Action     string
    IPAddress  string
    UserAgent  string
    CreatedAt  time.Time
}
```

### DocumentSubscription (订阅)
```go
type DocumentSubscription struct {
    ID         string
    TenantID   string
    DocumentID string
    NodeID     string
    UserID     string
    NotifyType string
    CreatedAt  time.Time
}
```

### Notification (通知)
```go
type Notification struct {
    ID          string
    TenantID    string
    UserID      string
    Type        string
    Title       string
    Content     string
    RelatedID   string
    RelatedType string
    IsRead      bool
    CreatedAt   time.Time
}
```

### EditLock (编辑锁)
```go
type EditLock struct {
    ID         string
    TenantID   string
    DocumentID string
    UserID     string
    LockedAt   time.Time
    ExpiresAt  time.Time
}
```

## 文件清单

### 后端文件
1. `internal/modules/wiki/model/wiki.go` - 新增数据模型
2. `internal/modules/wiki/dto/wiki_dto.go` - 新增DTO定义
3. `internal/modules/wiki/repository/wiki_repository_extended.go` - 扩展Repository
4. `internal/modules/wiki/service/wiki_service_extended.go` - 扩展Service
5. `internal/modules/wiki/handler/wiki_handler_extended.go` - 扩展Handler

## 使用示例

### 创建标签并添加到文档
```bash
# 创建标签
curl -X POST http://localhost:8080/api/wiki/spaces/tags \
  -H "Content-Type: application/json" \
  -d '{"name": "技术文档", "color": "#FF5733"}'

# 添加标签到文档
curl -X POST http://localhost:8080/api/wiki/documents/{docId}/tags/{tagId}
```

### 创建评论并@提及
```bash
curl -X POST http://localhost:8080/api/wiki/documents/{docId}/comments \
  -H "Content-Type: application/json" \
  -d '{
    "node_id": "{nodeId}",
    "content": "这个部分需要修改，@张三 请确认",
    "mention_ids": "userId1,userId2"
  }'
```

### 创建分享链接
```bash
curl -X POST http://localhost:8080/api/wiki/documents/{docId}/share \
  -H "Content-Type: application/json" \
  -d '{
    "password": "123456",
    "expire_at": "2026-12-31T23:59:59Z",
    "max_views": 100,
    "allow_download": false
  }'
```

### 获取编辑锁
```bash
curl -X POST http://localhost:8080/api/wiki/documents/{docId}/edit-lock
```

### 对比版本
```bash
curl http://localhost:8080/api/wiki/documents/{docId}/revisions/compare?version1=1&version2=2
```

### 导出文档
```bash
curl -X POST http://localhost:8080/api/wiki/documents/{docId}/export \
  -H "Content-Type: application/json" \
  -d '{"format": "markdown"}'
```

## 后续规划

### 短期（1-2个月）
- [ ] 前端UI实现所有新功能
- [ ] WebSocket实时协作编辑
- [ ] 富文本编辑器支持
- [ ] Mermaid图表渲染
- [ ] LaTeX公式支持

### 中期（3-6个月）
- [ ] 集成Elasticsearch全文搜索
- [ ] 文档导入（Word、PDF）
- [ ] 批量导入导出
- [ ] 高级权限控制（IP白名单、时间限制）
- [ ] 文档水印

### 长期（6-12个月）
- [ ] 多语言文档支持
- [ ] AI辅助写作
- [ ] 文档智能推荐
- [ ] 移动端适配
- [ ] 离线编辑支持

## 注意事项

1. **权限控制**: 所有操作都需要相应的权限验证
2. **租户隔离**: 所有数据都按租户隔离
3. **软删除**: 评论等数据使用软删除机制
4. **乐观锁**: 文档编辑使用乐观锁防止并发冲突
5. **自动清理**: 编辑锁30分钟自动过期
6. **访问日志**: 所有重要操作都会记录访问日志