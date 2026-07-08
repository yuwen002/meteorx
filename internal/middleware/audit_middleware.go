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
func AuditMiddleware(auditSvc *service.AuditService) func(http.Handler) http.Handler {
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
			go func() {
				recordAuditLog(auditSvc, r, recorder, requestBody, duration)
			}()
		})
	}
}

// recordAuditLog 记录审计日志
func recordAuditLog(auditSvc *service.AuditService, r *http.Request, recorder *responseRecorder, requestBody string, duration int64) {
	ctx := context.Background()

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
	}

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
		UserAgent:   r.UserAgent(),
		Duration:    duration,
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

	_, _ = auditSvc.CreateLog(ctx, req)
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