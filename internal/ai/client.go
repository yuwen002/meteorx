package ai

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"meteorx/internal/config"
)

// ErrDisabled 表示 AI 能力未启用或未配置可用凭据（缺 API Key），
// 调用方据此走降级路径，而非向用户抛出真实故障。
var ErrDisabled = errors.New("ai: provider disabled or not configured")

// Client 是 Provider 的默认实现，持有 OpenAI 兼容端点配置与 HTTP 客户端。
type Client struct {
	enabled     bool
	baseURL     string
	apiKey      string
	model       string
	temperature float64
	maxTokens   int

	http *http.Client
}

// New 依据配置构造 Client，零值字段回落内置默认。
// 未启用时仍返回非 nil 的 Client，其 Usable() 为 false、Chat() 直接返回 ErrDisabled，
// 使上层无需处理 nil 依赖即可安全降级。
func New(cfg config.AIConfig) *Client {
	timeout := cfg.GetTimeout()
	return &Client{
		enabled:     cfg.Enabled,
		baseURL:     strings.TrimRight(cfg.GetBaseURL(), "/"),
		apiKey:      strings.TrimSpace(cfg.APIKey),
		model:       cfg.GetModel(),
		temperature: cfg.GetTemperature(),
		maxTokens:   cfg.GetMaxTokens(),
		http:        &http.Client{Timeout: timeout},
	}
}

// Usable 是否可发起真实调用：已启用且配置了 API Key。
func (c *Client) Usable() bool {
	return c.enabled && c.apiKey != ""
}

// Enabled 返回开关状态，供能力探测接口透出。
func (c *Client) Enabled() bool {
	return c.enabled
}

// Model 返回当前默认模型名。
func (c *Client) Model() string {
	return c.model
}

// Chat 发起一次对话补全；未启用/未配置时直接返回 ErrDisabled 触发降级。
// 请求未指定 Model / MaxTokens（零值）或 Temperature（0）时回落配置默认。
func (c *Client) Chat(ctx context.Context, req ChatRequest) (ChatResponse, error) {
	if !c.Usable() {
		return ChatResponse{}, ErrDisabled
	}
	if req.Model == "" {
		req.Model = c.model
	}
	if req.MaxTokens <= 0 {
		req.MaxTokens = c.maxTokens
	}
	if req.Temperature == 0 {
		req.Temperature = c.temperature
	}
	return c.chatOnce(ctx, req)
}

// compile-time assertion：Client 实现 Provider 接口。
var _ Provider = (*Client)(nil)
