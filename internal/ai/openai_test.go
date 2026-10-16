package ai

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"meteorx/internal/config"
)

// newTestClient 构造指向本地测试服务端的可用 Client。
func newTestClient(baseURL string) *Client {
	return New(config.AIConfig{
		Enabled:   true,
		BaseURL:   baseURL,
		APIKey:    "test-key",
		Model:     "gpt-4o-mini",
		MaxTokens: 128,
		Timeout:   5 * time.Second,
	})
}

func TestClient_NotUsableReturnsDisabled(t *testing.T) {
	// 未启用：Chat 应直接返回 ErrDisabled，不发起任何 HTTP 调用
	c := New(config.AIConfig{Enabled: false, APIKey: "k"})
	if c.Usable() {
		t.Fatal("expected not usable when disabled")
	}
	_, err := c.Chat(context.Background(), ChatRequest{Messages: []Message{{Role: "user", Content: "hi"}}})
	if !errors.Is(err, ErrDisabled) {
		t.Fatalf("expected ErrDisabled, got %v", err)
	}

	// 启用但缺 key：同样不可用
	c2 := New(config.AIConfig{Enabled: true, APIKey: ""})
	if c2.Usable() {
		t.Fatal("expected not usable when api key empty")
	}
}

func TestClient_ChatSuccess(t *testing.T) {
	var gotAuth string
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"model":"gpt-4o-mini","choices":[{"message":{"role":"assistant","content":"{\"ok\":true}"}}]}`))
	}))
	defer srv.Close()

	c := newTestClient(srv.URL)
	resp, err := c.Chat(context.Background(), ChatRequest{
		Messages: []Message{{Role: "user", Content: "hi"}},
		JSONMode: true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Content != `{"ok":true}` {
		t.Fatalf("unexpected content: %q", resp.Content)
	}
	if gotAuth != "Bearer test-key" {
		t.Fatalf("unexpected auth header: %q", gotAuth)
	}
	// JSONMode 应透传 response_format=json_object
	rf, ok := gotBody["response_format"].(map[string]any)
	if !ok || rf["type"] != "json_object" {
		t.Fatalf("expected response_format json_object, got %#v", gotBody["response_format"])
	}
	// 零值 Model/MaxTokens 回落配置默认
	if gotBody["model"] != "gpt-4o-mini" {
		t.Fatalf("expected default model, got %#v", gotBody["model"])
	}
}

func TestClient_UpstreamError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"error":{"message":"rate limited","type":"rate_limit"}}`))
	}))
	defer srv.Close()

	c := newTestClient(srv.URL)
	_, err := c.Chat(context.Background(), ChatRequest{Messages: []Message{{Role: "user", Content: "hi"}}})
	if err == nil {
		t.Fatal("expected error on non-2xx")
	}
	if errors.Is(err, ErrDisabled) {
		t.Fatalf("upstream error should not be ErrDisabled: %v", err)
	}
}

func TestClient_EmptyChoices(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"model":"m","choices":[]}`))
	}))
	defer srv.Close()

	c := newTestClient(srv.URL)
	if _, err := c.Chat(context.Background(), ChatRequest{Messages: []Message{{Role: "user", Content: "hi"}}}); err == nil {
		t.Fatal("expected error on empty choices")
	}
}

func TestAIConfigDefaults(t *testing.T) {
	c := config.AIConfig{}
	if c.GetBaseURL() != "https://api.openai.com/v1" {
		t.Fatalf("unexpected default base url: %s", c.GetBaseURL())
	}
	if c.GetModel() != "gpt-4o-mini" {
		t.Fatalf("unexpected default model: %s", c.GetModel())
	}
	if c.GetMaxTokens() != 512 {
		t.Fatalf("unexpected default max tokens: %d", c.GetMaxTokens())
	}
	if c.GetTimeout() != 20*time.Second {
		t.Fatalf("unexpected default timeout: %v", c.GetTimeout())
	}
	// 温度为 0 是合法确定性取值，不应被默认覆盖；仅负值回落 0.2
	if c.GetTemperature() != 0 {
		t.Fatalf("temperature 0 should stay 0, got %v", c.GetTemperature())
	}
	neg := config.AIConfig{Temperature: -1}
	if neg.GetTemperature() != 0.2 {
		t.Fatalf("negative temperature should fall back to 0.2, got %v", neg.GetTemperature())
	}
}
