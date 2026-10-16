package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// openAIErrorBody 上游错误响应体（尽力解析出 message，用于日志与降级提示）。
type openAIErrorBody struct {
	Error struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error"`
}

// openAICompletionResponse OpenAI chat/completions 成功响应结构（仅取所需字段）。
type openAICompletionResponse struct {
	Model   string `json:"model"`
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

// chatOnce 构造 OpenAI 兼容请求并解析响应。
// 非 2xx 或结构异常返回包装错误；为防御性起见仅透出上游 message 概要，不外泄原始 body。
func (c *Client) chatOnce(ctx context.Context, req ChatRequest) (ChatResponse, error) {
	payload := map[string]any{
		"model":       req.Model,
		"messages":    req.Messages,
		"temperature": req.Temperature,
		"max_tokens":  req.MaxTokens,
	}
	if req.JSONMode {
		payload["response_format"] = map[string]string{"type": "json_object"}
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return ChatResponse{}, fmt.Errorf("ai: marshal request: %w", err)
	}

	url := c.baseURL + "/chat/completions"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return ChatResponse{}, fmt.Errorf("ai: build request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.http.Do(httpReq)
	if err != nil {
		return ChatResponse{}, fmt.Errorf("ai: http request failed: %w", err)
	}
	defer resp.Body.Close()

	// 限制读取体量，防御异常大的上游响应（上限 2MB）。
	data, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return ChatResponse{}, fmt.Errorf("ai: read response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		msg := resp.Status
		var e openAIErrorBody
		if json.Unmarshal(data, &e) == nil && e.Error.Message != "" {
			msg = e.Error.Message
		}
		return ChatResponse{}, fmt.Errorf("ai: upstream %d: %s", resp.StatusCode, msg)
	}

	var cr openAICompletionResponse
	if err := json.Unmarshal(data, &cr); err != nil {
		return ChatResponse{}, fmt.Errorf("ai: decode completion: %w", err)
	}
	if len(cr.Choices) == 0 {
		return ChatResponse{}, fmt.Errorf("ai: empty completion choices")
	}
	return ChatResponse{Content: cr.Choices[0].Message.Content, Model: cr.Model}, nil
}
