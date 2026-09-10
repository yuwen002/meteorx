package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// WebhookKind webhook 类型
type WebhookKind string

const (
	WebhookGeneric   WebhookKind = "generic"    // 通用 Webhook
	WebhookDingTalk  WebhookKind = "dingtalk"   // 钉钉
	WebhookWeChat    WebhookKind = "wechat"     // 企业微信
	WebhookFeishu    WebhookKind = "feishu"     // 飞书
)

// WebhookNotifier Webhook 通知渠道
type WebhookNotifier struct {
	client  *http.Client
	kind    WebhookKind
	url     string
	secret  string // 签名密钥（部分渠道需要）
}

// WebhookOption Webhook 配置选项
type WebhookOption struct {
	Kind    WebhookKind
	URL     string
	Secret  string
	Timeout time.Duration
}

// NewWebhookNotifier 创建 Webhook 通知渠道
func NewWebhookNotifier(opt WebhookOption) *WebhookNotifier {
	if opt.Timeout == 0 {
		opt.Timeout = 10 * time.Second
	}
	return &WebhookNotifier{
		client: &http.Client{Timeout: opt.Timeout},
		kind:   opt.Kind,
		url:    opt.URL,
		secret: opt.Secret,
	}
}

func (n *WebhookNotifier) Type() ChannelType { return ChannelWebhook }

func (n *WebhookNotifier) Name() string { return "webhook-" + string(n.kind) }

func (n *WebhookNotifier) Send(ctx context.Context, msg *Message) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	payload, err := n.buildPayload(msg)
	if err != nil {
		return fmt.Errorf("build webhook payload failed: %w", err)
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal payload failed: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, n.url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request failed: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := n.client.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("webhook responded %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// buildPayload 根据 webhook 类型构建不同格式的消息体
func (n *WebhookNotifier) buildPayload(msg *Message) (interface{}, error) {
	switch n.kind {
	case WebhookDingTalk:
		return n.buildDingTalkPayload(msg), nil
	case WebhookWeChat:
		return n.buildWeChatPayload(msg), nil
	case WebhookFeishu:
		return n.buildFeishuPayload(msg), nil
	default:
		return n.buildGenericPayload(msg), nil
	}
}

// 通用 Webhook 格式
type genericWebhookPayload struct {
	Title    string                 `json:"title"`
	Content  string                 `json:"content"`
	Priority string                 `json:"priority"`
	Time     int64                  `json:"time"`
	Extra    map[string]interface{} `json:"extra,omitempty"`
}

func (n *WebhookNotifier) buildGenericPayload(msg *Message) interface{} {
	return &genericWebhookPayload{
		Title:    msg.Title,
		Content:  msg.Content,
		Priority: msg.Priority.String(),
		Time:     msg.CreatedAt.Unix(),
		Extra:    msg.Extra,
	}
}

// 钉钉机器人消息格式
type dingTalkPayload struct {
	MsgType string          `json:"msgtype"`
	Text    *dingTalkText   `json:"text,omitempty"`
	Markdown *dingTalkMarkdown `json:"markdown,omitempty"`
}

type dingTalkText struct {
	Content string `json:"content"`
}

type dingTalkMarkdown struct {
	Title string `json:"title"`
	Text  string `json:"text"`
}

func (n *WebhookNotifier) buildDingTalkPayload(msg *Message) interface{} {
	text := fmt.Sprintf("## %s\n\n%s\n\n---\n*优先级: %s*", msg.Title, msg.Content, msg.Priority.String())
	return &dingTalkPayload{
		MsgType: "markdown",
		Markdown: &dingTalkMarkdown{
			Title: msg.Title,
			Text:  text,
		},
	}
}

// 企业微信机器人格式
type weChatPayload struct {
	MsgType  string          `json:"msgtype"`
	Markdown *weChatMarkdown `json:"markdown"`
}

type weChatMarkdown struct {
	Content string `json:"content"`
}

func (n *WebhookNotifier) buildWeChatPayload(msg *Message) interface{} {
	text := fmt.Sprintf("# %s\n%s\n\n> 优先级: %s", msg.Title, msg.Content, msg.Priority.String())
	return &weChatPayload{
		MsgType: "markdown",
		Markdown: &weChatMarkdown{
			Content: text,
		},
	}
}

// 飞书机器人格式
type feishuPayload struct {
	MsgType string          `json:"msg_type"`
	Content *feishuContent  `json:"content"`
}

type feishuContent struct {
	Post *feishuPost `json:"post,omitempty"`
}

type feishuPost struct {
	ZhCN *feishuZhCN `json:"zh_cn,omitempty"`
}

type feishuZhCN struct {
	Title   string          `json:"title"`
	Content [][]feishuBlock `json:"content"`
}

type feishuBlock struct {
	Tag    string `json:"tag"`
	Text   string `json:"text,omitempty"`
	Href   string `json:"href,omitempty"`
	UnEscape bool  `json:"un_escape,omitempty"`
}

func (n *WebhookNotifier) buildFeishuPayload(msg *Message) interface{} {
	content := [][]feishuBlock{
		{{Tag: "text", Text: msg.Content, UnEscape: true}},
		{{Tag: "text", Text: "优先级: " + msg.Priority.String()}},
	}
	return &feishuPayload{
		MsgType: "post",
		Content: &feishuContent{
			Post: &feishuPost{
				ZhCN: &feishuZhCN{
					Title:   msg.Title,
					Content: content,
				},
			},
		},
	}
}

// IsWebhookConfigured 检查 Webhook 是否已配置
func IsWebhookConfigured(url string) bool {
	return url != ""
}