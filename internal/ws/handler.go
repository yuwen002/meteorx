package ws

import (
	"net/http"

	"meteorx/internal/common/contextx"
	"meteorx/pkg/logger"
)

// Handler WebSocket HTTP 处理器
type Handler struct {
	hub *Hub
}

// NewHandler 创建 WebSocket 处理器
func NewHandler(hub *Hub) *Handler {
	return &Handler{hub: hub}
}

// ServeWS 处理 WebSocket 连接请求（需要认证）
func (h *Handler) ServeWS(w http.ResponseWriter, r *http.Request) {
	userID := contextx.GetUserID(r.Context())
	if userID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// 可选：限制同一用户的最大连接数
	if h.hub.CountByUser(userID) > 10 {
		logger.Warnf("[WS] Too many connections for user=%s: %d", userID, h.hub.CountByUser(userID))
		http.Error(w, "Too many connections", http.StatusTooManyRequests)
		return
	}

	ServeWS(h.hub, w, r, userID)
}

// GetHub 获取 Hub 实例
func (h *Handler) GetHub() *Hub {
	return h.hub
}

// Health 检查 WebSocket 服务状态
func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status":"ok","connections":` + string(rune(h.hub.Count())) + `}`))
}