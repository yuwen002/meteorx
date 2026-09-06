package wiki

import (
	"meteorx/internal/config"
	"meteorx/internal/modules/wiki/handler"
	"meteorx/internal/modules/wiki/repository"
	"meteorx/internal/modules/wiki/service"

	db "meteorx/internal/pkg/db"

	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"
)

// InitModule 初始化 Wiki 模块，注册路由与依赖
func InitModule(r chi.Router, gormDB *gorm.DB, tx *db.TxManager, cfg *config.Config) {
	repo := repository.NewWikiRepository(gormDB)
	svc := service.NewWikiService(repo, tx)
	// 文档内嵌 /uploads 图片签名支持（与文件模块共用访问基址与签名密钥）
	if cfg != nil {
		svc.SetUploadSigner(cfg.File.UploadURL, cfg.JWT.Secret)
	}
	h := handler.NewWikiHandler(svc)

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
		})
	})
}
