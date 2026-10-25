// Package message 站内消息（收件箱）模块入口。
// 提供消息的落库、查询、已读与删除能力，并作为 notify 的 InboxWriter
// 承接「实时推送之外的补推」：在线时由 WebSocket 推送角标，离线时重新登录拉未读。
package message

import (
	"meteorx/internal/modules/message/handler"
	"meteorx/internal/modules/message/repository"
	"meteorx/internal/modules/message/service"
	"meteorx/internal/ws"

	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"
)

// NewService 装配站内消息服务。
// hub 可为 nil（WS 关闭时不推送角标，收件箱读写不受影响）。
func NewService(db *gorm.DB, hub *ws.Hub) *service.MessageService {
	return service.NewMessageService(
		repository.NewMessageRepository(db),
		repository.NewUserDirectory(db),
		hub,
	)
}

// InitModule 注册站内消息路由并返回服务实例（供 bootstrap 装配到通知管理器）。
// 站内消息按「收件人」隔离，访问控制以数据归属为准，因此不挂 RBAC 权限码。
func InitModule(r chi.Router, db *gorm.DB, hub *ws.Hub) *service.MessageService {
	svc := NewService(db, hub)
	h := handler.NewMessageHandler(svc)
	RegisterRoutes(r, h)
	return svc
}

// RegisterRoutes 注册收件箱路由（需登录）。
// 静态段（/unread-count、/read-all）先于 /{id} 注册，避免路径参数吞掉静态段。
func RegisterRoutes(r chi.Router, h *handler.MessageHandler) {
	r.Route("/messages", func(r chi.Router) {
		r.Get("/unread-count", h.UnreadCount)
		r.Put("/read-all", h.MarkAllRead)
		r.Get("/", h.List)
		r.Put("/{id}/read", h.MarkRead)
		r.Delete("/{id}", h.Delete)
	})
}

// Migrate 执行站内消息表迁移（由 bootstrap 统一调度）。
func Migrate(db *gorm.DB) error {
	return repository.AutoMigrate(db)
}
