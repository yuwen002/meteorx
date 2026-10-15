package middleware

import (
	"encoding/json"
	"io"
	"net/http"

	"meteorx/pkg/logger"
)

// cspReportMaxBody 限制单次上报请求体大小，防止恶意/异常大 body 占用内存。
// 浏览器生成的 CSP 违规报告体积很小（通常 <2KB），8KB 已绰绰有余。
const cspReportMaxBody = 8 << 10

// cspReportPayload 是浏览器以 POST 发往 report-uri 的 CSP 违规报告结构。
// 顶层固定为 {"csp-report": {...}}，字段名由 CSP 规范约定，非本项目自定义。
type cspReportPayload struct {
	Report struct {
		DocumentURI        string `json:"document-uri"`
		Referrer           string `json:"referrer"`
		BlockedURI         string `json:"blocked-uri"`
		ViolatedDirective  string `json:"violated-directive"`
		EffectiveDirective string `json:"effective-directive"`
		StatusCode         int    `json:"status-code"`
		Sample             string `json:"sample"`
	} `json:"csp-report"`
}

// CSPReport 接收浏览器上报的 CSP 违规报告，仅落服务端日志、始终返回 204。
//
// 用途：配合 nginx 中的 Content-Security-Policy-Report-Only，把「观察期」内命中的
// 违规收集到服务端日志，便于后续把策略从 Report-Only 收紧为正式 CSP 前评估影响面。
//
// 设计取舍：
//   - 无鉴权：CSP 报告由浏览器直接发出、无法携带 Authorization，因此注册在公开分组，
//     并依赖公开分组已有的 IP 限流防止被刷爆日志。
//   - 静默失败：任何解析/读取错误都照常返回 204，绝不把错误回吐给浏览器（避免信息泄露，
//     也不影响页面渲染）。
//   - 仅记录关键字段：不落 original-policy（每次重复且体积大），sample 截断后记录。
func CSPReport() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 无论后续解析结果如何，都只回 204：浏览器会忽略响应体。
		writeStatus := func() { w.WriteHeader(http.StatusNoContent) }

		if r.Body == nil {
			writeStatus()
			return
		}
		defer func() { _ = r.Body.Close() }()

		var payload cspReportPayload
		dec := json.NewDecoder(io.LimitReader(r.Body, cspReportMaxBody))
		if err := dec.Decode(&payload); err != nil {
			// 非法/截断的 JSON：静默丢弃，不记录噪声
			writeStatus()
			return
		}

		rep := payload.Report
		// 无 blocked-uri 且无 violated-directive 视为无效上报，跳过
		if rep.BlockedURI == "" && rep.ViolatedDirective == "" {
			writeStatus()
			return
		}

		logger.Warnf("[csp-violation] blocked=%q directive=%q effective=%q document=%q referrer=%q status=%d sample=%q",
			rep.BlockedURI, rep.ViolatedDirective, rep.EffectiveDirective,
			rep.DocumentURI, rep.Referrer, rep.StatusCode, cspPreview(rep.Sample, 120))

		writeStatus()
	}
}

// cspPreview 截断字符串到最多 n 个字节（按字节安全截断，多字节字符可能被切断但不影响日志）。
func cspPreview(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
