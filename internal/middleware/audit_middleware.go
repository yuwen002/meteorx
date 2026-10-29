package middleware

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"meteorx/pkg/idgen"
	"meteorx/pkg/security"
	"net/http"
	"strings"
	"time"

	"meteorx/internal/common/contextx"
	"meteorx/internal/modules/audit/dto"
	"meteorx/internal/modules/audit/model"
	"meteorx/internal/modules/audit/service"
	"meteorx/pkg/iplocation"
)

// responseRecorder 包装 ResponseWriter 以捕获响应状态码和body
type responseRecorder struct {
	http.ResponseWriter
	statusCode  int
	body        *bytes.Buffer
	wroteHeader bool
}

func newResponseRecorder(w http.ResponseWriter) *responseRecorder {
	return &responseRecorder{
		ResponseWriter: w,
		statusCode:     http.StatusOK,
		body:           &bytes.Buffer{},
	}
}

// WriteHeader 记录首次写入的响应状态码并透传给底层 ResponseWriter。
func (rr *responseRecorder) WriteHeader(code int) {
	if !rr.wroteHeader {
		rr.statusCode = code
		rr.wroteHeader = true
		rr.ResponseWriter.WriteHeader(code)
	}
}

// Write 将响应体副本缓存以供审计记录，同时透传给底层 ResponseWriter。
func (rr *responseRecorder) Write(b []byte) (int, error) {
	rr.body.Write(b)
	return rr.ResponseWriter.Write(b)
}

// AuditMiddleware 审计日志中间件
// 自动记录所有经过的请求信息
func AuditMiddleware(auditSvc *service.AuditService, ipLocator iplocation.IPLocator) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// 1. 生成 request_id / trace_id，注入 context 和响应头。
			//    trace_id 以 TracingMiddleware 已注入的真实 OTel Trace ID 为准，
			//    仅当无追踪上下文时（OTel 关闭等）才回退到头部解析/生成，
			//    保证审计记录的 trace_id 能直接点进 Jaeger 链路。
			requestID := getRequestID(r)
			ctx := contextx.SetRequestID(r.Context(), requestID)
			traceID := resolveTraceID(r)
			ctx = contextx.SetTraceID(ctx, traceID)
			r = r.WithContext(ctx)

			// 2. 提前设置响应头（WriteHeader 之后不能再 SetHeader，必须在 handler 之前）
			w.Header().Set("X-Request-ID", requestID)
			w.Header().Set("X-Trace-ID", traceID)

			// 读取请求体
			requestBody := readRequestBodyForAudit(r)

			// 包装 ResponseWriter 以捕获响应
			recorder := newResponseRecorder(w)

			// 执行后续处理
			next.ServeHTTP(recorder, r)

			// 计算耗时
			duration := time.Since(start).Milliseconds()

			// 异步记录审计日志（不阻塞响应）
			auditCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			go func() {
				recordAuditLog(auditCtx, auditSvc, ipLocator, r, recorder, requestBody, duration)
			}()
		})
	}
}

// maxAuditBodyBytes 审计保留的请求体上限。审计只用于事后定位“改了什么”，
// 超过这个长度的内容既无阅读价值，也会白白占用内存与数据库行宽。
const maxAuditBodyBytes = 64 << 10

// multipartBodyPlaceholder 用于替代 multipart 正文。
// 文件上传（导入、附件）的正文是二进制与用户明文数据，既无法阅读也不应落库：
// 导入 CSV/XLSX 一旦被执行体采集，文件内容（含密码列）会进审计日志。
const multipartBodyPlaceholder = "[multipart/form-data 请求体不入审计日志]"

// readRequestBodyForAudit 为审计采集请求体副本，并保证后续 handler 仍能读到完整内容。
// multipart 请求直接跳过：既不缓存也不复制，handler 照常流式解析。
func readRequestBodyForAudit(r *http.Request) string {
	if r.Body == nil || r.Body == http.NoBody {
		return ""
	}
	if strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/") {
		return multipartBodyPlaceholder
	}

	bodyBytes, _ := io.ReadAll(io.LimitReader(r.Body, maxAuditBodyBytes+1))
	// 未读完的部分用 MultiReader 拼回，handler 读取到的仍是完整请求体
	r.Body = io.NopCloser(io.MultiReader(bytes.NewReader(bodyBytes), r.Body))
	return string(bodyBytes)
}

