// Package ai 提供可跨模块复用的通用 AI 能力抽象层。
// 以 OpenAI 兼容的 chat/completions 协议为默认实现，作为可选叠加层：
// 未启用或调用失败时返回明确的哨兵错误，由上层决定降级策略，绝不阻断核心业务流程。
package ai

import "context"

// Message 单条对话消息，Role 取 system / user / assistant。
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatRequest 一次对话补全请求。
// Model / MaxTokens 留零、Temperature 为 0 时由 Client 回落配置默认；
// JSONMode 为真时以 JSON 对象格式返回（response_format=json_object），便于结构化解析。
type ChatRequest struct {
	Model       string
	Messages    []Message
	Temperature float64
	MaxTokens   int
	JSONMode    bool
}

// ChatResponse 对话补全结果，Content 为首个 choice 的文本内容。
type ChatResponse struct {
	Content string
	Model   string
}

// Provider 对话补全能力抽象。服务层依赖此接口而非具体实现，
// 既便于测试注入 stub，也为将来接入非 OpenAI 协议留出可插拔边界。
type Provider interface {
	Chat(ctx context.Context, req ChatRequest) (ChatResponse, error)
}
