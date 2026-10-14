package middleware

import (
	"context"

	"meteorx/internal/common/contextx"
)

// permissionGranted 统一权限判定，供代码级权限中间件复用。
//
// 判定优先级：
//  1. 严格模式——请求来自「带 scope 限制的 API Token」时，所需权限码必须落在令牌收敛后的
//     权限集合（属主权限 ∩ scopes）内；此模式**忽略超管旁路**，即使属主为 superadmin，
//     令牌也只能行使被显式授予的 scope，杜绝长期令牌默认继承全部权限。
//  2. 超级管理员直接放行。
//  3. 普通模式——实时查询用户权限码并匹配。
//
// 返回 (granted, sysErr)：sysErr 非空表示权限查询失败，调用方应映射为 500；
// 严格模式与超管旁路不会产生 sysErr。
func permissionGranted(ctx context.Context, checker PermissionChecker, userID, code string) (bool, error) {
	if perms, strict := contextx.GetScopedPermissions(ctx); strict {
		return containsPermCode(perms, code), nil
	}

	if contextx.HasRole(ctx, "superadmin") {
		return true, nil
	}

	if userID == "" {
		return false, nil
	}

	codes, err := checker.GetUserPermissionCodes(ctx, userID)
	if err != nil {
		return false, err
	}
	return containsPermCode(codes, code), nil
}

// containsPermCode 判断权限码是否存在于集合中。
func containsPermCode(codes []string, code string) bool {
	for _, c := range codes {
		if c == code {
			return true
		}
	}
	return false
}
