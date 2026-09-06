package service

import (
	"strings"
	"testing"
)

func TestRewriteImageSrcDisabled(t *testing.T) {
	svc := &wikiService{}
	html := `<p><img src="/uploads/abc123.png" alt="x"></p>`
	if out := svc.rewriteImageSrc(html); out != html {
		t.Fatalf("签名参数未配置时不应改写内容: %s", out)
	}
}

func TestRewriteImageSrcRelative(t *testing.T) {
	svc := &wikiService{}
	svc.SetUploadSigner("http://host:8081/uploads", "test-secret")

	out := svc.rewriteImageSrc(`<p><img src="/uploads/abc123.png" alt="x"></p>`)
	if !strings.Contains(out, `src="http://host:8081/uploads/abc123.png?e=`) {
		t.Fatalf("相对上传地址未签发签名: %s", out)
	}
}

func TestRewriteImageSrcIgnoresSignedAndExternal(t *testing.T) {
	svc := &wikiService{}
	svc.SetUploadSigner("http://host:8081/uploads", "test-secret")

	html := `<p>` +
		`<img src="https://other.example.com/uploads/x.png" alt="ext">` +
		`<img src="/uploads/y.png?e=1700000000&s=deadbeef" alt="old">` +
		`</p>`
	out := svc.rewriteImageSrc(html)
	if !strings.Contains(out, `https://other.example.com/uploads/x.png"`) {
		t.Fatalf("外部图片不应被改写: %s", out)
	}
	if !strings.Contains(out, `/uploads/y.png?e=1700000000&s=deadbeef"`) {
		t.Fatalf("已携带签名参数的图片不应重复改写: %s", out)
	}
}

func TestRewriteImageSrcAbsoluteBase(t *testing.T) {
	svc := &wikiService{}
	svc.SetUploadSigner("http://host:8081/uploads", "test-secret")

	out := svc.rewriteImageSrc(`<p><img src="http://host:8081/uploads/z.png" alt="z"></p>`)
	if !strings.Contains(out, `src="http://host:8081/uploads/z.png?e=`) {
		t.Fatalf("同源绝对上传地址未签发签名: %s", out)
	}
}

func TestRenderPreviewSanitizesAndSigns(t *testing.T) {
	svc := &wikiService{markdownSvc: NewMarkdownService()}
	svc.SetUploadSigner("http://host:8081/uploads", "test-secret")

	html, err := svc.RenderPreview(
		nil,
		"<script>alert(1)</script>\n<textarea onclick='x()'>hi</textarea>\n\n![](/uploads/evil.png)",
		"markdown",
	)
	if err != nil {
		t.Fatalf("RenderPreview error: %v", err)
	}
	if strings.Contains(html, "<script") {
		t.Fatalf("预览结果未清除脚本: %s", html)
	}
	if strings.Contains(html, "<textarea") || strings.Contains(html, "onclick") {
		t.Fatalf("预览结果仍含危险标签或事件属性: %s", html)
	}
	if !strings.Contains(html, "http://host:8081/uploads/evil.png?e=") {
		t.Fatalf("预览中的图片地址未重新签发: %s", html)
	}
}

func TestRenderPreviewRejectsUnknownFormat(t *testing.T) {
	svc := &wikiService{markdownSvc: NewMarkdownService()}
	if _, err := svc.RenderPreview(nil, "# hi", "html"); err == nil {
		t.Fatal("未知格式应返回错误")
	}
}
