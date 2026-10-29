package wiki

import (
	"meteorx/internal/middleware"
	"meteorx/internal/modules/wiki/handler"

	"github.com/go-chi/chi/v5"
)

// RegisterRoutes 注册 Wiki 模块所有路由（需登录 + 自动权限校验）。
func RegisterRoutes(r chi.Router, h *handler.WikiHandler, extH *handler.WikiHandlerExtended) {
	r.Route("/wiki", func(r chi.Router) {
		r.Get("/stats", h.GetStats) // Wiki 统计数据
		r.Get("/search", h.Search)  // Wiki 搜索（标题+内容）

		// Trash 回收站
		r.Get("/trash", h.ListTrashItems)                   // 回收站列表
		r.Post("/trash/{id}/restore", h.RestoreTrashItem)   // 从回收站恢复
		r.Delete("/trash/{id}", h.PermanentDeleteTrashItem) // 永久删除

		// 运维：手动触发回收站过期清理（仅平台超级管理员）
		r.With(middleware.RequiresMasterAdmin()).Post("/admin/trash/expire", h.TriggerTrashExpiry)

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
			// 单一来源：与测试共用 extH.RegisterSpaceExtendedRoutes，避免两处路由定义漂移
			extH.RegisterSpaceExtendedRoutes(r)
		})

		// Documents 文档管理
		r.Route("/documents", func(r chi.Router) {
			r.Post("/preview", h.PreviewMarkdown)       // Markdown 实时预览
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
			// 单一来源：与测试共用 extH.RegisterDocumentExtendedRoutes，避免两处路由定义漂移
			extH.RegisterDocumentExtendedRoutes(r)
		})

		r.Get("/pending-reviews", extH.ListPendingReviews) // 获取待审核列表
	})
}
