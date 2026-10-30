package service

import (
	"testing"

	"meteorx/internal/modules/wiki/model"
)

// TestMapActionToSpaceLevel 锁定业务动作到 Space 权限矩阵键的映射。
// review/publish 若原样返回（矩阵中无同名键），审核与发布链路会对所有成员恒定 403。
func TestMapActionToSpaceLevel(t *testing.T) {
	tests := []struct {
		action string
		want   string
	}{
		{"read", "node:read"},
		{"create", "node:create"},
		{"update", "node:update"},
		{"delete", "node:delete"},
		{"review", "document:review"},
		{"publish", "document:publish"},
		{"document:review", "document:review"}, // 已是矩阵键时保持原样
		{"document:publish", "document:publish"},
	}

	for _, tt := range tests {
		t.Run(tt.action, func(t *testing.T) {
			if got := mapActionToSpaceLevel(tt.action); got != tt.want {
				t.Errorf("mapActionToSpaceLevel(%q) = %q, want %q", tt.action, got, tt.want)
			}
		})
	}
}

// TestSpaceRolePermissions_ActionCoverage 校验映射结果都真实存在于矩阵中，
// 并锁定各角色在审核/发布上的能力边界（Owner/Admin 全量，Editor 仅 review，Viewer 只读）。
func TestSpaceRolePermissions_ActionCoverage(t *testing.T) {
	actions := []string{"read", "create", "update", "delete", "review", "publish"}

	// 期望矩阵：action -> 是否应被授予
	expectations := map[string]map[string]bool{
		model.SpaceRoleOwner: {
			"read": true, "create": true, "update": true, "delete": true, "review": true, "publish": true,
		},
		model.SpaceRoleAdmin: {
			"read": true, "create": true, "update": true, "delete": true, "review": true, "publish": true,
		},
		model.SpaceRoleEditor: {
			"read": true, "create": true, "update": true, "delete": false, "review": true, "publish": false,
		},
		model.SpaceRoleViewer: {
			"read": true, "create": false, "update": false, "delete": false, "review": false, "publish": false,
		},
	}

	for role, wantPerms := range expectations {
		perms, exists := spaceRolePermissions[role]
		if !exists {
			t.Fatalf("角色 %q 未在 spaceRolePermissions 中定义", role)
		}

		for _, action := range actions {
			key := mapActionToSpaceLevel(action)
			if _, defined := spaceRolePermissions[model.SpaceRoleOwner][key]; !defined {
				t.Fatalf("动作 %q 映射出的权限键 %q 不在权限矩阵中（Owner 也查不到，必然 403）", action, key)
			}

			if got := perms[key]; got != wantPerms[action] {
				t.Errorf("角色 %q 的动作 %q（键 %q）权限 = %v, want %v", role, action, key, got, wantPerms[action])
			}
		}
	}
}
