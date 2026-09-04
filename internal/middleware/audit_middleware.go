package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
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

func (rr *responseRecorder) WriteHeader(code int) {
	if !rr.wroteHeader {
		rr.statusCode = code
		rr.wroteHeader = true
		rr.ResponseWriter.WriteHeader(code)
	}
}

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

			// 读取请求体
			var requestBody string
			if r.Body != nil && r.Body != http.NoBody {
				bodyBytes, _ := io.ReadAll(r.Body)
				requestBody = string(bodyBytes)
				// 重新设置 body 供后续处理器读取
				r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
			}

			// 包装 ResponseWriter 以捕获响应
			recorder := newResponseRecorder(w)

			// 执行后续处理
			next.ServeHTTP(recorder, r)

			// 计算耗时
			duration := time.Since(start).Milliseconds()

			// 异步记录审计日志（不阻塞响应）
			// 使用带超时的 context，避免服务器关闭时 goroutine 泄漏
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			go func() {
				recordAuditLog(ctx, auditSvc, ipLocator, r, recorder, requestBody, duration)
			}()
		})
	}
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

	// 获取客户端IP
	clientIP := r.RemoteAddr
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		clientIP = strings.Split(forwarded, ",")[0]
	} else if realIP := r.Header.Get("X-Real-IP"); realIP != "" {
		clientIP = realIP
	}

	// 解析IP地理位置
	var ipLocation string
	if ipLocator != nil {
		loc, err := ipLocator.Locate(ctx, clientIP)
		if err == nil && loc != nil {
			ipLocation = loc.FullText
		}
	}

	// 解析设备信息
	deviceInfo := parseDeviceInfo(r.UserAgent())

	// 评估风险等级
	riskLevel := evaluateRiskLevel(r.URL.Path, r.Method, module, action)

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
		RequestID:   getRequestID(r),
		TraceID:     getTraceID(r),
		Referer:     r.Referer(),
		RiskLevel:   riskLevel,
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

// sanitizeBody 对请求/响应体进行脱敏处理
func sanitizeBody(body string) string {
	if body == "" {
		return ""
	}

	// 尝试解析 JSON
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(body), &data); err != nil {
		return body // 不是 JSON，原样返回
	}

	// 敏感字段脱敏
	sensitiveFields := []string{"password", "token", "secret", "authorization", "credit_card", "id_card"}
	for _, field := range sensitiveFields {
		if _, exists := data[field]; exists {
			data[field] = "***"
		}
	}

	result, _ := json.Marshal(data)
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

// parseDeviceInfo 从 User-Agent 解析设备信息
func parseDeviceInfo(userAgent string) string {
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

// getSessionID 从请求中获取会话ID
func getSessionID(r *http.Request) string {
	// 优先从 header 获取
	if sessionID := r.Header.Get("X-Session-ID"); sessionID != "" {
		return sessionID
	}

	// 从 cookie 获取
	if cookie, err := r.Cookie("session_id"); err == nil {
		return cookie.Value
	}

	return ""
}

// getRequestID 从请求中获取请求ID
func getRequestID(r *http.Request) string {
	// 优先从 header 获取
	if requestID := r.Header.Get("X-Request-ID"); requestID != "" {
		return requestID
	}
	if requestID := r.Header.Get("X-Correlation-ID"); requestID != "" {
		return requestID
	}

	return ""
}

// getTraceID 从请求中获取链路追踪ID
func getTraceID(r *http.Request) string {
	// 优先从 header 获取
	if traceID := r.Header.Get("X-Trace-ID"); traceID != "" {
		return traceID
	}
	if traceID := r.Header.Get("Traceparent"); traceID != "" {
		return traceID
	}

	return ""
}
