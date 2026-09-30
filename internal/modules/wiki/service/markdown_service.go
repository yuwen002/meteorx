package service

import (
	"html"
	"regexp"
	"strings"
)

// MarkdownService 定义 Markdown 渲染与 HTML 净化的能力接口。
type MarkdownService interface {
	Render(markdown string) string
	SanitizeHTML(html string) string
	RenderAndSanitize(markdown string) string
}

// markdownService MarkdownService 的默认实现，基于正则做轻量级转换与清洗。
type markdownService struct{}

// NewMarkdownService 创建 Markdown 渲染服务实例。
func NewMarkdownService() MarkdownService {
	return &markdownService{}
}

// Render 将 Markdown 文本渲染为 HTML（不做净化）。
func (s *markdownService) Render(markdown string) string {
	html := s.basicMarkdownToHTML(markdown)
	return html
}

// SanitizeHTML 移除脚本标签、事件处理器、危险协议及危险标签，防止 XSS 注入。
func (s *markdownService) SanitizeHTML(inputHTML string) string {
	sanitized := inputHTML

	// Remove script tags (content included, may span multiple lines)
	scriptRegex := regexp.MustCompile(`(?is)<script[^>]*>.*?</script>`)
	sanitized = scriptRegex.ReplaceAllString(sanitized, "")

	// Remove event handlers
	eventRegex := regexp.MustCompile(`(?i)\s+on\w+\s*=\s*"[^"]*"|\s+on\w+\s*=\s*'[^']*'|\s+on\w+\s*=\s*\S+`)
	sanitized = eventRegex.ReplaceAllString(sanitized, "")

	// Remove javascript: protocol
	jsRegex := regexp.MustCompile(`(?i)javascript\s*:`)
	sanitized = jsRegex.ReplaceAllString(sanitized, "")

	// Remove dangerous tags. RE2 does not support backreferences, so pair-content
	// removal is impossible in a single regexp; strip opening/closing/self-closing
	// tags instead (leftover text is inert).
	dangerousTagsRegex := regexp.MustCompile(`(?i)</?(?:iframe|object|embed|form|input|textarea|button|link|style|base|meta|applet|frame|frameset|ilayer|layer|event|select|isindex)(?:\s[^>]*)?/?>`)
	sanitized = dangerousTagsRegex.ReplaceAllString(sanitized, "")

	// Remove data: protocol URLs
	dataURLRegex := regexp.MustCompile(`(?i)data\s*:`)
	sanitized = dataURLRegex.ReplaceAllString(sanitized, "")

	// Remove vbscript: protocol
	vbscriptRegex := regexp.MustCompile(`(?i)vbscript\s*:`)
	sanitized = vbscriptRegex.ReplaceAllString(sanitized, "")

	return sanitized
}

// RenderAndSanitize 先渲染 Markdown 再净化 HTML，返回安全可直接展示的片段。
func (s *markdownService) RenderAndSanitize(markdown string) string {
	rendered := s.Render(markdown)
	return s.SanitizeHTML(rendered)
}

func (s *markdownService) basicMarkdownToHTML(markdown string) string {
	if markdown == "" {
		return ""
	}

	result := markdown

	// Escape HTML first
	result = html.EscapeString(result)

	// Headers
	headerRegex := regexp.MustCompile(`^######\s+(.+)$`)
	result = headerRegex.ReplaceAllString(result, "<h6>$1</h6>")

	headerRegex = regexp.MustCompile(`^#####\s+(.+)$`)
	result = headerRegex.ReplaceAllString(result, "<h5>$1</h5>")

	headerRegex = regexp.MustCompile(`^####\s+(.+)$`)
	result = headerRegex.ReplaceAllString(result, "<h4>$1</h4>")

	headerRegex = regexp.MustCompile(`^###\s+(.+)$`)
	result = headerRegex.ReplaceAllString(result, "<h3>$1</h3>")

	headerRegex = regexp.MustCompile(`^##\s+(.+)$`)
	result = headerRegex.ReplaceAllString(result, "<h2>$1</h2>")

	headerRegex = regexp.MustCompile(`^#\s+(.+)$`)
	result = headerRegex.ReplaceAllString(result, "<h1>$1</h1>")

	// Bold
	boldRegex := regexp.MustCompile(`\*\*(.+?)\*\*`)
	result = boldRegex.ReplaceAllString(result, "<strong>$1</strong>")

	// Italic
	italicRegex := regexp.MustCompile(`\*(.+?)\*`)
	result = italicRegex.ReplaceAllString(result, "<em>$1</em>")

	// Inline code
	codeRegex := regexp.MustCompile("`([^`]+)`")
	result = codeRegex.ReplaceAllString(result, "<code>$1</code>")

	// Code blocks
	codeBlockRegex := regexp.MustCompile("```\\w*\\n([\\s\\S]*?)```")
	result = codeBlockRegex.ReplaceAllString(result, "<pre><code>$1</code></pre>")

	// Links
	linkRegex := regexp.MustCompile(`\[([^\]]+)\]\(([^)]+)\)`)
	result = linkRegex.ReplaceAllString(result, `<a href="$2">$1</a>`)

	// Images
	imageRegex := regexp.MustCompile(`!\[([^\]]*)\]\(([^)]+)\)`)
	result = imageRegex.ReplaceAllString(result, `<img src="$2" alt="$1">`)

	// Unordered lists
	listRegex := regexp.MustCompile(`^[\-\*]\s+(.+)$`)
	result = listRegex.ReplaceAllString(result, "<li>$1</li>")

	// Wrap list items
	// This is a simplified approach - real implementation would need more sophisticated parsing
	if strings.Contains(result, "<li>") {
		result = "<ul>" + result + "</ul>"
	}

	// Line breaks
	result = strings.ReplaceAll(result, "\n", "<br>")

	// Replace the auto-generated <br> in the wrapping
	result = strings.ReplaceAll(result, "<ul><br>", "<ul>")
	result = strings.ReplaceAll(result, "<br></ul>", "</ul>")

	return result
}
