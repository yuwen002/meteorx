package ws

import (
	"encoding/json"
	"net/http"
	"strings"

	"meteorx/internal/common/jwt"
	"meteorx/pkg/logger"
)

// Handler WebSocket HTTP 处理器
type Handler struct {
	hub         *Hub
	tokenHelper *jwt.TokenHelper
}

// NewHandler 创建 WebSocket 处理器
func NewHandler(hub *Hub, tokenHelper *jwt.TokenHelper) *Handler {
	return &Handler{hub: hub, tokenHelper: tokenHelper}
}

// ServeWS 处理 WebSocket 连接请求
// 支持两种认证方式：
// 1. Header 中的 Authorization: Bearer token（通过 API 网关认证后）
// 2. URL 查询参数 ?token=xxx（前端 WebSocket 直接连接）
func (h *Handler) ServeWS(w http.ResponseWriter, r *http.Request) {
	// 尝试从查询参数中获取 token
	tokenString := r.URL.Query().Get("token")
	if tokenString == "" {
		// 回退到 Header 中的 Authorization
		authHeader := r.Header.Get("Authorization")
		if authHeader != "" {
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) == 2 && strings.ToLower(parts[0]) == "bearer" {
				tokenString = parts[1]
			}
		}
	}

	if tokenString == "" {
		http.Error(w, "Unauthorized: missing token", http.StatusUnauthorized)
		return
	}

	// 解析 token 获取用户信息
	claims, err := h.tokenHelper.ParseToken(tokenString)
	if err != nil {
		http.Error(w, "Unauthorized: invalid token", http.StatusUnauthorized)
		return
	}

	userID := claims.UserID
	if userID == "" {
		http.Error(w, "Unauthorized: invalid user", http.StatusUnauthorized)
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
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":      "ok",
		"connections": h.hub.Count(),
	})
}

// GetTokenHelper 获取 TokenHelper
func (h *Handler) GetTokenHelper() *jwt.TokenHelper {
	return h.tokenHelper
}