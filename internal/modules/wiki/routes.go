package wiki

import (
	"meteorx/internal/config"
	"meteorx/internal/modules/wiki/handler"
	"meteorx/internal/modules/wiki/repository"
	"meteorx/internal/modules/wiki/service"
	"meteorx/internal/search"

	db "meteorx/internal/pkg/db"

	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"
)

// InitModule 初始化 Wiki 模块，注册路由与依赖
func InitModule(r chi.Router, gormDB *gorm.DB, tx *db.TxManager, cfg *config.Config) {
	repo := repository.NewWikiRepository(gormDB)
	svc := service.NewWikiService(repo, tx)

	// 初始化全文搜索引擎（配置为 none 时降级为数据库 LIKE 搜索）
	searchCfg := search.Config{
		Provider:    cfg.Search.Provider,
		Host:        cfg.Search.Host,
		APIKey:      cfg.Search.APIKey,
		IndexPrefix: cfg.Search.IndexPrefix,
	}
	engine := search.NewEngine(searchCfg)
	indexer := search.NewWikiIndexer(engine, repo, repo)
	svc.SetWikiIndexer(indexer)

	// 文档内嵌 /uploads 图片签名支持（与文件模块共用访问基址与独立签名密钥）
	if cfg != nil {
		svc.SetUploadSigner(cfg.File.UploadURL, cfg.File.UploadSignKey(cfg.JWT.Secret))
	}
	h := handler.NewWikiHandler(svc)

	// 初始化扩展服务
	extRepo := repository.NewWikiRepositoryExtended(gormDB)
	extSvc := service.NewWikiServiceExtended(extRepo, tx)
	extH := handler.NewWikiHandlerExtended(extSvc)

	r.Route("/wiki", func(r chi.Router) {
		r.Get("/stats", h.GetStats) // Wiki 统计数据
		r.Get("/search", h.Search)  // Wiki 搜索（标题+内容）

		// Trash 回收站
		r.Get("/trash", h.ListTrashItems)                   // 回收站列表
		r.Post("/trash/{id}/restore", h.RestoreTrashItem)   // 从回收站恢复
		r.Delete("/trash/{id}", h.PermanentDeleteTrashItem) // 永久删除

		// Spaces 空间管理
		r.Route("/spaces", func(r chi.Router) {
			r.Get("/", h.ListSpaces)         // 空间列表
			r.Post("/", h.CreateSpace)       // 创建空间
			r.Get("/{id}", h.GetSpace)       // 空间详情
			r.Put("/{id}", h.UpdateSpace)    // 更新空间
			r.Delete("/{id}", h.DeleteSpace) // 删除空间（进回收站）

			// Nodes 节点管理
			r.Route("/{spaceId}/nodes", func(r chi.Router) {
				r.Get("/tree", h.GetNodeTree)    // 节点树
				r.Post("/", h.CreateNode)        // 创建节点
				r.Get("/{id}", h.GetNode)        // 节点详情
				r.Put("/{id}", h.UpdateNode)     // 更新节点
				r.Delete("/{id}", h.DeleteNode)  // 删除节点
				r.Post("/{id}/move", h.MoveNode) // 移动节点
				r.Put("/{id}/sort", h.SortNode)  // 节点排序

				// Node Permission 节点权限
				r.Get("/{id}/permissions", h.GetNodePermissions)                            // 获取节点权限
				r.Post("/{id}/permissions", h.SetNodePermission)                            // 设置节点权限
				r.Delete("/{id}/permissions/{userId}/{permission}", h.RemoveNodePermission) // 移除节点权限
			})

			// Members 成员管理
			r.Route("/{spaceId}/members", func(r chi.Router) {
				r.Get("/", h.ListMembers)             // 成员列表
				r.Post("/", h.AddMember)              // 添加成员
				r.Delete("/{userId}", h.RemoveMember) // 移除成员
			})

			// 扩展路由 - Spaces（标签、模板、通知、订阅、批量操作）
			r.Post("/tags", extH.CreateTag)             // 创建标签
			r.Get("/tags", extH.ListTags)               // 获取标签列表
			r.Delete("/tags/{id}", extH.DeleteTag)      // 删除标签

			r.Post("/nodes/batch", extH.BatchOperation) // 批量操作（移动/删除）

			r.Post("/templates", extH.CreateTemplate)       // 创建文档模板
			r.Get("/templates", extH.ListTemplates)         // 获取模板列表
			r.Get("/templates/{id}", extH.GetTemplate)      // 获取模板详情
			r.Put("/templates/{id}", extH.UpdateTemplate)   // 更新模板
			r.Delete("/templates/{id}", extH.DeleteTemplate) // 删除模板

			r.Get("/notifications", extH.ListNotifications)                     // 获取通知列表
			r.Put("/notifications/read-all", extH.MarkAllNotificationsAsRead)   // 全部标记已读
			r.Get("/notifications/unread-count", extH.GetUnreadNotificationCount) // 未读通知数
			r.Put("/notifications/{id}/read", extH.MarkNotificationAsRead)      // 标记单条已读

			r.Get("/subscriptions", extH.ListUserSubscriptions) // 获取用户订阅列表
		})

		// Documents 文档管理
		r.Route("/documents", func(r chi.Router) {
			r.Post("/preview", h.PreviewMarkdown)        // Markdown 实时预览
			r.Post("/nodes/{nodeId}", h.CreateDocument) // 创建文档
			r.Get("/nodes/{nodeId}", h.GetDocument)     // 获取文档
			r.Put("/{id}", h.UpdateDocument)            // 更新文档
			r.Delete("/{id}", h.DeleteDocument)         // 删除文档

			// Revisions 历史版本
			r.Get("/{documentId}/revisions", h.ListRevisions)                      // 版本列表
			r.Get("/{documentId}/revisions/{version}", h.GetRevision)              // 获取指定版本
			r.Post("/{documentId}/revisions/{version}/restore", h.RestoreRevision) // 恢复版本

			// Attachments 附件管理
			r.Post("/attachments", h.CreateAttachment)            // 创建附件
			r.Get("/{documentId}/attachments", h.ListAttachments) // 附件列表
			r.Delete("/attachments/{id}", h.DeleteAttachment)     // 删除附件

			// 扩展路由 - Documents（标签、评论、分享、统计、访问日志）
			r.Post("/{id}/tags/{tagId}", extH.AddDocumentTag)      // 为文档添加标签
			r.Delete("/{id}/tags/{tagId}", extH.RemoveDocumentTag) // 移除文档标签
			r.Get("/{id}/tags", extH.ListDocumentTags)             // 获取文档标签列表

			r.Post("/{id}/comments", extH.CreateComment)   // 创建评论
			r.Get("/{id}/comments", extH.ListComments)     // 获取评论列表
			r.Put("/comments/{id}", extH.UpdateComment)    // 更新评论
			r.Delete("/comments/{id}", extH.DeleteComment) // 删除评论

			r.Post("/{id}/share", extH.CreateShareLink)    // 创建分享链接
			r.Get("/{id}/shares", extH.ListShareLinks)     // 获取分享链接列表
			r.Delete("/shares/{id}", extH.DeleteShareLink) // 删除分享链接

			r.Get("/{id}/stats", extH.GetDocumentStats)       // 获取文档访问统计
			r.Get("/{id}/access-logs", extH.ListAccessLogs)   // 获取文档访问日志

			r.Post("/{id}/subscribe", extH.SubscribeDocument)     // 订阅文档变更通知
			r.Delete("/{id}/subscribe", extH.UnsubscribeDocument) // 取消订阅文档

			r.Post("/{id}/edit-lock", extH.AcquireEditLock)   // 获取编辑锁
			r.Delete("/{id}/edit-lock", extH.ReleaseEditLock) // 释放编辑锁
			r.Put("/{id}/edit-lock", extH.RefreshEditLock)    // 刷新编辑锁
			r.Get("/{id}/edit-lock", extH.GetEditLock)        // 查询编辑锁状态

			r.Post("/{id}/export", extH.ExportDocument) // 导出文档
			r.Post("/{id}/import", extH.ImportDocument) // 导入文档

			r.Get("/{id}/revisions/compare", extH.CompareRevisions) // 版本对比

			r.Post("/{id}/submit-review", extH.SubmitForReview) // 提交审核
			r.Post("/{id}/approve", extH.ApproveDocument)       // 审核通过
			r.Post("/{id}/reject", extH.RejectDocument)         // 审核驳回
			r.Post("/{id}/publish", extH.PublishDocument)       // 发布文档
			r.Post("/{id}/unpublish", extH.UnpublishDocument)   // 取消发布
			r.Post("/{id}/archive", extH.ArchiveDocument)       // 归档文档
			r.Get("/{id}/review-comments", extH.ListReviewComments) // 获取审核评论
		})

		r.Get("/pending-reviews", extH.ListPendingReviews) // 获取待审核列表
	})
}