// recordAuditLog 记录审计日志
func recordAuditLog(ctx context.Context, auditSvc *service.AuditService, ipLocator iplocation.IPLocator, r *http.Request, recorder *responseRecorder, requestBody string, duration int64) {
	// 获取用户信息
	userID := contextx.GetUserID(r.Context())
	username := "anonymous"
	if userID != "" {
		username = userID
	}
	tenantID := contextx.GetTenantID(r.Context())

	// 解析模块和操作类型
	module, action := parseModuleAndAction(r.URL.Path, r.Method)

	// 脱敏处理
	sanitizedReq := sanitizeBody(requestBody)
	sanitizedResp := sanitizeBody(recorder.body.String())

	// 截断过长的body
	if len(sanitizedReq) > 2000 {
		sanitizedReq = sanitizedReq[:2000] + "..."
	}
	if len(sanitizedResp) > 2000 {
		sanitizedResp = sanitizedResp[:2000] + "..."
	}

	// 判断结果
	result := model.ResultSuccess
	if recorder.statusCode >= 400 {
		result = model.ResultFailure
	}

	// 获取客户端IP（经可信代理解析，防 XFF/X-Real-IP 头部伪造）
	clientIP := security.GetClientIP(r)

	// 解析IP地理位置
	var ipLocation string
	if ipLocator != nil {
		loc, err := ipLocator.Locate(ctx, clientIP)
		if err == nil && loc != nil {
			ipLocation = loc.FullText
		}
	}

	// 解析设备信息
	deviceInfo := ParseDeviceInfo(r.UserAgent())

	// 评估风险等级
	riskLevel := evaluateRiskLevel(r.URL.Path, r.Method, module, action)

	// 自动生成标签
	tags := buildTags(module, action, result, r.Method, r.URL.Path)

	req := dto.CreateAuditLogReq{
		UserID:      userID,
		Username:    username,
		TenantID:    tenantID,
		Module:      module,
		Action:      action,
		Resource:    r.URL.Path,
		Method:      r.Method,
		Path:        r.URL.Path,
		RequestBody: sanitizedReq,
		StatusCode:  recorder.statusCode,
		Result:      result,
		ClientIP:    clientIP,
		IPLocation:  ipLocation,
		UserAgent:   r.UserAgent(),
		DeviceInfo:  deviceInfo,
		Duration:    duration,
		SessionID:   getSessionID(r),
		RequestID:   contextx.GetRequestID(r.Context()),
		TraceID:     contextx.GetTraceID(r.Context()),
		Referer:     r.Referer(),
		RiskLevel:   riskLevel,
		Tags:        tags,
	}

	// 尝试解析错误信息
	if recorder.statusCode >= 400 && len(sanitizedResp) > 0 {
		var errResp map[string]interface{}
		if err := json.Unmarshal([]byte(sanitizedResp), &errResp); err == nil {
			if msg, ok := errResp["message"].(string); ok {
				req.ErrorMessage = msg
			}
		}
	}

	// 忽略健康检查等不需要记录的请求
	if shouldSkipAudit(r.URL.Path) {
		return
	}

	// 使用批量处理器（如果已初始化）
	if GlobalBatchProcessor != nil {
		GlobalBatchProcessor.Add(&req)
	} else {
		_, _ = auditSvc.CreateLog(ctx, req)
	}
}

