// Package chat 协作即时消息模块入口。
// 首版提供同租户内 1:1 私聊：发起/复用会话、发送消息（落库 + WS 实时投递）、
// 会话列表、历史消息分页与已读标记。路由挂载于受保护分组（需登录），访问控制以会话成员归属为准。
package chat

import (
	"meteorx/internal/modules/chat/handler"
	"meteorx/internal/modules/chat/repository"
	"meteorx/internal/modules/chat/service"
	"meteorx/internal/ws"

	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"
)

// InitModule 装配聊天依赖并注册需登录的聊天路由。
// hub 可为 nil（WS 未启用时发送仅落库、不做实时推送，用户下次拉取历史即可补上）。
func InitModule(r chi.Router, db *gorm.DB, hub *ws.Hub) {
	repo := repository.NewChatRepository(db)
	dir := repository.NewUserDirectory(db)
	svc := service.NewChatService(repo, dir, hub)
	h := handler.NewChatHandler(svc)
	RegisterRoutes(r, h)
}

// RegisterRoutes 注册聊天路由（需登录）。
// 静态段 /conversations/direct 必须先于 /conversations/{id} 注册，避免路径参数吞掉静态段。
func RegisterRoutes(r chi.Router, h *handler.ChatHandler) {
	r.Route("/chat", func(r chi.Router) {
		r.Post("/conversations/direct", h.OpenDirect)
		r.Post("/conversations/group", h.CreateGroup)
		r.Get("/conversations", h.ListConversations)
		r.Put("/conversations/{id}", h.RenameGroup)
		r.Get("/conversations/{id}/messages", h.ListMessages)
		r.Post("/conversations/{id}/messages", h.SendMessage)
		r.Delete("/conversations/{id}/messages/{msgId}", h.RecallMessage)
		r.Get("/conversations/{id}/members", h.ListMembers)
		r.Post("/conversations/{id}/members", h.AddMembers)
		r.Delete("/conversations/{id}/members/{userId}", h.RemoveMember)
		r.Post("/conversations/{id}/leave", h.LeaveGroup)
		r.Post("/conversations/{id}/read", h.MarkRead)
		r.Post("/conversations/{id}/typing", h.Typing)
		r.Get("/presence", h.Presence)
	})
}
