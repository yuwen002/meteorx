package service

import (
	"context"

	"meteorx/internal/modules/wiki/dto"
	"meteorx/internal/modules/wiki/repository"
	db "meteorx/internal/pkg/db"
)

// WikiService Wiki 模块服务接口
type WikiService interface {
	// ========== WikiSpace ==========
	// CreateSpace 创建 Wiki Space，并将创建者自动添加为 Owner
	CreateSpace(ctx context.Context, tenantID string, userID string, req *dto.CreateWikiSpaceReq) (*dto.WikiSpaceResp, error)
	// GetSpace 获取 Space 详情
	GetSpace(ctx context.Context, id string, userID string) (*dto.WikiSpaceResp, error)
	// ListSpaces 列出用户可访问的 Spaces（分页，keyword 按名称模糊过滤）
	ListSpaces(ctx context.Context, tenantID string, userID string, keyword string, page, pageSize int) ([]*dto.WikiSpaceResp, int64, error)
	// UpdateSpace 更新 Space 信息（userID 作为权限校验主体）
	UpdateSpace(ctx context.Context, id string, userID string, req *dto.UpdateWikiSpaceReq) (*dto.WikiSpaceResp, error)
	// DeleteSpace 删除 Space（进入回收站）
	DeleteSpace(ctx context.Context, id string, tenantID string) error

	// ========== WikiNode ==========
	// CreateNode 创建节点（文件夹或文档）
	CreateNode(ctx context.Context, spaceID string, userID string, req *dto.CreateWikiNodeReq) (*dto.WikiNodeResp, error)
	// GetNode 获取节点详情
	GetNode(ctx context.Context, id string) (*dto.WikiNodeResp, error)
	// GetNodeTree 获取 Space 下的完整节点树（已排序）
	GetNodeTree(ctx context.Context, spaceID string) ([]*dto.WikiNodeTreeResp, error)
	// UpdateNode 更新节点（支持移动、重命名、排序）
	UpdateNode(ctx context.Context, id string, req *dto.UpdateWikiNodeReq) (*dto.WikiNodeResp, error)
	// DeleteNode 删除节点及其子树（进入回收站 + 级联删除）
	DeleteNode(ctx context.Context, id string) error
	// MoveNode 移动节点到新父节点（含安全校验）
	MoveNode(ctx context.Context, id string, newParentID string, userID string) error
	// SortNode 设置节点排序值
	SortNode(ctx context.Context, id string, sort int, userID string) error

	// ========== Document ==========
	// CreateDocument 创建文档（支持 Markdown 渲染）
	CreateDocument(ctx context.Context, nodeID string, userID string, req *dto.CreateDocumentReq) (*dto.DocumentResp, error)
	// GetDocument 获取文档内容
	GetDocument(ctx context.Context, nodeID string, userID string) (*dto.DocumentResp, error)
	// UpdateDocument 更新文档（事务保证 + Revision 自动创建）
	UpdateDocument(ctx context.Context, id string, userID string, req *dto.UpdateDocumentReq) (*dto.DocumentResp, error)
	// DeleteDocument 删除文档（进入回收站 + 级联删除附件）
	DeleteDocument(ctx context.Context, id string) error

	// ========== Document Revision ==========
	// ListRevisions 列出文档的所有历史版本
	ListRevisions(ctx context.Context, documentID string) ([]*dto.DocumentRevisionResp, error)
	// GetRevision 获取指定版本的历史内容
	GetRevision(ctx context.Context, documentID string, version int) (*dto.DocumentRevisionResp, error)
	// RestoreRevision 恢复到指定版本（自动保存当前版本为新 Revision）
	RestoreRevision(ctx context.Context, documentID string, version int, userID string) (*dto.DocumentResp, error)

	// ========== Space Members ==========
	// AddMember 添加 Space 成员
	AddMember(ctx context.Context, spaceID string, req *dto.WikiSpaceMemberReq) (*dto.WikiSpaceMemberResp, error)
	// RemoveMember 移除成员
	RemoveMember(ctx context.Context, spaceID, userID string) error
	// ListMembers 列出 Space 成员列表
	ListMembers(ctx context.Context, spaceID string) ([]*dto.WikiSpaceMemberResp, error)

	// ========== Node Permissions ==========
	// SetNodePermission 设置节点级别权限
	SetNodePermission(ctx context.Context, nodeID string, req *dto.SetNodePermissionReq) (*dto.NodePermissionResp, error)
	// GetNodePermissions 获取节点的所有权限配置
	GetNodePermissions(ctx context.Context, nodeID string) ([]*dto.NodePermissionResp, error)
	// RemoveNodePermission 移除节点级别权限
	RemoveNodePermission(ctx context.Context, nodeID, userID, permission string) error

	// ========== Permission Check ==========
	// CheckSpacePermission 检查 Space 级别权限
	CheckSpacePermission(ctx context.Context, spaceID, userID, action string) error
	// CheckNodePermission 检查 Node 级别权限（含继承 + Node 补充授权）
	CheckNodePermission(ctx context.Context, nodeID, userID, action string) error
	// CheckDocumentPermission 检查 Document 权限（基于 Node）
	CheckDocumentPermission(ctx context.Context, documentID, userID, action string) error
	// GetEffectivePermissions 获取用户在 Space 中的有效权限映射
	GetEffectivePermissions(ctx context.Context, spaceID, userID string) map[string]bool

	// ========== Trash ==========
	// ListTrashItems 列出回收站项目（分页）
	ListTrashItems(ctx context.Context, tenantID string, spaceID string, itemType string, page, pageSize int) ([]*dto.TrashItemResp, int64, error)
	// RestoreTrashItem 从回收站恢复项目
	RestoreTrashItem(ctx context.Context, id string, userID string) error
	// PermanentDeleteTrashItem 永久删除回收站项目
	PermanentDeleteTrashItem(ctx context.Context, id string, userID string) error

	// ========== Search ==========
	// Search 搜索 Wiki（标题 + 内容，自动过滤无权限内容）
	Search(ctx context.Context, tenantID string, userID string, query string, spaceID string, page, pageSize int) ([]*dto.SearchResultResp, int64, error)

	// ========== Attachment ==========
	// CreateAttachment 创建文档附件
	CreateAttachment(ctx context.Context, userID string, req *dto.CreateAttachmentReq) (*dto.AttachmentResp, error)
	// ListAttachments 列出文档的所有附件
	ListAttachments(ctx context.Context, documentID string, userID string) ([]*dto.AttachmentResp, error)
	// DeleteAttachment 删除附件
	DeleteAttachment(ctx context.Context, id string, userID string) error

	// ========== Rendering / Uploads ==========
	// SetUploadSigner 注入 /uploads 静态资源签名参数（文档内嵌图片动态签发，仅模块初始化时调用）
	SetUploadSigner(uploadBaseURL, signKey string)
	// RenderPreview 将 Markdown 渲染为安全 HTML（编辑器实时预览，上传图片地址自动重签）
	RenderPreview(ctx context.Context, content, format string) (string, error)

	// ========== Stats ==========
	// GetStats 获取 Wiki 统计数据
	GetStats(ctx context.Context, tenantID string) (*dto.WikiStatsResp, error)
}

// wikiService Wiki 服务实现
type wikiService struct {
	repo        repository.WikiRepository
	tx          *db.TxManager
	markdownSvc MarkdownService

	// uploadBaseURL 对外访问 /uploads 的基础地址（如 http://host:8081/uploads）；
	// uploadSignKey 与文件存储一致的上传签名密钥。两者为空时禁用文档图片动态重签。
	uploadBaseURL string
	uploadSignKey string
}

// NewWikiService 创建 Wiki Service 实例
func NewWikiService(repo repository.WikiRepository, tx *db.TxManager) WikiService {
	return &wikiService{
		repo:        repo,
		tx:          tx,
		markdownSvc: NewMarkdownService(),
	}
}

// GetStats 获取 Wiki 统计数据（Space、Node、Document 数量及浏览数）
func (s *wikiService) GetStats(ctx context.Context, tenantID string) (*dto.WikiStatsResp, error) {
	stats, err := s.repo.GetWikiStats(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	return &dto.WikiStatsResp{
		TotalSpaces:    stats.TotalSpaces,
		TotalNodes:     stats.TotalNodes,
		TotalDocuments: stats.TotalDocuments,
		TotalViews:     stats.TotalViews,
	}, nil
}