// parseModuleAndAction 从路径和方法解析模块和操作类型
func parseModuleAndAction(path, method string) (module, action string) {
	// 解析模块
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) >= 2 {
		// 路径格式通常是 api/v1/xxx/...
		if parts[0] == "api" && len(parts) >= 3 {
			module = parts[2]
		} else if len(parts) >= 1 {
			module = parts[0]
		}
	}

	// 解析操作类型
	switch strings.ToUpper(method) {
	case "GET":
		action = model.ActionTypeQuery
	case "POST":
		if strings.Contains(path, "login") {
			action = model.ActionTypeLogin
		} else if strings.Contains(path, "logout") {
			action = model.ActionTypeLogout
		} else {
			action = model.ActionTypeCreate
		}
	case "PUT", "PATCH":
		action = model.ActionTypeUpdate
	case "DELETE":
		action = model.ActionTypeDelete
	default:
		action = model.ActionTypeOther
	}

	return module, action
}

// sensitiveFieldKeywords 命中即脱敏的字段名关键字（小写子串匹配）。
// 按“字段名包含关键字”而非全等匹配，覆盖 password/token/secret 的各种命名变体，
// 并补齐 api_key/access_key/private_key 等密钥类字段——只靠旧的全等清单会整片漏掉。
var sensitiveFieldKeywords = []string{
	"password", "passwd", "pwd",
	"token", "secret", "authorization",
	"api_key", "apikey", "access_key", "private_key", "secret_key",
	"credit_card", "card_no", "cvv",
	"id_card", "idcard",
	"totp", "otp", "mfa",
}

// isSensitiveField 判断字段名是否命中敏感关键字。
func isSensitiveField(key string) bool {
	lower := strings.ToLower(key)
	for _, kw := range sensitiveFieldKeywords {
		if strings.Contains(lower, kw) {
			return true
		}
	}
	return false
}

// sanitizeValue 递归遍历 JSON 值：命中敏感字段名的直接替换为 ***，
// 并继续下钻嵌套对象与数组——避免凭据藏在二层结构里明文进审计库。
func sanitizeValue(v interface{}) interface{} {
	switch val := v.(type) {
	case map[string]interface{}:
		for k, child := range val {
			if isSensitiveField(k) {
				val[k] = "***"
				continue
			}
			val[k] = sanitizeValue(child)
		}
		return val
	case []interface{}:
		for i, child := range val {
			val[i] = sanitizeValue(child)
		}
		return val
	default:
		return v
	}
}

// sanitizeBody 对请求/响应体进行脱敏处理，递归覆盖嵌套结构。
func sanitizeBody(body string) string {
	if body == "" {
		return ""
	}

	// 解析为通用值（而非仅 map）：顶层是数组的 JSON 也要能递归脱敏。
	var data interface{}
	if err := json.Unmarshal([]byte(body), &data); err != nil {
		return body // 不是 JSON，原样返回
	}

	result, err := json.Marshal(sanitizeValue(data))
	if err != nil {
		return body
	}
	return string(result)
}

// shouldSkipAudit 判断是否需要跳过审计记录
func shouldSkipAudit(path string) bool {
	skipPaths := []string{
		"/health",
		"/api/v1/audit/logs", // 避免记录审计日志本身的请求，防止循环
		"/api/v1/audit/stats",
	}

	for _, skip := range skipPaths {
		if strings.HasPrefix(path, skip) {
			return true
		}
	}
	return false
}

// NewAuditLogReq 从 HTTP 请求构建基础的审计日志请求（填充IP、设备、链路关联等公共字段）
// 供登录/注册/重置密码/OAuth 回调等非中间件场景手动记录使用。
func NewAuditLogReq(r *http.Request) dto.CreateAuditLogReq {
	return dto.CreateAuditLogReq{
		Method:     r.Method,
		Path:       r.URL.Path,
		ClientIP:   security.GetClientIP(r),
		UserAgent:  r.UserAgent(),
		DeviceInfo: ParseDeviceInfo(r.UserAgent()),
		Referer:    r.Referer(),
		// 链路关联：手动审计同样携带 trace/request/session，与自动审计同口径接入 Jaeger 关联
		TraceID:   contextx.GetTraceID(r.Context()),
		RequestID: contextx.GetRequestID(r.Context()),
		SessionID: getSessionID(r),
	}
}

