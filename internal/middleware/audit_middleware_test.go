package middleware

import (
	"testing"

	"meteorx/internal/modules/audit/model"

	"github.com/stretchr/testify/require"
)

// TestParseModuleAndAction_BatchMessages 批量端点仍归入消息模块，
// 动作按 HTTP 方法归一：已读是 update、删除是 delete，资源路径保留 /batch/ 段，
// 审计详情里可直接分辨这次请求是批量还是单条。
func TestParseModuleAndAction_BatchMessages(t *testing.T) {
	cases := []struct {
		path       string
		method     string
		wantModule string
		wantAction string
	}{
		{"/api/v1/messages/batch/read", "PUT", "messages", model.ActionTypeUpdate},
		{"/api/v1/messages/batch/delete", "DELETE", "messages", model.ActionTypeDelete},
		{"/api/v1/messages/read-all", "PUT", "messages", model.ActionTypeUpdate},
	}

	for _, c := range cases {
		module, action := parseModuleAndAction(c.path, c.method)
		require.Equal(t, c.wantModule, module, "path=%s", c.path)
		require.Equal(t, c.wantAction, action, "path=%s", c.path)
	}
}

// TestBuildTags_BatchMarker 批量请求必须带上 batch 标签：
// handler 的入参上限是 100 个 ID，请求体可能被审计的 2000 字符截断，
// 标签是「按类型筛出全部批量写操作」的唯一可靠依据。
func TestBuildTags_BatchMarker(t *testing.T) {
	tags := buildTags("messages", model.ActionTypeDelete, model.ResultSuccess, "DELETE", "/api/v1/messages/batch/delete")
	require.Contains(t, tags, `"batch"`)
	require.Contains(t, tags, `"delete"`)
	require.Contains(t, tags, `"messages"`)

	// 单条删除不带 batch 标签，两类操作在检索上要能区分
	single := buildTags("messages", model.ActionTypeDelete, model.ResultSuccess, "DELETE", "/api/v1/messages/01HXYZ")
	require.NotContains(t, single, `"batch"`)
}

// TestAuditRiskLevel_BatchDeleteIsCritical 批量删除的影响面比单条更大，
// 风险等级不得因「路径不是 /logs/xxx」而降级。
func TestAuditRiskLevel_BatchDeleteIsCritical(t *testing.T) {
	require.Equal(t, model.RiskCritical,
		evaluateRiskLevel("/api/v1/messages/batch/delete", "DELETE", "messages", model.ActionTypeDelete))

	// 批量已读只是读状态变更，落在 update 的中风险档
	require.Equal(t, model.RiskMedium,
		evaluateRiskLevel("/api/v1/messages/batch/read", "PUT", "messages", model.ActionTypeUpdate))
}

// TestShouldSkipAudit_BatchMessagesKept 批量写操作必须进审计；
// 跳过规则只针对健康检查与审计接口自身，不能按前缀误伤业务路径。
func TestShouldSkipAudit_BatchMessagesKept(t *testing.T) {
	require.False(t, shouldSkipAudit("/api/v1/messages/batch/delete"))
	require.False(t, shouldSkipAudit("/api/v1/messages/batch/read"))
	require.True(t, shouldSkipAudit("/health"))
	require.True(t, shouldSkipAudit("/api/v1/audit/logs"))
}
