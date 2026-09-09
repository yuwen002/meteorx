package middleware

import (
	"context"
	"net/http"
	"strconv"
	"strings"

	"meteorx/internal/common/response"
	"meteorx/pkg/logger"
)

// ResourcePermissionChecker 资源级权限检查器接口
type ResourcePermissionChecker interface {
	// CheckResourcePermission 检查用户是否有权操作指定资源
	CheckResourcePermission(ctx context.Context, userID, resourceType, resourceID, action string) (bool, error)
	// GetResourceOwnerID 获取资源的拥有者 ID
	GetResourceOwnerID(ctx context.Context, resourceType, resourceID string) (string, error)
}

// ResourcePermission 资源级权限中间件
// 用于实现行级/文档级的细粒度权限控制
func ResourcePermission(checker ResourcePermissionChecker, resourceType string, action string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID := logger.GetUserID(r.Context())
			if userID == "" {
				response.Fail(w, http.StatusForbidden, "未认证")
				return
			}

			// 超级管理员直接放行
			role := logger.GetRequestID(r.Context())
			if role == "superadmin" {
				next.ServeHTTP(w, r)
				return
			}

			// 从 URL 路径或查询参数中获取资源 ID
			resourceID := extractResourceID(r)
			if resourceID == "" {
				// 如果是创建操作，可能还没有资源 ID
				if action == "create" {
					next.ServeHTTP(w, r)
					return
				}
				response.Fail(w, http.StatusBadRequest, "缺少资源 ID")
				return
			}

			// 检查资源权限
			allowed, err := checker.CheckResourcePermission(r.Context(), userID, resourceType, resourceID, action)
			if err != nil {
				logger.Ctx(r.Context()).Error("Resource permission check failed",
					"error", err,
					"resource_type", resourceType,
					"resource_id", resourceID,
					"action", action,
				)
				response.Fail(w, http.StatusInternalServerError, "权限检查失败")
				return
			}

			if !allowed {
				logger.Ctx(r.Context()).Warn("Resource permission denied",
					"user_id", userID,
					"resource_type", resourceType,
					"resource_id", resourceID,
					"action", action,
				)
				response.Fail(w, http.StatusForbidden, "无权操作此资源")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// extractResourceID 从请求中提取资源 ID
func extractResourceID(r *http.Request) string {
	// 尝试从 URL 路径参数中获取（Chi 路由）
	if id := r.PathValue("id"); id != "" {
		return id
	}

	// 尝试从查询参数中获取
	if id := r.URL.Query().Get("id"); id != "" {
		return id
	}

	// 尝试从常见的路径段中提取
	path := strings.Trim(r.URL.Path, "/")
	segments := strings.Split(path, "/")

	// 查找数字或 UUID 格式的段
	for _, seg := range segments {
		if isValidResourceID(seg) {
			return seg
		}
	}

	return ""
}

// isValidResourceID 判断是否为有效的资源 ID
func isValidResourceID(s string) bool {
	if s == "" {
		return false
	}

	// 允许 UUID 格式
	if len(s) == 36 && strings.Contains(s, "-") {
		return true
	}

	// 允许纯数字
	if _, err := strconv.Atoi(s); err == nil {
		return true
	}

	// 允许字母数字组合（长度 >= 8）
	if len(s) >= 8 {
		for _, c := range s {
			if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '-') {
				return false
			}
		}
		return true
	}

	return false
}

// OwnerOnly 仅资源拥有者可操作中间件
func OwnerOnly(checker ResourcePermissionChecker, resourceType string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID := logger.GetUserID(r.Context())
			if userID == "" {
				response.Fail(w, http.StatusForbidden, "未认证")
				return
			}

			resourceID := extractResourceID(r)
			if resourceID == "" {
				response.Fail(w, http.StatusBadRequest, "缺少资源 ID")
				return
			}

			// 获取资源拥有者 ID
			ownerID, err := checker.GetResourceOwnerID(r.Context(), resourceType, resourceID)
			if err != nil {
				logger.Ctx(r.Context()).Error("Failed to get resource owner",
					"error", err,
					"resource_type", resourceType,
					"resource_id", resourceID,
				)
				response.Fail(w, http.StatusInternalServerError, "权限检查失败")
				return
			}

			// 检查是否为资源拥有者
			if ownerID != userID {
				logger.Ctx(r.Context()).Warn("Not resource owner",
					"user_id", userID,
					"owner_id", ownerID,
					"resource_type", resourceType,
					"resource_id", resourceID,
				)
				response.Fail(w, http.StatusForbidden, "仅资源拥有者可操作")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}