// ParseDeviceInfo 从 User-Agent 解析设备信息（导出供外部使用）
func ParseDeviceInfo(userAgent string) string {
	if userAgent == "" {
		return "Unknown"
	}

	// 简化解析，提取主要信息
	var deviceParts []string

	// 检测操作系统
	if strings.Contains(userAgent, "Windows") {
		deviceParts = append(deviceParts, "Windows")
	} else if strings.Contains(userAgent, "Macintosh") || strings.Contains(userAgent, "Mac OS X") {
		deviceParts = append(deviceParts, "macOS")
	} else if strings.Contains(userAgent, "Linux") {
		deviceParts = append(deviceParts, "Linux")
	} else if strings.Contains(userAgent, "Android") {
		deviceParts = append(deviceParts, "Android")
	} else if strings.Contains(userAgent, "iPhone") || strings.Contains(userAgent, "iPad") {
		deviceParts = append(deviceParts, "iOS")
	}

	// 检测浏览器
	if strings.Contains(userAgent, "Chrome") && !strings.Contains(userAgent, "Edg") {
		deviceParts = append(deviceParts, "Chrome")
	} else if strings.Contains(userAgent, "Firefox") {
		deviceParts = append(deviceParts, "Firefox")
	} else if strings.Contains(userAgent, "Safari") && !strings.Contains(userAgent, "Chrome") {
		deviceParts = append(deviceParts, "Safari")
	} else if strings.Contains(userAgent, "Edg") {
		deviceParts = append(deviceParts, "Edge")
	}

	if len(deviceParts) == 0 {
		return userAgent
	}

	return strings.Join(deviceParts, " / ")
}

// evaluateRiskLevel 评估操作风险等级
func evaluateRiskLevel(path, method, module, action string) string {
	// 严重风险：删除操作、权限变更
	if action == model.ActionTypeDelete {
		return model.RiskCritical
	}
	if strings.Contains(path, "permission") || strings.Contains(path, "role") {
		if method == "POST" || method == "PUT" || method == "DELETE" {
			return model.RiskCritical
		}
	}

	// 严重风险：双因素认证状态变更（开启/关闭、重生成恢复码）。
	// 第二因子被关掉或恢复码被重置，往往意味着账号已被接管后的“消除痕迹”，
	// 风险不亚于权限变更，须按 critical 留痕供实时告警捕获。
	isWrite := method == "POST" || method == "PUT" || method == "DELETE"
	if strings.Contains(path, "/2fa/") && isWrite {
		return model.RiskCritical
	}

	// 高风险：密码变更 / 重置（自助改密、管理员重置、忘记密码重置）。
	// 改密同样是账号接管的高信号事件，统一抬到 high。
	if strings.Contains(path, "password") && (method == "POST" || method == "PUT") {
		return model.RiskHigh
	}

	// 高风险：用户管理、租户管理、登录失败
	if module == "user" && (method == "POST" || method == "DELETE") {
		return model.RiskHigh
	}
	if module == "tenant" && (method == "POST" || method == "PUT" || method == "DELETE") {
		return model.RiskHigh
	}
	if action == model.ActionTypeLogin {
		return model.RiskHigh
	}

	// 中风险：更新操作、导出操作
	if action == model.ActionTypeUpdate {
		return model.RiskMedium
	}
	if strings.Contains(path, "export") {
		return model.RiskMedium
	}

	// 低风险：查询操作
	if action == model.ActionTypeQuery {
		return model.RiskLow
	}

	// 默认低风险
	return model.RiskLow
}

