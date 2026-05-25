package response

import (
	"encoding/json"
	"net/http"
)

type Response struct {
	Code    int         `json:"code"`    // 业务状态码
	Data    interface{} `json:"data"`    // 数据
	Message string      `json:"message"` // 消息
}

// JSON 基础发送方法，减少代码重复
func JSON(w http.ResponseWriter, httpStatus int, code int, message string, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(httpStatus)

	resp := Response{
		Code:    code,
		Data:    data,
		Message: message,
	}

	// 实际项目中可以增加对 Encode 错误的处理
	_ = json.NewEncoder(w).Encode(resp)
}

// Success 成功响应
func Success(w http.ResponseWriter, data interface{}) {
	JSON(w, http.StatusOK, 0, "success", data)
}

// Fail 业务失败（HTTP 状态码通常也是 200，靠 code 区分）
func Fail(w http.ResponseWriter, code int, message string) {
	JSON(w, http.StatusOK, code, message, nil)
}

// BadRequest 客户端参数错误 (HTTP 400)
func BadRequest(w http.ResponseWriter, message string) {
	JSON(w, http.StatusBadRequest, 400, message, nil)
}

// Unauthorized 未授权 (HTTP 401)
func Unauthorized(w http.ResponseWriter, message string) {
	JSON(w, http.StatusUnauthorized, 401, message, nil)
}

// InternalError 服务器内部错误 (HTTP 500)
func InternalError(w http.ResponseWriter, message string) {
	if message == "" {
		message = "Internal Server Error"
	}
	JSON(w, http.StatusInternalServerError, 500, message, nil)
}
