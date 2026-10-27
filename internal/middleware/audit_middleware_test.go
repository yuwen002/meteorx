package middleware

import (
	"bytes"
	"context"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"meteorx/internal/common/contextx"
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

// TestReadRequestBodyForAudit_SkipsMultipart 导入/附件上传的正文不采集：
// 文件内容（含密码列）不能进审计日志，同时 handler 必须仍能拿到完整表单。
func TestReadRequestBodyForAudit_SkipsMultipart(t *testing.T) {
	body := &bytes.Buffer{}
	w := multipart.NewWriter(body)
	part, err := w.CreateFormFile("file", "users.csv")
	require.NoError(t, err)
	_, err = part.Write([]byte("username,password\nalice,S3cret!\n"))
	require.NoError(t, err)
	require.NoError(t, w.Close())

	r := httptest.NewRequest(http.MethodPost, "/api/v1/users/import", bytes.NewReader(body.Bytes()))
	r.Header.Set("Content-Type", w.FormDataContentType())

	got := readRequestBodyForAudit(r)
	require.Equal(t, multipartBodyPlaceholder, got)
	require.NotContains(t, got, "S3cret!")

	// handler 侧仍要能正常解析出文件与内容
	require.NoError(t, r.ParseMultipartForm(1<<20))
	file, header, err := r.FormFile("file")
	require.NoError(t, err)
	defer func() { _ = file.Close() }()
	require.Equal(t, "users.csv", header.Filename)
	content, err := io.ReadAll(file)
	require.NoError(t, err)
	require.Contains(t, string(content), "alice")
}

// TestReadRequestBodyForAudit_KeepsJSONBody JSON 请求体照旧采集，
// 且超过上限时只截取前 maxAuditBodyBytes 字节，handler 仍能读到完整内容。
func TestReadRequestBodyForAudit_KeepsJSONBody(t *testing.T) {
	payload := `{"username":"alice"}`
	r := httptest.NewRequest(http.MethodPost, "/api/v1/users", strings.NewReader(payload))
	r.Header.Set("Content-Type", "application/json")
	require.Equal(t, payload, readRequestBodyForAudit(r))

	rest, err := io.ReadAll(r.Body)
	require.NoError(t, err)
	require.Equal(t, payload, string(rest), "handler 应能读到完整请求体")
}

func TestReadRequestBodyForAudit_TruncatesLargeBody(t *testing.T) {
	big := strings.Repeat("a", maxAuditBodyBytes+1024)
	r := httptest.NewRequest(http.MethodPost, "/api/v1/users", strings.NewReader(big))

	got := readRequestBodyForAudit(r)
	require.Len(t, got, maxAuditBodyBytes+1)

	rest, err := io.ReadAll(r.Body)
	require.NoError(t, err)
	// handler 从头部开始读，拿到的仍是完整的原始请求体
	require.Equal(t, len(big), len(rest), "handler 应能读到完整请求体")
	require.True(t, strings.HasPrefix(string(rest), got))
	require.Equal(t, big, string(rest))
}

// TestSanitizeBody_MasksPasswordVariants 密码字段的各种命名变体都要被脱敏。
func TestSanitizeBody_MasksPasswordVariants(t *testing.T) {
	body := `{"old_password":"a","NewPassword":"b","initial_password":"c","refresh_token":"d","username":"alice"}`
	out := sanitizeBody(body)

	require.NotContains(t, out, `"a"`)
	require.NotContains(t, out, `"b"`)
	require.NotContains(t, out, `"c"`)
	require.NotContains(t, out, `"d"`)
	require.Contains(t, out, "alice")
}

// TestBuildTags_ImportMarker 批量导入需要 import 标签，便于审计按类型检索。
func TestBuildTags_ImportMarker(t *testing.T) {
	tags := buildTags("users", model.ActionTypeCreate, model.ResultSuccess, "POST", "/api/v1/users/import")
	require.Contains(t, tags, `"import"`)
}

// TestResolveTraceID_PrefersContextTrace 审计必须优先复用 TracingMiddleware 注入的真实
// OTel trace id，即使请求头带了另一个值，也不得覆盖——否则落库的 trace_id 与链路对不上。
func TestResolveTraceID_PrefersContextTrace(t *testing.T) {
	const realTrace = "0af7651916cd43dd8448eb211c80319c"
	ctx := contextx.SetTraceID(context.Background(), realTrace)
	r := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil).WithContext(ctx)
	r.Header.Set("X-Trace-ID", "deadbeef") // 干扰头，应被忽略

	require.Equal(t, realTrace, resolveTraceID(r))
}

// TestResolveTraceID_FallsBackToTraceparent 无追踪上下文时，回退只取 traceparent
// 中的 trace-id 段（而非整串），保证存入 trace_id 列（size:50）的是可检索的 32 位十六进制。
func TestResolveTraceID_FallsBackToTraceparent(t *testing.T) {
	const tid = "12345678901234567890123456789012"
	r := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
	r.Header.Set("Traceparent", "00-"+tid+"-00f067aa0ba902b7-01")

	require.Equal(t, tid, resolveTraceID(r))
}

// TestParseTraceparentTraceID 校验 W3C traceparent 解析的边界：合法取段、非法/全零返回空。
func TestParseTraceparentTraceID(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"valid", "00-0af7651916cd43dd8448eb211c80319c-00f067aa0ba902b7-01", "0af7651916cd43dd8448eb211c80319c"},
		{"all-zero-trace", "00-00000000000000000000000000000000-00f067aa0ba902b7-01", ""},
		{"too-short", "00-abc-def", ""},
		{"wrong-length", "00-0af7651916cd43dd-00f067aa0ba902b7-01", ""},
		{"empty", "", ""},
	}
	for _, c := range cases {
		require.Equal(t, c.want, parseTraceparentTraceID(c.in), c.name)
	}
}