// buildTags 根据请求特征自动生成审计日志标签（JSON数组字符串）
func buildTags(module, action, result, method, path string) string {
	tags := []string{}
	if module != "" {
		tags = append(tags, module)
	}
	if action != "" {
		tags = append(tags, action)
	}
	tags = append(tags, strings.ToLower(method))
	if result == model.ResultSuccess {
		tags = append(tags, "success")
	} else {
		tags = append(tags, "failure")
	}
	if action == model.ActionTypeLogin {
		if result == model.ResultSuccess {
			tags = append(tags, "login_success")
		} else {
			tags = append(tags, "login_failed")
		}
	}
	if strings.Contains(path, "upload") {
		tags = append(tags, "upload")
	}
	if strings.Contains(path, "import") {
		tags = append(tags, "import")
	}
	// 批量写（如消息批量已读 / 批量删除）：请求体中的 ID 列表可能因长度被截断，
	// 打上 batch 标签后审计仍能按标签整类筛出来，不必依赖 body 内容。
	if strings.Contains(path, "batch") {
		tags = append(tags, "batch")
	}
	if strings.Contains(path, "download") || strings.Contains(path, "export") {
		tags = append(tags, "download")
	}
	if strings.Contains(path, "password") && (method == "POST" || method == "PUT") {
		tags = append(tags, "password_change")
	}
	// 双因素认证状态变更单独打标签，便于按标签整类检索 2FA 开关/恢复码重生成。
	if strings.Contains(path, "/2fa/") && (method == "POST" || method == "PUT" || method == "DELETE") {
		tags = append(tags, "2fa_change")
	}
	if len(tags) == 0 {
		return ""
	}
	b, _ := json.Marshal(tags)
	return string(b)
}

// getSessionID 从请求中获取会话ID
// 优先顺序: X-Session-ID 请求头 > session_id Cookie > JWT Token 哈希
func getSessionID(r *http.Request) string {
	// 优先从 header 获取
	if sessionID := r.Header.Get("X-Session-ID"); sessionID != "" {
		return sessionID
	}

	// 从 cookie 获取
	if cookie, err := r.Cookie("session_id"); err == nil {
		return cookie.Value
	}

	// 自动从 JWT Token 生成会话ID（同一 Token 映射到同一会话）
	if authHeader := r.Header.Get("Authorization"); authHeader != "" {
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) == 2 && parts[0] == "Bearer" && parts[1] != "" {
			hash := sha256.Sum256([]byte(parts[1]))
			return fmt.Sprintf("ses_%x", hash[:8])
		}
	}

	return ""
}

// getRequestID 从请求中获取请求ID
// 优先顺序: X-Request-ID > X-Correlation-ID > 自动生成 UUID
func getRequestID(r *http.Request) string {
	// 优先从 header 获取
	if requestID := r.Header.Get("X-Request-ID"); requestID != "" {
		return requestID
	}
	if requestID := r.Header.Get("X-Correlation-ID"); requestID != "" {
		return requestID
	}

	// 自动生成请求ID
	return idgen.NewUUID()
}

// resolveTraceID 解析当前请求应关联的链路追踪ID：
// 优先取 TracingMiddleware 已注入的真实 OTel Trace ID，
// 无追踪上下文时（OTel 关闭等）才回退到请求头/生成。
func resolveTraceID(r *http.Request) string {
	if traceID := contextx.GetTraceID(r.Context()); traceID != "" {
		return traceID
	}
	return getTraceID(r)
}

// getTraceID 在无追踪上下文时从请求头回退获取链路追踪ID
// 优先顺序: Traceparent 的 trace-id 段 > X-Trace-ID > 自动生成 UUID
func getTraceID(r *http.Request) string {
	// 标准 W3C traceparent 优先：只取其中的 trace-id 段（而非整串，避免超列宽且无法检索）
	if traceID := parseTraceparentTraceID(r.Header.Get("Traceparent")); traceID != "" {
		return traceID
	}
	if traceID := r.Header.Get("X-Trace-ID"); traceID != "" {
		return traceID
	}

	// 无上游追踪时的占位 ID（此时并无 Jaeger 链路可关联，仅用于日志内自身区分）
	return idgen.NewUUID()
}

// parseTraceparentTraceID 从 W3C traceparent（version-traceId-parentId-flags）中抽取 traceId。
// 仅当 traceId 为 32 位十六进制且非全零时返回，否则返回空串。
func parseTraceparentTraceID(traceparent string) string {
	if traceparent == "" {
		return ""
	}
	parts := strings.Split(traceparent, "-")
	if len(parts) < 4 {
		return ""
	}
	traceID := parts[1]
	if len(traceID) != 32 || strings.Trim(traceID, "0") == "" {
		return ""
	}
	return traceID
}
