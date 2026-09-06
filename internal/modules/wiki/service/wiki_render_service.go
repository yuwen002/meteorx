package service

import (
	"context"
	"regexp"
	"strings"

	"meteorx/internal/common/signedurl"
	apperrors "meteorx/internal/pkg/apperrors"
)

// uploadImageSrcRe 匹配 HTML 中 <img> 的 src 值，仅命中引用本系统 /uploads 资源
// （相对 /uploads/xxx 或绝对 http(s)://host/uploads/xxx），文件名限制为单段安全字符。
var uploadImageSrcRe = regexp.MustCompile(`(?i)src="([^"]*?/uploads/([A-Za-z0-9._-]+))"`)

// SetUploadSigner 注入 /uploads 静态资源的签名参数；供模块初始化时由路由层调用。
// 参数为空时表示未启用签名/无上传地址，此时文档图片保持原样输出。
func (s *wikiService) SetUploadSigner(uploadBaseURL, signKey string) {
	s.uploadBaseURL = strings.TrimSuffix(strings.TrimSpace(uploadBaseURL), "/")
	s.uploadSignKey = strings.TrimSpace(signKey)
}

// rewriteImageSrc 将文档 HTML 中引用 /uploads 的图片地址动态替换为短时效签名 URL，
// 使内嵌图片在签名保护下仍可正常访问（链接过期后重新获取文档即可刷新）。
func (s *wikiService) rewriteImageSrc(html string) string {
	if s.uploadBaseURL == "" || s.uploadSignKey == "" || html == "" {
		return html
	}
	return uploadImageSrcRe.ReplaceAllStringFunc(html, func(m string) string {
		src := m[len(`src="`) : len(m)-1]
		urlPath := src
		if i := strings.IndexByte(urlPath, '?'); i >= 0 {
			urlPath = urlPath[:i]
		}

		file := ""
		switch {
		case strings.HasPrefix(urlPath, "/uploads/"):
			file = strings.TrimPrefix(urlPath, "/uploads/")
		case strings.HasPrefix(urlPath, s.uploadBaseURL+"/"):
			file = strings.TrimPrefix(urlPath, s.uploadBaseURL+"/")
		default:
			// 非本系统上传资源（外部图片等）不处理
			return m
		}
		if file == "" || strings.ContainsAny(file, `/\`) {
			return m
		}

		// 原地址已带签名参数的保持原样（避免对旧签名链接重复叠加）
		if strings.Contains(src, "?") {
			return m
		}
		signed := signedurl.Build(s.uploadBaseURL, file, s.uploadSignKey, signedurl.Expires(signedurl.DefaultTTL))
		return `src="` + signed + `"`
	})
}

// RenderPreview 将 Markdown 内容渲染为安全 HTML（供编辑器实时预览，图片地址自动重签）。
func (s *wikiService) RenderPreview(ctx context.Context, content, format string) (string, error) {
	_ = ctx
	if strings.TrimSpace(content) == "" {
		return "", nil
	}
	switch format {
	case "", "markdown":
		return s.rewriteImageSrc(s.markdownSvc.RenderAndSanitize(content)), nil
	default:
		return "", apperrors.ErrBadRequest("仅支持 markdown 格式预览")
	}
}
